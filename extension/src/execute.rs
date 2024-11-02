use crate::util::{get_request_id, get_request_server, get_sapi_module_name};

use crate::{mode, threshold};
use chrono::prelude::*;
use phper::{sys, values::ExecuteData};
use probe::probe;
use std::ptr::null_mut;
use tracing::error;

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

    let function = execute_data.func();

    let class_name = match function.get_class() {
        Some(x) => x.get_name().to_str(),
        None => {
            upstream_execute_ex(Some(execute_data));
            return;
        }
    };

    let function_name = match function.get_function_name() {
        Some(x) => x.to_str(),
        None => {
            upstream_execute_ex(Some(execute_data));
            return;
        }
    };

    let start = get_unix_timestamp_micros();

    // Run the upstream function.
    upstream_execute_ex(Some(execute_data));

    let end = get_unix_timestamp_micros();

    if block_probe_event(
        mode::header_enabled(),
        threshold::is_under_function_threshold(end - start),
    ) {
        return;
    }

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        Err(_err) => {
            error!("unable to get server info: {}", _err);
            return;
        }
    };

    let request_id = get_request_id(server);

    probe!(
        compass,
        php_function,
        request_id.as_ptr(),
        class_name.unwrap().as_ptr(),
        function_name.unwrap().as_ptr(),
        start,
        end,
    );
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
