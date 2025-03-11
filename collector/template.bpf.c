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

// Used to wrap up and send function execution data.
SEC("uprobe/compass_php_function")
int uprobe_compass_php_function(struct pt_regs *ctx) {
  struct event *event;

  event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!event)
    return 0;

  // Add in the extra call information.
  bpf_core_read(&event->type, STRSZ, &event_type_function);
  bpf_probe_read_user_str(&event->request_id, STRSZ, (void *)ctx->PHP_FUNCTION_ARG_REQUEST_ID);
  bpf_probe_read_user_str(&event->function_name, STRSZ, (void *)ctx->PHP_FUNCTION_ARG_FUNCTION_NAME);
  event->timestamp = bpf_ktime_get_ns();
  event->elapsed = ctx->PHP_FUNCTION_ARG_ELAPSED;

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
