//go:build ignore

#define STRSZ 100 + 1
#define URI_MAX_LEN 2000

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

const char event_type_function[] = "function";

const char event_type_request_init[] = "request_init";

const char event_type_request_shutdown[] = "request_shutdown";

struct event {
  u8 type[STRSZ];
  u8 request_id[STRSZ];
  u8 uri[URI_MAX_LEN];
  u8 method[STRSZ];
  u8 function_name[STRSZ];
  u64 timestamp;
  u64 elapsed;
};

// Force emitting structs into the ELF.
const struct event *unused_event __attribute__((unused));

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 4096);
} events SEC(".maps");

struct {
  __uint(type, BPF_MAP_TYPE_LRU_HASH);
  __uint(max_entries, 256 * 4096);
  __type(key, u32);
  __type(value, u64);
} functions SEC(".maps");

// Used to inform the user space application that a request initialised.
SEC("uprobe/compass_request_init")
int uprobe_compass_request_init(struct pt_regs *ctx) {
  struct event *event;

  event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!event)
    return 0;

  // Add in the extra call information.
  bpf_core_read(&event->type, STRSZ, &event_type_request_init);
  bpf_probe_read_user_str(&event->request_id, STRSZ, (void *)ctx->REQUEST_INIT_ARG_REQUEST_ID);
  bpf_probe_read_user_str(&event->uri, URI_MAX_LEN, (void *)ctx->REQUEST_INIT_ARG_URI);
  bpf_probe_read_user_str(&event->method, STRSZ, (void *)ctx->REQUEST_INIT_ARG_METHOD);
  event->timestamp = bpf_ktime_get_ns();

  // Send it up to user space.
  bpf_ringbuf_submit(event, 0);

  return 0;
}

// Used record when a function begins.
SEC("uprobe/compass_php_function_begin")
int uprobe_compass_php_function_begin(struct pt_regs *ctx) {
  u8 id[STRSZ];

  bpf_probe_read_user_str(id, STRSZ, (void *)ctx->PHP_FUNCTION_BEGIN_ARG_ID);

  u64 ts;

  ts = bpf_ktime_get_ns();

  // Store in the map so that we can pick it up again when the function ends.
  bpf_map_update_elem(&functions, &id, &ts, BPF_ANY);
}

// Used to wrap up and send function execution data after ended.
SEC("uprobe/compass_php_function_end")
int uprobe_compass_php_function_end(struct pt_regs *ctx) {
  u8 id[STRSZ];

  bpf_probe_read_user_str(id, STRSZ, (void *)ctx->PHP_FUNCTION_END_ARG_ID);

  u64 *ts;

  ts = bpf_map_lookup_elem(&functions, id);
  if (!ts)
    return 0;

  u64 now = bpf_ktime_get_ns();

  s64 execution_time;

  execution_time = now - *ts;
  if (execution_time < 0)
    return 0;
  
  struct event *event;

  event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!event)
    return 0;

  // Add in the extra call information.
  bpf_core_read(&event->type, STRSZ, &event_type_function);
  bpf_probe_read_user_str(&event->request_id, STRSZ, (void *)ctx->PHP_FUNCTION_END_ARG_REQUEST_ID);
  bpf_probe_read_user_str(&event->function_name, STRSZ, (void *)ctx->PHP_FUNCTION_END_ARG_FUNCTION_NAME);
  event->timestamp = now;
  event->elapsed = execution_time;

  // Send it up to user space.
  bpf_ringbuf_submit(event, 0);

  return 0;
}

// Used to inform the user space application that a request has shutdown.
SEC("uprobe/compass_request_shutdown")
int uprobe_compass_request_shutdown(struct pt_regs *ctx) {
  struct event *event;

  event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!event)
    return 0;

  // Add in the extra call information.
  bpf_core_read(&event->type, STRSZ, &event_type_request_shutdown);
  bpf_probe_read_user_str(&event->request_id, STRSZ, (void *)ctx->REQUEST_SHUTDOWN_ARG_REQUEST_ID);
  event->timestamp = bpf_ktime_get_ns();

  // Send it up to user space.
  bpf_ringbuf_submit(event, 0);

  return 0;
}
