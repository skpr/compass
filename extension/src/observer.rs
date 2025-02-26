use crate::util::{get_header_key, get_request_id, get_request_server, get_sapi_module_name};
use crate::{header, threshold};
use coarsetime::Instant;
use phper::{strings::ZStr, sys, values::ExecuteData};
use probe::probe_lazy;
use std::{cell::RefCell, collections::HashMap};
use tracing::error;

thread_local! {
    static CONTEXT_GUARD_MAP: RefCell<HashMap<usize, Instant>> = RefCell::new(HashMap::new());
}

fn store_guard(exec_ptr: *mut sys::zend_execute_data, guard: Instant) {
    let key = exec_ptr as usize;
    CONTEXT_GUARD_MAP.with(|map| {
        map.borrow_mut().insert(key, guard);
    });
}

fn take_guard(exec_ptr: *mut sys::zend_execute_data) -> Option<Instant> {
    let key = exec_ptr as usize;
    CONTEXT_GUARD_MAP.with(|map| map.borrow_mut().remove(&key))
}

pub unsafe extern "C" fn observer_begin(execute_data: *mut sys::zend_execute_data) {
    store_guard(execute_data, Instant::now());
}

pub unsafe extern "C" fn observer_end(
    execute_data: *mut sys::zend_execute_data,
    _return_value: *mut sys::zval,
) {
    let start = match take_guard(execute_data) {
        Some(start) => start,
        None => {
            return;
        }
    };

    let elapsed = start.elapsed().as_nanos();

    if threshold::is_under_function_threshold(elapsed) {
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

    if header::block_execution(get_header_key(server)) {
        return;
    }

    let request_id = get_request_id(server);

    let execute_data = match ExecuteData::try_from_mut_ptr(execute_data) {
        Some(execute_data) => execute_data,
        None => {
            return;
        }
    };

    probe_lazy!(
        compass,
        php_function,
        request_id.as_ptr(),
        execute_data
            .func()
            .get_function_or_method_name()
            .as_c_str_ptr(),
        elapsed,
    );
}

pub unsafe extern "C" fn observer_instrument(
    _execute_data: *mut sys::zend_execute_data,
) -> sys::zend_observer_fcall_handlers {
    // @todo, Consider making this work for other situations eg. Apache, CLI etc
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
