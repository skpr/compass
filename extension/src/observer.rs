use crate::canary::probe_enabled;
use crate::fpm::is_fpm;
use crate::threshold;
use crate::util::{get_request_id, get_request_server};
use coarsetime::Instant;
use phper::{sys, values::ExecuteData};
use probe::probe_lazy;
use std::cell::RefCell;

thread_local! {
    static FUNCTION_TIMES: RefCell<Vec<(usize, Instant)>> = RefCell::new(Vec::with_capacity(32));
}

#[inline(always)]
fn set_function_time(exec_ptr: *mut sys::zend_execute_data, now: Instant) {
    let key = exec_ptr as usize;
    FUNCTION_TIMES.with(|stack| stack.borrow_mut().push((key, now)));
}

#[inline(always)]
fn take_elapsed_if_over_threshold(exec_ptr: *mut sys::zend_execute_data) -> Option<u64> {
    let key = exec_ptr as usize;
    FUNCTION_TIMES.with(|stack| {
        let mut stack = stack.borrow_mut();
        if let Some(pos) = stack.iter().rposition(|(k, _)| *k == key) {
            let (_, start) = stack.remove(pos);
            let elapsed = start.elapsed().as_nanos() as u64;
            if threshold::is_over_function_threshold(elapsed) {
                return Some(elapsed);
            }
        }
        None
    })
}

pub unsafe extern "C" fn observer_begin(execute_data: *mut sys::zend_execute_data) {
    set_function_time(execute_data, Instant::recent());
}

pub unsafe extern "C" fn observer_end(
    execute_data: *mut sys::zend_execute_data,
    _return_value: *mut sys::zval,
) {
    let elapsed = match take_elapsed_if_over_threshold(execute_data) {
        Some(e) => e,
        None => return,
    };

    let server = match get_request_server() {
        Ok(s) => s,
        Err(_) => return, // Avoid logging in hot path
    };

    let request_id = get_request_id(server);

    // Explicit unsafe block as required in Rust 2024
    let execute_data = match unsafe { ExecuteData::try_from_mut_ptr(execute_data) } {
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
    if !probe_enabled() || !is_fpm() {
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
