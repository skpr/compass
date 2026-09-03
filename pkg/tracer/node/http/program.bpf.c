//go:build ignore

#define STRSZ 100 + 1
#define URI_MAX_LEN 2000

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

enum event_type : __u8 {
  EVENT_TYPE_FUNCTION = 0,
  EVENT_TYPE_REQUEST_INIT = 1,
  EVENT_TYPE_REQUEST_SHUTDOWN = 2,
};

struct request_init_event {
  __u8 type;
  __u8 request_id[STRSZ];
  __u8 method[STRSZ];
  __u8 uri[URI_MAX_LEN];
  __u64 timestamp;
};

struct function_event {
  __u8 type;
  __u8 request_id[STRSZ];
  __u8 function_name[STRSZ];
  __u64 timestamp;
  __u64 elapsed;
  __u64 memory;
};

struct request_shutdown_event {
  __u8 type;
  __u8 request_id[STRSZ];
  __u64 timestamp;
};

const struct request_init_event *unused_request_init_event __attribute__((unused));
const struct function_event *unused_function_event __attribute__((unused));
const struct request_shutdown_event *unused_request_shutdown_event __attribute__((unused));

struct {
  __uint(type, BPF_MAP_TYPE_RINGBUF);
  __uint(max_entries, 256 * 4096);
} events SEC(".maps");

#define RINGBUF_STREAM_EVENTS 0

// Per-CPU accounting avoids an atomic operation on the probe hot path.
struct {
  __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
  __uint(max_entries, 1);
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
volatile const __u32 http_request_init_arg0_offset = 0;
volatile const __u32 http_request_init_arg1_offset = 0;
volatile const __u32 http_request_init_arg2_offset = 0;
volatile const __u32 http_function_arg0_offset = 0;
volatile const __u32 http_function_arg1_offset = 0;
volatile const __u32 http_function_arg2_offset = 0;
volatile const __u32 http_function_arg3_offset = 0;
volatile const __u32 http_request_shutdown_arg0_offset = 0;

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

SEC("uprobe/compass_http_request_init")
int uprobe_compass_http_request_init(struct pt_regs *ctx) {
  struct request_init_event *event =
      bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);
    return 0;
  }

  event->type = EVENT_TYPE_REQUEST_INIT;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, http_request_init_arg0_offset));
  bpf_core_read_user_str(&event->uri, URI_MAX_LEN,
                         (void *)read_arg(ctx, http_request_init_arg1_offset));
  bpf_core_read_user_str(&event->method, STRSZ,
                         (void *)read_arg(ctx, http_request_init_arg2_offset));
  event->timestamp = bpf_ktime_get_ns();

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_http_function")
int uprobe_compass_http_function(struct pt_regs *ctx) {
  struct function_event *event =
      bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);
    return 0;
  }

  event->type = EVENT_TYPE_FUNCTION;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, http_function_arg0_offset));
  bpf_core_read_user_str(&event->function_name, STRSZ,
                         (void *)read_arg(ctx, http_function_arg1_offset));
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = read_arg(ctx, http_function_arg2_offset);
  event->memory = read_arg(ctx, http_function_arg3_offset);

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_http_request_shutdown")
int uprobe_compass_http_request_shutdown(struct pt_regs *ctx) {
  struct request_shutdown_event *event =
      bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event) {
    count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);
    return 0;
  }

  event->type = EVENT_TYPE_REQUEST_SHUTDOWN;
  bpf_core_read_user_str(&event->request_id, STRSZ,
                         (void *)read_arg(ctx, http_request_shutdown_arg0_offset));
  event->timestamp = bpf_ktime_get_ns();

  bpf_ringbuf_submit(event, 0);
  return 0;
}
