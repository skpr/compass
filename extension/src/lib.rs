use phper::{
    arrays::ZArr,
    eg,
    ini::{ini_get, Policy},
    modules::Module,
    pg, php_get_module,
    strings::ZStr,
    sys,
    values::ExecuteData,
    values::ZVal,
};

use std::ffi::CStr;

use anyhow::Context;

use once_cell::sync::Lazy;

// Used to enable Compass.
const COMPASS_TRACE_ENABLED: &str = "compass.enabled";

// This is the entrypoint of the PHP extension.
#[php_get_module]
pub fn get_module() -> Module {
    let mut module = Module::new(
        env!("CARGO_CRATE_NAME"),
        env!("CARGO_PKG_VERSION"),
        env!("CARGO_PKG_AUTHORS"),
    );

    module.add_ini(COMPASS_TRACE_ENABLED, false, Policy::All);

    module.on_request_init(on_request_init);
    module.on_request_shutdown(on_request_shutdown);

    module.on_module_init(|| unsafe {
        sys::zend_observer_fcall_register(Some(observer_handler));
    });

    module
}

pub fn on_request_init() {
    if !is_enabled() {
        return;
    }

    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        return;
    }

    jit_initialization();

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        // @todo, This should not panic.
        Err(error) => panic!("Problem getting the server: {:?}", error),
    };

    let request_id = get_request_id(server);

    request_init(request_id.as_ptr());
}

pub fn on_request_shutdown() {
    if !is_enabled() {
        return;
    }

    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        return;
    }

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        // @todo, This should not panic.
        Err(error) => panic!("Problem getting the server: {:?}", error),
    };

    let request_id = get_request_id(server);

    request_shutdown(request_id.as_ptr());
}

pub unsafe extern "C" fn observer_handler(
    _execute_data: *mut sys::zend_execute_data,
) -> sys::zend_observer_fcall_handlers {
    if !is_enabled() {
        return Default::default();
    }

    sys::zend_observer_fcall_handlers {
        begin: Some(observer_begin),
        end: Some(observer_end),
    }
}

unsafe extern "C" fn observer_begin(execute_data: *mut sys::zend_execute_data) {
    let Some(execute_data) = ExecuteData::try_from_mut_ptr(execute_data) else {
        return;
    };

    let (function, class) = match get_function_and_class_name(execute_data) {
        Ok(x) => x,
        Err(_err) => {
            // @todo, Handle the error.
            return;
        }
    };

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        Err(_err) => {
            // @todo, Handle the error.
            return;
        }
    };

    let request_id = get_request_id(server);

    let function_name: String = function.map(|f| f.to_string()).unwrap_or_default();
    let class_name: String = class.map(|c| c.to_string()).unwrap_or_default();

    if function_name == "" && class_name == "" {
        return;
    }

    let combined = get_combined_name(class_name, function_name);

    function_begin(
        request_id.as_ptr(),
        format!("{:p}", execute_data.as_ptr()).as_ptr(),
        combined.as_ptr(),
    );
}

unsafe extern "C" fn observer_end(
    execute_data: *mut sys::zend_execute_data,
    _retval: *mut sys::zval,
) {
    let Some(execute_data) = ExecuteData::try_from_mut_ptr(execute_data) else {
        return;
    };

    let (function, class) = match get_function_and_class_name(execute_data) {
        Ok(x) => x,
        Err(_err) => {
            // @todo, Handle the error.
            return;
        }
    };

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        Err(_err) => {
            // @todo, Handle the error.
            return;
        }
    };

    let request_id = get_request_id(server);
    let function_name: String = function.map(|f| f.to_string()).unwrap_or_default();
    let class_name: String = class.map(|c| c.to_string()).unwrap_or_default();

    if function_name == "" && class_name == "" {
        return;
    }

    let combined = get_combined_name(class_name, function_name);

    function_begin(
        request_id.as_ptr(),
        format!("{:p}", execute_data.as_ptr()).as_ptr(),
        combined.as_ptr(),
    );
}

// Helper function taken from skywalking-php
// https://github.com/apache/skywalking-php/blob/master/src/execute.rs#L283
fn get_function_and_class_name(
    execute_data: &mut ExecuteData,
) -> anyhow::Result<(Option<String>, Option<String>)> {
    let function = execute_data.func();

    let function_name = function
        .get_function_name()
        .map(ZStr::to_str)
        .transpose()?
        .map(ToOwned::to_owned);
    let class_name = function
        .get_class()
        .map(|cls| cls.get_name().to_str().map(ToOwned::to_owned))
        .transpose()?;

    Ok((function_name, class_name))
}

static IS_ENABLED: Lazy<bool> = Lazy::new(|| {
    return ini_get::<bool>(COMPASS_TRACE_ENABLED);
});

#[inline]
pub fn is_enabled() -> bool {
    *IS_ENABLED
}

// https://github.com/apache/skywalking-php/blob/master/src/request.rs#L93
fn jit_initialization() {
    unsafe {
        let jit_initialization: u8 = pg!(auto_globals_jit).into();
        if jit_initialization != 0 {
            let mut server = "_SERVER".to_string();
            sys::zend_is_auto_global_str(server.as_mut_ptr().cast(), server.len());
        }
    }
}

// Based off: https://github.com/apache/skywalking-php/blob/master/src/request.rs#L145C4-L145C27
fn get_request_id(server: &ZArr) -> String {
    server
        .get("HTTP_X_REQUEST_ID")
        .and_then(z_val_to_string)
        .unwrap_or_else(|| "UNKNOWN".to_string())
}

// https://github.com/apache/skywalking-php/blob/master/src/request.rs#L152
fn get_request_server<'a>() -> anyhow::Result<&'a ZArr> {
    unsafe {
        let symbol_table = ZArr::from_mut_ptr(&mut eg!(symbol_table));
        let carrier = symbol_table
            .get("_SERVER")
            .and_then(|carrier| carrier.as_z_arr())
            .context("$_SERVER is null")?;
        Ok(carrier)
    }
}

// https://github.com/apache/skywalking-php/blob/master/src/util.rs#L63
pub fn z_val_to_string(zv: &ZVal) -> Option<String> {
    zv.as_z_str()
        .and_then(|zs| zs.to_str().ok())
        .map(|s| s.to_string())
}

// https://github.com/apache/skywalking-php/blob/master/src/util.rs#L86C1-L88C2
pub fn get_sapi_module_name() -> &'static CStr {
    unsafe { CStr::from_ptr(sys::sapi_module.name) }
}

fn get_combined_name(class_name: String, function_name: String) -> String {
    if class_name != "" && function_name != "" {
        return format!("{}::{}", class_name, function_name).to_string();
    }

    if class_name != "" {
        return class_name;
    }

    return function_name;
}

#[export_name = "request_init"]
pub fn request_init(request_id: *const u8) {
    // Do things.
}

#[export_name = "request_shutdown"]
pub fn request_shutdown(request_id: *const u8) {
    // Do things.
}

#[export_name = "function_begin"]
pub fn function_begin(request_id: *const u8, hash: *const u8, function_name: *const u8) {
    // Do things.
}

#[export_name = "function_end"]
pub fn function_end(request_id: *const u8, hash: *const u8, function_name: *const u8) {
    // Do things.
}
