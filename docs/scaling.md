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

Items 1, 2, 3, 4, 8 and 9 are implemented; 5, 6, 7 and 10, and the fleet work,
are the plan. Each item says what it measured when it landed, or what it is
expected to buy if it has not.

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

### 2. Stop touching the cache on every event — done

`Handler.touch` called `go-cache`'s `Set` for every function event to push out
the request's expiry: a second lock and a second map write, on top of the
handler's own lock, on the hottest path in the process.

`pkg/tracer/requests` replaced it. Keeping a request alive is a field write on
the entry the lookup already found, and expiry still runs from the last event
a request produced, so a long-running Drush command is kept for as long as it
keeps calling functions. Sweeping moved from a janitor goroutine to the path
which creates requests, at most once per expiry period, and runs on the
probes' monotonic clock rather than the system one.

    BenchmarkHandleFunction    307.3ns -> 174.6ns per function event

### 3. Stop allocating a string per call — done

Two allocations were left on the path a function event takes, and both were
for strings made out of bytes the event already carried.

The request id was one: the storage was keyed on a string, so every event
converted a 101-byte field to find a key it had already used a thousand times.
The storage is now keyed on the field itself, which is comparable and so is a
map key as it stands.

The function name was the other. The aggregator holds the names it has made
and reuses them, bounded at 8,192 per collector because the names come from
the application rather than from us. The lookup is a map index on a string
conversion of the bytes, which the compiler does without allocating — which is
also what lets item 4 build its span key without allocating.

    BenchmarkHandleFunction    174.6ns, 2 allocs -> 151.5ns, 0 allocs

### 4. Aggregate on ingest instead of retaining calls — done

This is the change which removed the ceiling.

A trace no longer carries function calls. It carries **spans**: the calls of
one function within one slice of the request, with how many there were, the
longest of them, what they cost altogether, where the earliest of them started
and the most memory any of them reported. Aggregating as the events arrive
measured **61ns per call and no allocations**, and the output is bounded by
the distinct functions a request calls times the slices of it they ran in,
rather than by how many times it called them.

The display never showed anything else. `segmented.Unmarshal` computed exactly
this aggregate at render time, from calls which had been carried the whole way
for the purpose, so moving it to ingest took a step out of the pipeline rather
than adding one: `pkg/trace/segmented` and `pkg/trace/count` are gone, and the
Functions page reads `trace.Spans` directly.

One rule, applied to every call:

- a span is `(function, offset / bucket)`, where the bucket is a fixed ten
  milliseconds — fixed because the span a call belongs to has to be chosen
  when its event arrives, and how long the request ran for is not known until
  it ends, so a share of the request is not available to bucket by;
- a trace carries at most `MAX_SPANS` of them, and a call which needs a span
  the trace has no room for is counted rather than placed.

The count of calls a request made stays exact whether or not every call found
a span, and peak memory still takes every call into account. There is no
second regime: no prefix of the calls is treated differently from the rest.

On the wire `functionCalls` is replaced by `spans`, which is a breaking change
to the stream: a CLI older than the sidecar it connects to will show traces
with no functions in them. The two are released together, tagged by version,
so they are upgraded together.

`COMPASS_SIDECAR_MAX_FUNCTION_CALLS` and `COMPASS_DAEMON_MAX_FUNCTION_CALLS`
became `..._MAX_SPANS`. The old names are still read, so a deployment keeps
the bound it configured, and the sidecar warns once at startup when it takes
one.

    BenchmarkHandleFunction    151.5ns, 0 allocs -> 138.2ns, 0 allocs, 0 B

The whole path a function event takes is now 138.2ns against the 307.3ns it
started at, and nothing on it allocates: the per-call record it used to append
is gone, so a request's memory is bounded by its spans rather than by how many
times it called anything.

**What it costs a reader.** Two calls of the same function in the same bucket
are no longer separable: the page shows the longest of them with a repeat
count, and what they cost altogether. The display already did this at one
percent of the request, so the only change is the resolution.

**What the bucket decides.** Coverage depends on how a request's calls are
spread, which is the price of a fixed bucket over one which adapts to the
request. A million calls over one second, as a trace bounded at 10,000 spans:

| Calls spread over | Bucket | Spans | Calls covered | Wire |
| --- | ---: | ---: | ---: | ---: |
| 400 functions, round robin | 1ms | 10,000 | 2.5% | 1.62 MiB |
| 400 functions, round robin | 10ms | 10,000 | 25% | 1.66 MiB |
| 400 functions, round robin | 50ms | 8,000 | 100% | 1.35 MiB |
| 20 hot functions, 380 occasional | 10ms | 10,000 | 26% | 1.64 MiB |
| 20 hot functions, 380 occasional | 50ms | 7,960 | 100% | 1.33 MiB |

Against 136 MiB and a trace the transport refused to carry, every row is an
improvement, and the exact call count is reported whatever the coverage. But a
deployment whose requests run for much longer than a second, or call far more
distinct functions than a page usually does, has to raise `MAX_SPANS` or the
bucket to keep every call represented. That is what the knobs are for, and
adapting the bucket to the request — doubling it and merging buckets pairwise
when a trace fills — is the change which would remove the choice. It is worth
doing if the defaults turn out to drop calls in practice.

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

### 8. Aggregate once per trace, not once per keystroke — done

`functionsSetRows` re-ran `segmented.Unmarshal` and re-sorted its output on
every filter keystroke and every resize. The work depended only on the trace,
so it is now done once when a trace is opened and reused while it stays open,
and the filter narrows what is held. Rows still rebuild on resize, because
they carry their own widths.

Since item 4 the aggregate arrives with the trace, so what is cached here is
the ordering rather than the aggregation. The measurements below are from
before that, when this page still did both.

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
key of the same fields is allocation-free and compares the same way, which is
what the aggregator in item 4 now does on the ingest side.

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
