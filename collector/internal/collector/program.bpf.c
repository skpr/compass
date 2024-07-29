//go:build ignore

// # $ readelf -n /usr/lib/php/modules/compass.so
//
//   Displaying notes found in: .note.gnu.build-id
//     Owner                Data size 	Description
//     GNU                  0x00000014	NT_GNU_BUILD_ID (unique build ID bitstring)
//       Build ID: fde3700e522439278b248b3eb082fa60a0f69dae
//
//   Displaying notes found in: .note.stapsdt
//     Owner                Data size 	Description
//     stapsdt              0x00000045	NT_STAPSDT (SystemTap probe descriptors)
//       Provider: compass
//       Name: php_function
//       Location: 0x000000000000e78c, Base: 0x000000000005ea9b, Semaphore: 0x0000000000000000
//       Arguments: -8@%rdi -8@%rbx -8@%rax
//     stapsdt              0x00000039	NT_STAPSDT (SystemTap probe descriptors)
//       Provider: compass
//       Name: request_shutdown
//       Location: 0x000000000000e9a0, Base: 0x000000000005ea9b, Semaphore: 0x0000000000000000
//       Arguments: -8@%rdi

#define STRSZ 100 + 1

#include "common.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

char __license[] SEC("license") = "Dual MIT/GPL";

const char event_type_function[] = "function";

const char event_type_request_shutdown[] = "request_shutdown";

struct event {
  u8 type[STRSZ];
  u8 request_id[STRSZ];
  u8 function_name[STRSZ];
  u64 execution_time;
};

// Force emitting structs into the ELF.
const struct event *unused_event __attribute__((unused));

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 4096);
} events SEC(".maps");

// Used to wrap up and send function execution data.
SEC("uprobe/compass_php_function")
int uprobe_compass_php_function(struct pt_regs *ctx) {
  return 0;
}

// Used to inform the user space application that a request has shutdown.
SEC("uprobe/compass_request_shutdown")
int uprobe_compass_request_shutdown(struct pt_regs *ctx) {
  return 0;
}
