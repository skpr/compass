use phper::{arrays::ZArr, eg, pg, strings::ZStr, sys, values::ExecuteData, values::ZVal};

use std::ffi::CStr;

use anyhow::Context;

// https://github.com/apache/skywalking-php/blob/master/src/request.rs#L93
pub fn jit_initialization() {
    unsafe {
        let jit_initialization: u8 = pg!(auto_globals_jit).into();
        if jit_initialization != 0 {
            let mut server = "_SERVER".to_string();
            sys::zend_is_auto_global_str(server.as_mut_ptr().cast(), server.len());
        }
    }
}

pub fn get_function_and_class_name(
    execute_data: &mut ExecuteData,
) -> anyhow::Result<(Option<String>, Option<String>)> {
    let function = execute_data.func();

    let class_name = function
        .get_class()
        .map(ZStr::to_str)
        .transpose()?
        .map(ToOwned::to_owned);

    let function_name = function
        .get_function_name()
        .map(ZStr::to_str)
        .transpose()?
        .map(ToOwned::to_owned);

    Ok((function_name, class_name))
}

pub fn get_combined_name(class_name: String, function_name: String) -> String {
    if class_name != "" && function_name != "" {
        return format!("{}::{}", class_name, function_name).to_string();
    }

    if class_name != "" {
        return class_name;
    }

    return function_name;
}

// https://github.com/apache/skywalking-php/blob/master/src/request.rs#L152
pub fn get_request_server<'a>() -> anyhow::Result<&'a ZArr> {
    unsafe {
        let symbol_table = ZArr::from_mut_ptr(&mut eg!(symbol_table));
        let carrier = symbol_table
            .get("_SERVER")
            .and_then(|carrier| carrier.as_z_arr())
            .context("$_SERVER is null")?;
        Ok(carrier)
    }
}

// Based off: https://github.com/apache/skywalking-php/blob/master/src/request.rs#L145C4-L145C27
pub fn get_request_id(server: &ZArr) -> String {
    server
        .get("HTTP_X_REQUEST_ID")
        .and_then(z_val_to_string)
        .unwrap_or_else(|| "UNKNOWN".to_string())
}

pub fn get_header_key(server: &ZArr) -> String {
    server
        .get("HTTP_X_COMPASS")
        .and_then(z_val_to_string)
        .unwrap_or_else(|| "UNKNOWN".to_string())
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
