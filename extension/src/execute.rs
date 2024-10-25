use crate::util::{
    get_combined_name, get_function_and_class_name, get_header_key, get_request_id,
    get_request_server, get_sapi_module_name,
};

use crate::{header, mode, threshold};
use chrono::prelude::*;
use phper::{sys, values::ExecuteData};
use probe::probe;
use std::{ptr::null_mut, time::SystemTime};
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

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        Err(_err) => {
            error!("unable to get server info: {}", _err);
            upstream_execute_ex(Some(execute_data));
            return;
        }
    };

    if header::block_execution(get_header_key(server)) {
        upstream_execute_ex(Some(execute_data));
        return;
    }

    let (function_name, class_name) = match get_function_and_class_name(execute_data) {
        Ok(x) => x,
        Err(_err) => {
            error!("failed to get class and function name: {}", _err);
            upstream_execute_ex(Some(execute_data));
            return;
        }
    };

    let function_name: String = function_name.map(|f| f.to_string()).unwrap_or_default();
    let class_name: String = class_name.map(|c| c.to_string()).unwrap_or_default();
    let combined_name = get_combined_name(class_name, function_name);

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

    let request_id = get_request_id(server);

    probe!(
        compass,
        php_function,
        request_id.as_ptr(),
        combined_name.as_ptr(),
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
