//go:build ignore

// # readelf -n /usr/lib/php/modules/compass.so
//
// Displaying notes found in: .note.gnu.build-id
//   Owner                Data size 	Description
//   GNU                  0x00000014	NT_GNU_BUILD_ID (unique build ID bitstring)
//     Build ID: cd16bd69e73b609d3ab6bf9c5657bfd024ee52f0
//
// Displaying notes found in: .note.stapsdt
//   Owner                Data size 	Description
//   stapsdt              0x00000039	NT_STAPSDT (SystemTap probe descriptors)
//     Provider: compass
//     Name: fpm_request_init
//     Location: 0x000000000000bbfa, Base: 0x000000000005e9eb, Semaphore: 0x0000000000071078
//     Arguments: -8@%r14
//   stapsdt              0x0000003d	NT_STAPSDT (SystemTap probe descriptors)
//     Provider: compass
//     Name: fpm_request_shutdown
//     Location: 0x000000000000be96, Base: 0x000000000005e9eb, Semaphore: 0x000000000007107a
//     Arguments: -8@%r14
//   stapsdt              0x0000004b	NT_STAPSDT (SystemTap probe descriptors)
//     Provider: compass
//     Name: php_function_begin
//     Location: 0x000000000000c46f, Base: 0x000000000005e9eb, Semaphore: 0x000000000007107c
//     Arguments: -8@%rcx -8@%rdi -8@%rax
//   stapsdt              0x00000049	NT_STAPSDT (SystemTap probe descriptors)
//     Provider: compass
//     Name: php_function_end
//     Location: 0x000000000000ca1f, Base: 0x000000000005e9eb, Semaphore: 0x000000000007107e
//     Arguments: -8@%rcx -8@%rdi -8@%rax

#define STRSZ 100 + 1

#define MAX_ENTRIES 10240

#include "common.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

const char event_type_function[] = "function";

const char event_type_request[] = "request";

struct event {
  u8 type[STRSZ];
  u8 request_id[STRSZ];
  u8 name[STRSZ];
  u64 execution_time;
};

// Force emitting structs into the ELF.
const struct event *unused_event __attribute__((unused));

struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, MAX_ENTRIES);
  __type(key, u32);
  __type(value, u64);
} functions SEC(".maps");

struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, MAX_ENTRIES);
  __type(key, u32);
  __type(value, u64);
} requests SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

// Used to initialize tracking for a function execution.
SEC("uprobe/compass_php_function_begin")
int uprobe_compass_php_function_begin(struct pt_regs *ctx) {
  u8 id[STRSZ];

  bpf_probe_read_user_str(id, STRSZ, (void *)ctx->rdi);

  u64 ts;

  ts = bpf_ktime_get_ns();

  // Store in the map so that we can pick it up again when the function ends.
  bpf_map_update_elem(&functions, &id, &ts, BPF_ANY);

  return 0;
}

// Used to wrap up and send function execution data.
SEC("uprobe/compass_php_function_end")
int uprobe_compass_php_function_end(struct pt_regs *ctx) {
  u8 id[STRSZ];

  bpf_probe_read_user_str(id, STRSZ, (void *)ctx->rdi);

  u64 *ts;

  ts = bpf_map_lookup_elem(&functions, id);
  if (!ts)
    return 0;

  s64 execution_time;

  execution_time = bpf_ktime_get_ns() - *ts;
  if (execution_time < 0)
    goto cleanup;

  struct event *event;

  event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!event)
    goto cleanup;

  // Add in the extra call information.
  bpf_core_read(&event->type, STRSZ, &event_type_function);
  bpf_probe_read_user_str(&event->request_id, STRSZ, (void *)ctx->rcx);
  bpf_probe_read_user_str(&event->name, STRSZ, (void *)ctx->rax);
  event->execution_time = execution_time;

  // Send it up to user space.
  //bpf_ringbuf_submit(event, 0);
  // @todo, Store it in a map for pickup later.

  // Cleanup function tracking from the map.
  cleanup:
  bpf_map_delete_elem(&functions, id);
  return 0;
}

// Used to inform the user space application that a new request has started.
SEC("uprobe/compass_fpm_request_init")
int uprobe_compass_fpm_request_init(struct pt_regs *ctx) {
  u8 request_id[STRSZ];

  bpf_probe_read_user_str(request_id, STRSZ, (void *)ctx->r14);

  u64 ts;

  ts = bpf_ktime_get_ns();

  // Store in the map so that we can pick it up again when the function ends.
  bpf_map_update_elem(&requests, &request_id, &ts, BPF_ANY);

  return 0;
}

// Used to inform the user space application that a request has shutdown.
SEC("uprobe/compass_fpm_request_shutdown")
int uprobe_compass_fpm_request_shutdown(struct pt_regs *ctx) {
  u8 request_id[STRSZ];

  bpf_probe_read_user_str(request_id, STRSZ, (void *)ctx->r14);

  u64 *ts;

  ts = bpf_map_lookup_elem(&requests, request_id);
  if (!ts)
    return 0;

  s64 execution_time;

  execution_time = bpf_ktime_get_ns() - *ts;
  if (execution_time < 0)
    goto cleanup;

  struct event *event;

  event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!event)
    goto cleanup;

  bpf_core_read(&event->type, STRSZ, &event_type_request);
  bpf_probe_read_user_str(&event->request_id, STRSZ, (void *)ctx->r14);
  event->execution_time = execution_time;

  // Send it up to user space.
  bpf_ringbuf_submit(event, 0);

  // Cleanup request tracking from the map.
  cleanup:
  bpf_map_delete_elem(&requests, request_id);
  return 0;
}
