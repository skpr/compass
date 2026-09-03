//go:build ignore

#define STRSZ 100 + 1
#define URI_MAX_LEN 2000

// Fully qualified Drupal method and class names routinely run past STRSZ, so
// the caller and object type of a cache event get larger buffers than a
// function name does. Truncating either of these turns a name you can act on
// into one you cannot.
#define CALLER_MAX_LEN 255 + 1
#define OBJECT_TYPE_MAX_LEN 255 + 1

// Cache tags and contexts arrive space delimited, so a render array with many
// tags produces a long string. These are the points past which the tail is
// dropped rather than the event.
#define CACHE_TAGS_MAX_LEN 1024
#define CACHE_CONTEXTS_MAX_LEN 512

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

enum event_type : __u8 {
  EVENT_TYPE_FUNCTION = 0,
  EVENT_TYPE_REQUEST_INIT = 1,
  EVENT_TYPE_REQUEST_SHUTDOWN = 2,
  EVENT_TYPE_DRUPAL_CACHE_RENDER_ARRAY = 3,
  EVENT_TYPE_DRUPAL_CACHE_OBJECT = 4,
};

struct event {
  __u8 type;
  __u8 request_id[STRSZ];
  __u8 method[STRSZ];
  __u8 function_name[STRSZ];
  __u8 uri[URI_MAX_LEN];
  __u64 timestamp;
  __u64 elapsed;
  __u64 memory;
};

const struct event *unused_event __attribute__((unused));

struct {
  __uint(type, BPF_MAP_TYPE_RINGBUF);
  __uint(max_entries, 256 * 4096);
} events SEC(".maps");

// The max age Drupal uses for "cacheable forever".
#define DRUPAL_CACHE_MAX_AGE_PERMANENT -1

// Drupal derives cacheability with no threshold in front of it, so these events
// arrive far more often, and carry far more string data, than a function call
// does. They get their own event type and ring buffer to keep that weight off
// the function path, which is the hot one.
struct drupal_cache_event {
  __u8 type;
  __u8 request_id[STRSZ];
  __u8 caller[CALLER_MAX_LEN];
  __u8 object_type[OBJECT_TYPE_MAX_LEN];
  __u8 tags[CACHE_TAGS_MAX_LEN];
  __u8 contexts[CACHE_CONTEXTS_MAX_LEN];
  __s64 max_age;
  __u64 timestamp;
};

const struct drupal_cache_event *unused_drupal_cache_event __attribute__((unused));

// Sized larger than the function ring buffer: these records are bigger and
// arrive in greater numbers, so the same capacity would hold far less history.
struct {
  __uint(type, BPF_MAP_TYPE_RINGBUF);
  __uint(max_entries, 1024 * 4096);
} drupal_cache_events SEC(".maps");

#define RINGBUF_STREAM_EVENTS 0
#define RINGBUF_STREAM_DRUPAL_CACHE 1

// Per-CPU counters keep reserve-failure accounting off the contended hot path.
// User space sums CPUs and exports only deltas from this BPF object's lifetime.
struct {
  __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
  __uint(max_entries, 2);
  __type(key, __u32);
  __type(value, __u64);
} ringbuf_reserve_failures SEC(".maps");

static __always_inline void count_ringbuf_reserve_failure(__u32 stream) {
  __u64 *failures = bpf_map_lookup_elem(&ringbuf_reserve_failures, &stream);
  if (failures)
    (*failures)++;
}

// Byte offsets of probe arguments within struct pt_regs. These are populated
// from Go (via ebpf.Variable / CollectionSpec.Variables) before the program is
// loaded into the kernel, based on the USDT argument descriptors parsed from
// the .note.stapsdt section of the instrumented binary.
volatile const __u32 fpm_request_init_arg0_offset = 0;
volatile const __u32 fpm_request_init_arg1_offset = 0;
volatile const __u32 fpm_request_init_arg2_offset = 0;
volatile const __u32 fpm_function_arg0_offset = 0;
volatile const __u32 fpm_function_arg1_offset = 0;
volatile const __u32 fpm_function_arg2_offset = 0;
volatile const __u32 fpm_function_arg3_offset = 0;
volatile const __u32 fpm_request_shutdown_arg0_offset = 0;
volatile const __u32 drupal_cache_render_array_arg0_offset = 0;
volatile const __u32 drupal_cache_render_array_arg1_offset = 0;
volatile const __u32 drupal_cache_render_array_arg2_offset = 0;
volatile const __u32 drupal_cache_render_array_arg3_offset = 0;
volatile const __u32 drupal_cache_render_array_arg4_offset = 0;
volatile const __u32 drupal_cache_object_arg0_offset = 0;
volatile const __u32 drupal_cache_object_arg1_offset = 0;
volatile const __u32 drupal_cache_object_arg2_offset = 0;
volatile const __u32 drupal_cache_object_arg3_offset = 0;
volatile const __u32 drupal_cache_object_arg4_offset = 0;
volatile const __u32 drupal_cache_object_arg5_offset = 0;

// Discards cache events which carry no cacheability at all: permanent max age,
// no tags and no contexts. Such an event cannot change the effective max age,
// the tag union or the context union, so dropping it costs the consumer nothing
// while removing the bulk of the volume. Set to 0 to emit every event, which is
// how you measure what the filter is actually discarding.
volatile const __u8 drupal_cache_filter_empty = 1;

// read_arg reads a 64-bit register value from pt_regs at a given byte offset.
static __always_inline __u64 read_arg(struct pt_regs *ctx, __u32 offset) {
  __u64 val = 0;
  bpf_probe_read_kernel(&val, sizeof(val), (char *)ctx + offset);
  return val;
}

SEC("uprobe/compass_canary")
int uprobe_compass_canary(struct pt_regs *ctx) {
  return 0;
}

SEC("uprobe/compass_fpm_request_init")
int uprobe_compass_fpm_request_init(struct pt_regs *ctx) {
  struct event *event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);
    return 0;
  }

  event->type = EVENT_TYPE_REQUEST_INIT;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, fpm_request_init_arg0_offset));
  bpf_core_read_user_str(&event->uri, URI_MAX_LEN,
                         (void *)read_arg(ctx, fpm_request_init_arg1_offset));
  bpf_core_read_user_str(&event->method, STRSZ,
                         (void *)read_arg(ctx, fpm_request_init_arg2_offset));
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = 0;

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_fpm_function")
int uprobe_compass_fpm_function(struct pt_regs *ctx) {
  struct event *event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);
    return 0;
  }

  event->type = EVENT_TYPE_FUNCTION;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, fpm_function_arg0_offset));
  bpf_core_read_user_str(&event->function_name, STRSZ,
                         (void *)read_arg(ctx, fpm_function_arg1_offset));
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = read_arg(ctx, fpm_function_arg2_offset);
  event->memory = read_arg(ctx, fpm_function_arg3_offset);

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_fpm_request_shutdown")
int uprobe_compass_fpm_request_shutdown(struct pt_regs *ctx) {
  struct event *event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);
    return 0;
  }

  event->type = EVENT_TYPE_REQUEST_SHUTDOWN;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, fpm_request_shutdown_arg0_offset));
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = 0;

  bpf_ringbuf_submit(event, 0);
  return 0;
}

// drupal_cache_event_empty reports whether an event carries no cacheability at
// all, which makes it a no-op for every consumer of these events.
static __always_inline int drupal_cache_event_empty(struct drupal_cache_event *event) {
  return event->max_age == DRUPAL_CACHE_MAX_AGE_PERMANENT &&
         event->tags[0] == 0 && event->contexts[0] == 0;
}

SEC("uprobe/compass_drupal_cacheablemetadata_createfromrenderarray")
int uprobe_compass_drupal_cache_render_array(struct pt_regs *ctx) {
  struct drupal_cache_event *event =
      bpf_ringbuf_reserve(&drupal_cache_events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_DRUPAL_CACHE);
    return 0;
  }

  // A ring buffer record is handed over uninitialised, and a failed string read
  // does not guarantee it writes a terminator, so every string field is
  // terminated up front. Otherwise a failed read leaves whatever the previous
  // record put there, which the empty check below would then act on.
  event->request_id[0] = 0;
  event->caller[0] = 0;
  event->object_type[0] = 0;
  event->tags[0] = 0;
  event->contexts[0] = 0;

  event->type = EVENT_TYPE_DRUPAL_CACHE_RENDER_ARRAY;
  bpf_core_read_user_str(
      &event->request_id, STRSZ,
      (void *)read_arg(ctx, drupal_cache_render_array_arg0_offset));
  bpf_core_read_user_str(
      &event->caller, CALLER_MAX_LEN,
      (void *)read_arg(ctx, drupal_cache_render_array_arg1_offset));
  event->max_age =
      (__s64)read_arg(ctx, drupal_cache_render_array_arg2_offset);
  bpf_core_read_user_str(
      &event->tags, CACHE_TAGS_MAX_LEN,
      (void *)read_arg(ctx, drupal_cache_render_array_arg3_offset));
  bpf_core_read_user_str(
      &event->contexts, CACHE_CONTEXTS_MAX_LEN,
      (void *)read_arg(ctx, drupal_cache_render_array_arg4_offset));
  event->timestamp = bpf_ktime_get_ns();

  if (drupal_cache_filter_empty && drupal_cache_event_empty(event)) {
    bpf_ringbuf_discard(event, 0);
    return 0;
  }

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_drupal_cacheablemetadata_createfromobject")
int uprobe_compass_drupal_cache_object(struct pt_regs *ctx) {
  struct drupal_cache_event *event =
      bpf_ringbuf_reserve(&drupal_cache_events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_DRUPAL_CACHE);
    return 0;
  }

  // A ring buffer record is handed over uninitialised, and a failed string read
  // does not guarantee it writes a terminator, so every string field is
  // terminated up front. Otherwise a failed read leaves whatever the previous
  // record put there, which the empty check below would then act on.
  event->request_id[0] = 0;
  event->caller[0] = 0;
  event->object_type[0] = 0;
  event->tags[0] = 0;
  event->contexts[0] = 0;

  event->type = EVENT_TYPE_DRUPAL_CACHE_OBJECT;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, drupal_cache_object_arg0_offset));
  bpf_core_read_user_str(&event->caller, CALLER_MAX_LEN,
                         (void *)read_arg(ctx, drupal_cache_object_arg1_offset));
  event->max_age = (__s64)read_arg(ctx, drupal_cache_object_arg2_offset);
  bpf_core_read_user_str(&event->object_type, OBJECT_TYPE_MAX_LEN,
                         (void *)read_arg(ctx, drupal_cache_object_arg3_offset));
  bpf_core_read_user_str(&event->tags, CACHE_TAGS_MAX_LEN,
                         (void *)read_arg(ctx, drupal_cache_object_arg4_offset));
  bpf_core_read_user_str(&event->contexts, CACHE_CONTEXTS_MAX_LEN,
                         (void *)read_arg(ctx, drupal_cache_object_arg5_offset));
  event->timestamp = bpf_ktime_get_ns();

  // The object type is cacheability metadata in its own right, so an event with
  // one is never empty even when everything else is unset.
  if (drupal_cache_filter_empty && event->object_type[0] == 0 &&
      drupal_cache_event_empty(event)) {
    bpf_ringbuf_discard(event, 0);
    return 0;
  }

  bpf_ringbuf_submit(event, 0);
  return 0;
}
