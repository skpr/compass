use crate::util::{
    get_combined_name, get_function_and_class_name, get_request_id, get_request_server,
    get_sapi_module_name,
};

use phper::{sys, values::ExecuteData};

use std::{ptr::null_mut, time::SystemTime};

use probe::probe;

static mut UPSTREAM_EXECUTE_EX: Option<
    unsafe extern "C" fn(execute_data: *mut sys::zend_execute_data),
> = None;

pub fn register_exec_functions() {
    unsafe {
        UPSTREAM_EXECUTE_EX = sys::zend_execute_ex;
        sys::zend_execute_ex = Some(execute_ex);
    }
}

unsafe extern "C" fn execute_ex(execute_data: *mut sys::zend_execute_data) {
    let execute_data = match ExecuteData::try_from_mut_ptr(execute_data) {
        Some(execute_data) => execute_data,
        None => {
            upstream_execute_ex(None);
            return;
        }
    };

    // @todo, Consider making this work for other situations eg. CLI.
    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        return;
    }

    let (function_name, class_name) = match get_function_and_class_name(execute_data) {
        Ok(x) => x,
        Err(_err) => {
            // @todo, Log the error.
            // error!(?err, "get function and class name failed");
            upstream_execute_ex(Some(execute_data));
            return;
        }
    };

    let function_name: String = function_name.map(|f| f.to_string()).unwrap_or_default();
    let class_name: String = class_name.map(|c| c.to_string()).unwrap_or_default();
    let combined_name = get_combined_name(class_name, function_name);

    let now = SystemTime::now();

    // Run the upstream function.
    upstream_execute_ex(Some(execute_data));

    let elapsed = match now.elapsed() {
        Ok(elapsed) => elapsed,
        Err(_e) => {
            return;
        }
    };

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        // @todo, This should not panic.
        Err(error) => panic!("Problem getting the server: {:?}", error),
    };

    let request_id = get_request_id(server);

    probe!(
        compass,
        php_function,
        request_id.as_ptr(),
        combined_name.as_ptr(),
        elapsed.as_nanos()
    );
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
