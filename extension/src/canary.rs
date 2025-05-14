use cached::proc_macro::once;
use probe::probe_lazy;

// A cached function with TTL of 10 seconds
#[once(time = 1)]
pub fn probe_enabled() -> bool {
    probe_lazy!(compass, canary)
}
