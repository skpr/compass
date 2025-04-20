use crate::util::{get_request_id, get_request_server, get_sapi_module_name};
use phper::{sys, values::ExecuteData};
use probe::probe_lazy;
use tracing::error;

pub unsafe extern "C" fn observer_begin(execute_data: *mut sys::zend_execute_data) {
    let id = execute_data as usize;
    probe_lazy!(compass, php_function_begin, id);
}

pub unsafe extern "C" fn observer_end(
    execute_data: *mut sys::zend_execute_data,
    _return_value: *mut sys::zval,
) {
    let id = execute_data as usize;

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        Err(_err) => {
            error!("unable to get server info: {}", _err);
            return;
        }
    };

    let request_id = get_request_id(server);

    let execute_data = match ExecuteData::try_from_mut_ptr(execute_data) {
        Some(execute_data) => execute_data,
        None => {
            return;
        }
    };

    probe_lazy!(
        compass,
        php_function_end,
        id,
        request_id.as_ptr(),
        execute_data
            .func()
            .get_function_or_method_name()
            .as_c_str_ptr(),
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
