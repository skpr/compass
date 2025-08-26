use cached::proc_macro::once;
use probe::probe_lazy;
use std::time::Duration;

// A cached function with TTL of 1 second
#[once(time = 1)]
pub fn probe_enabled() -> bool {
    probe_lazy!(compass, canary)
}
