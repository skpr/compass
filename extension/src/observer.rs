use crate::canary::probe_enabled;
use crate::threshold;
use crate::util::{get_request_id, get_request_server, get_sapi_module_name};
use coarsetime::Instant;
use phper::{sys, values::ExecuteData};
use probe::probe_lazy;
use rustc_hash::FxHashMap;
use std::{cell::RefCell};
use tracing::error;

thread_local! {
    static CONTEXT_FUNCTION_MAP: RefCell<FxHashMap<usize, Instant>> = RefCell::new(FxHashMap::default());
}

#[inline(always)]
fn set_function_time(exec_ptr: *mut sys::zend_execute_data, now: Instant) {
    let key = exec_ptr as usize;
    CONTEXT_FUNCTION_MAP.with(|map| {
        if let Ok(mut m) = map.try_borrow_mut() {
            m.insert(key, now);
        }
    });
}

#[inline(always)]
fn get_function_time(exec_ptr: *mut sys::zend_execute_data) -> Option<Instant> {
    let key = exec_ptr as usize;
    CONTEXT_FUNCTION_MAP.with(|map| map.borrow_mut().remove(&key))
}

pub unsafe extern "C" fn observer_begin(execute_data: *mut sys::zend_execute_data) {
    // Instant::recent is faster and sufficient for short-duration timing.
    set_function_time(execute_data, Instant::recent());
}

pub unsafe extern "C" fn observer_end(
    execute_data: *mut sys::zend_execute_data,
    _return_value: *mut sys::zval,
) {
    let start = match get_function_time(execute_data) {
        Some(start) => start,
        None => return,
    };

    let elapsed = start.elapsed().as_nanos();

    if threshold::is_under_function_threshold(elapsed) {
        return;
    }

    let server = match get_request_server() {
        Ok(s) => s,
        Err(err) => {
            // Consider rate-limiting this if log volume is high
            error!("unable to get server info: {}", err);
            return;
        }
    };

    let request_id = get_request_id(server);

    let execute_data = match ExecuteData::try_from_mut_ptr(execute_data) {
        Some(data) => data,
        None => return,
    };

    let function_name = execute_data.func().get_function_or_method_name();

    probe_lazy!(
        compass,
        php_function,
        request_id.as_ptr(),
        function_name.as_c_str_ptr(),
        elapsed,
    );
}

pub unsafe extern "C" fn observer_instrument(
    _execute_data: *mut sys::zend_execute_data,
) -> sys::zend_observer_fcall_handlers {
    if !probe_enabled() {
        return sys::zend_observer_fcall_handlers {
            begin: None,
            end: None,
        };
    }

    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        return sys::zend_observer_fcall_handlers {
            begin: None,
            end: None,
        };
    }

    sys::zend_observer_fcall_handlers {
        begin: Some(observer_begin),
        end: Some(observer_end),
    }
}