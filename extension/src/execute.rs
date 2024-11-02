use crate::util::get_sapi_module_name;

use chrono::prelude::*;
use phper::{sys, values::ExecuteData};
use std::ptr::null_mut;

static mut UPSTREAM_EXECUTE_EX: Option<
    unsafe extern "C" fn(execute_data: *mut sys::zend_execute_data),
> = None;

// This function swaps out the PHP exec function for our own. Allowing us to wrap it.
pub fn register_exec_functions() {
    unsafe {
        UPSTREAM_EXECUTE_EX = sys::zend_execute_ex;
        sys::zend_execute_ex = Some(execute_ex);
    }
}

// This is our exec function that wraps the upstream PHP one.
// This allows us to gather our execution timing data.
unsafe extern "C" fn execute_ex(execute_data: *mut sys::zend_execute_data) {
    let execute_data = match ExecuteData::try_from_mut_ptr(execute_data) {
        Some(execute_data) => execute_data,
        None => {
            upstream_execute_ex(None);
            return;
        }
    };

    // @todo, Consider making this work for other situations eg. Apache, CLI etc
    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        upstream_execute_ex(Some(execute_data));
        return;
    }

    upstream_execute_ex(Some(execute_data));
    return;
}

// Helper function to allow all probes if the header mode is enabled.
fn block_probe_event(header_mode_enabled: bool, is_under_threshold: bool) -> bool {
    if header_mode_enabled {
        return false;
    };

    is_under_threshold
}

#[inline]
fn upstream_execute_ex(execute_data: Option<&mut ExecuteData>) {
    unsafe {
        if let Some(f) = UPSTREAM_EXECUTE_EX {
            f(execute_data
                .map(ExecuteData::as_mut_ptr)
                .unwrap_or(null_mut()))
        }
    }
}

pub fn get_unix_timestamp_micros() -> i64 {
    let now = Utc::now();
    now.timestamp_micros()
}
