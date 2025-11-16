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

struct event {
  __u8 type;
  __u8 request_id[STRSZ];
  __u8 method[STRSZ];
  __u8 function_name[STRSZ];
  __u8 uri[URI_MAX_LEN];
  __u64 timestamp;
  __u64 elapsed;
};

const struct event *unused_event __attribute__((unused));

struct {
  __uint(type, BPF_MAP_TYPE_RINGBUF);
  __uint(max_entries, 256 * 4096);
} events SEC(".maps");

SEC("uprobe/compass_canary")
int uprobe_compass_canary(struct pt_regs *ctx) {
  return 0;
}

SEC("uprobe/compass_request_init")
int uprobe_compass_request_init(struct pt_regs *ctx) {
  struct event *event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event)
    return 0;

  event->type = EVENT_TYPE_REQUEST_INIT;
  bpf_core_read_user_str(&event->request_id, STRSZ, (void *)ctx->REQUEST_INIT_ARG_REQUEST_ID);
  bpf_core_read_user_str(&event->method, STRSZ, (void *)ctx->REQUEST_INIT_ARG_METHOD);
  bpf_core_read_user_str(&event->uri, URI_MAX_LEN, (void *)ctx->REQUEST_INIT_ARG_URI);
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = 0;

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_php_function")
int uprobe_compass_php_function(struct pt_regs *ctx) {
  struct event *event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event)
    return 0;

  event->type = EVENT_TYPE_FUNCTION;
  bpf_core_read_user_str(&event->request_id, STRSZ, (void *)ctx->PHP_FUNCTION_ARG_REQUEST_ID);
  bpf_core_read_user_str(&event->function_name, STRSZ, (void *)ctx->PHP_FUNCTION_ARG_FUNCTION_NAME);
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = ctx->PHP_FUNCTION_ARG_ELAPSED;

  bpf_ringbuf_submit(event, 0);
  return 0;
}

SEC("uprobe/compass_request_shutdown")
int uprobe_compass_request_shutdown(struct pt_regs *ctx) {
  struct event *event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
  if (!event)
    return 0;

  event->type = EVENT_TYPE_REQUEST_SHUTDOWN;
  bpf_core_read_user_str(&event->request_id, STRSZ, (void *)ctx->REQUEST_SHUTDOWN_ARG_REQUEST_ID);
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = 0;

  bpf_ringbuf_submit(event, 0);
  return 0;
}
