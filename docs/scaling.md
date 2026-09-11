# Scaling Compass

A Drupal request can make more than a million PHP function calls. This document
records what the pipeline costs at that size, where it stops, and the order in
which to change it.

The short version: **one wire record per function call does not reach a million
calls**, and every limit below follows from that. The pipeline is fast enough
for the traces it is willing to carry today because it refuses to carry big
ones — the sidecar's 10,000-call cap discards the tail of a large request and
reports it as a number.

## What a million calls costs

Measured per stage on linux/amd64, 8 cores, Go 1.26, with a synthetic trace of
400 distinct symbols called repeatedly over a one second request. These are
microbenchmarks of each stage rather than an end-to-end run, and the eBPF
programs are not included — see [Not yet measured](#not-yet-measured).

| Stage | Cost per call | At 1,000,000 calls |
| --- | --- | --- |
| Ring-buffer record | 232 B | 232 MB through a 1 MiB ring (4,369 records) |
| Decode, `ingest.DecodeExact` → `binary.Read` | 1,624 ns, 2 allocs | 1.6 s |
| Handler, mutex + `go-cache` get + add + touch | 203 ns, 1 alloc | 0.2 s |
| Retention, `functioncalls.Limiter` | capped at 10,000 | 99% dropped and counted |
| JSON encode | 183 ns | 136 MiB on a single NDJSON line |
| Transport | — | over the CLI's 10 MiB line limit; trace dropped whole |
| JSON decode, CLI | 1,592 ns, 2 allocs | 1.6 s, 360 MB, on the stream goroutine |
| `segmented.Unmarshal`, TUI | 262 ns, 2 allocs | 262 ms, 116 MB — **per keystroke** while filtering |

These are the costs the plan starts from: the rows for items marked done below
have changed since, and each of those items records its own before and after.

Two of them are ceilings rather than costs.

**Ingest.** Decode and handle together are 1,827 ns per event, so one collector
drains about 547,000 events per second. A million-call request which completes
in a second produces events faster than that, the ring fills, and
`compass_sidecar_ringbuf_reserve_failures_total` climbs: data is then lost by
timing rather than by policy, which is the worst of the two.

**Display.** `segmented.Unmarshal` runs from `functionsSetRows`, which runs on
every filter keystroke and every resize. At the current 10,000-call cap that is
2.4 ms per keystroke, which is survivable; at 100,000 it is 27 ms and at a
million it is 262 ms, which is not.

## Where this stands

Items 1, 8 and 9 are implemented; everything else is the plan. Each item says
what it measured when it landed, or what it is expected to buy if it has not.

## The plan

Ordered by what each change buys against what it risks. The first three are
contained and change no protocol; the fourth is what actually lifts the ceiling.

### 1. Decode the hot record by hand — done

`ingest.DecodeExact` uses `binary.Read`, which walks the struct with reflection
for every event. Reading the same layout at explicit offsets, which is what
each runtime's `decodeFunctionEvent` now does, measured **1,556 ns → 17.5 ns
per record, and no allocations** — 89 times faster, from about 615,000 records
per second to more than fifty million.

The function record is the only one which arrives more than a handful of times
per request, so it is the only one which needs this: request init, request
shutdown and the Drupal cache events keep the reflection path, where a clear
error message is worth more than nanoseconds.

The layouts are generated from C by `bpf2go`, so hand-written offsets could
drift from the struct they decode. Each runtime's decoder is therefore checked
against `binary.Read` over randomised samples, which fails on any offset that
does not agree with the generated layout.

### 2. Stop touching the cache on every event

`Handler.touch` calls `go-cache`'s `Set` for every function event to push out
the request's expiry. That takes a second lock, on top of the handler's own
mutex, on the hottest path in the process: **203 ns → 51 ns** without it.

Expiry only needs to be pushed out while a request is in flight, which the
lifecycle events already mark. Either touch on those alone, or hold the state in
a plain map with a `lastSeen` field swept by a ticker.

### 3. Intern function names

`functioncalls.Limiter.Add` converts the name to a string for every retained
call: one allocation of about 59 bytes each. A Drupal request calls a few
hundred distinct symbols, most of them thousands of times, so a per-collector
intern map makes the steady state allocation-free.

### 4. Aggregate on ingest instead of retaining calls

This is the change which removes the ceiling.

Key events by `(name, offset bucket)` and accumulate calls, total elapsed,
longest call and peak memory: **42 ns per event, no allocations, and an output
bounded by distinct symbols times segments** — 400 buckets for a million calls,
against the hundred megabytes of records the same request retains today.

The TUI already computes exactly this, in `segmented.Unmarshal`, at display
time. Moving it to ingest means the aggregate is what travels, and neither the
10,000-call cap nor the 10 MiB line limit decides what a reader can see any
more.

Retention becomes a two-part policy rather than a cliff:

- the first `MAX_FUNCTION_CALLS` calls are retained exactly, so the call
  sequence a normal request shows today is unchanged, and
- everything after that is **aggregated rather than dropped**, so a large
  request reports where its time went instead of reporting how much of itself
  it threw away.

On the wire this is a `spans` field beside `functionCalls`, so a million-call
trace is a few hundred kilobytes rather than 136 MiB. Version it: a CLI which
predates the field still reads the calls, and a CLI which has it can show the
aggregate for the part which was never retained.

### 5. Shard the rings and the handler state

A `BPF_MAP_TYPE_RINGBUF` is a single reservation domain. Every php-fpm worker on
every CPU competes for it, and one `Handler.mu` then serialises the function
reader against the Drupal reader in user space.

Above roughly a million events per second both need splitting: an
`ARRAY_OF_MAPS` of ring buffers keyed by `bpf_get_smp_processor_id()`, one
reader goroutine per ring, and handler state sharded by request-id hash. Ring
memory is per-shard, so the sidecar's and the daemon's memory accounting has to
follow — see [Fleet](#fleet).

### 6. Aggregate in the kernel

The step past 4. A per-request BPF hash keyed by `(request cookie, symbol id)`,
accumulating count and elapsed time, flushed to user space at
`fpm_request_shutdown`. A million ring records become one flush, and the ring
stops being a bottleneck at all.

It needs a symbol identity scheme: hash the name in the probe, emit the
101-byte name once when the hash is first seen, and let every later call carry
eight bytes instead of 232.

### 7. The probe is the floor

Everything above is about what happens after a probe fires. At a million calls
the probes themselves dominate: a USDT probe is a uprobe, so each hit traps into
the kernel, and the per-hit cost is of the order of a microsecond on x86_64.
A million of them is therefore of the order of a second of added request
latency, whatever the sidecar does with the events.

Three ways out, all in the extension:

- **Threshold.** `compass.function_threshold` is the front line and already
  exists. It deserves documented guidance by call volume rather than a single
  default.
- **In-process aggregation.** A hot function costs a struct update, tens of
  nanoseconds, instead of a trap. The extension emits one probe at request
  shutdown carrying its table. This is the only route to seeing all million
  calls.
- **Sampling.** Once a symbol has been seen *k* times, fire for one call in *n*
  and extrapolate the count, which bounds the tail on pathological requests.

### 8. Segment once per trace, not once per keystroke — done

`functionsSetRows` re-ran `segmented.Unmarshal` and re-sorted its output on
every filter keystroke and every resize. The spans depend only on the trace, so
they are now computed when a trace is opened and reused while it stays open,
and the filter narrows the cached spans. Rows still rebuild on resize, because
they carry their own widths.

A keystroke on a trace of 10,000 calls went from 2.7 ms to 1.7 ms, and on
100,000 calls from 22.1 ms to 15.3 ms. Those are against the aggregation in
item 9; before that the same rebuilds were about 4.2 ms and 42 ms.

What is left in the number is row building, which is still O(spans): 4,000
spans is 15 ms and about 25 allocations a row, for the thirty or so rows which
are on screen. Building rows for the visible window rather than for every span
is the next step, and it is a change to the datatable rather than to this
page.

### 9. Key spans without formatting a string — done

`segmented.Unmarshal` built its map key with `fmt.Sprintf("%s-%d-%d", …)`: two
allocations per function call, and 116 MB of the million-call cost. A struct
key of the same three fields is allocation-free and compares the same way.

| Calls | Before | After |
| --- | ---: | ---: |
| 10,000 | 2.4 ms, 20,030 allocs | 0.9 ms, 30 allocs |
| 100,000 | 27.1 ms, 200,070 allocs | 7.5 ms, 37 allocs |
| 1,000,000 | 261.8 ms, 116 MB, 2,000,313 allocs | 83.0 ms, 21.6 MB, 272 allocs |

### 10. Bound the CLI by bytes

`--max-traces` bounds the retained traces by count, so what the TUI holds is
unbounded in bytes: 500 traces times whatever each carries. Bound it by bytes as
well, and keep only the aggregate for traces which have aged out of the recent
window.

### Fleet

Once a single collector keeps up, the remaining limits are about how many of
them there are.

- **Ring memory per target.** The daemon allows 50 concurrent targets, each
  with a 1 MiB event ring and a 4 MiB Drupal ring: 250 MB of locked memory
  before the sharding in item 5 multiplies it. Make the ring sizes configurable
  and account for them in `COMPASS_DAEMON_MAX_TARGETS`.
- **Fan-out buffers.** The broadcaster and the router give each subscriber a
  ten-*trace* buffer and drop on full. That is fine at a megabyte a
  trace, and 1.4 GB per slow subscriber at 136 MB. Bound the buffer by bytes too, and prefer shedding the
  oldest trace rather than the arriving one: for a performance tool the newest
  trace is usually the interesting one.
- **Fan-in.** One collector per target and one CLI per node means a human picks
  the pod. A merge layer above the router — subscribing to many daemons and
  merging by request id, with per-source drop accounting — is additive to what
  the router already does per target.
- **Sampling across the fleet.** A request-level gate, keyed on a hash of
  `X-Request-ID` or driven by a BPF map, traces one percent of traffic on every
  pod instead of all of the traffic on one.

### Metrics the plan needs

The existing counters describe the losses of the current design. The new
bottlenecks need their own, or a regression will be invisible: events read per
second per ring, drain lag behind the producer, aggregate bucket count per
trace, and locked memory per target.

## Not yet measured

- **Uprobe cost per probe hit**, which item 7 turns on. The order-of-magnitude
  figure above is from the general behaviour of uprobes, not from this
  extension: it needs measuring against a real workload before anything is
  traded against it.
- **Ring-buffer overflow behaviour** under a sustained million events per
  second, including whether reserve failures cluster or spread.
- **End-to-end latency** from probe to TUI. Every number here is a stage in
  isolation.
