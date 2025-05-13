use phper::{arrays::ZArr, eg, pg, sys, values::ZVal};

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

// https://github.com/apache/skywalking-php/blob/master/src/request.rs#L152
pub fn get_request_server<'a>() -> anyhow::Result<&'a ZArr> {
    unsafe {
        let symbol_table = ZArr::from_mut_ptr(&raw mut eg!(symbol_table));
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

pub fn get_request_uri(server: &ZArr) -> String {
    server
        .get("REQUEST_URI")
        .and_then(z_val_to_string)
        .or_else(|| server.get("PHP_SELF").and_then(z_val_to_string))
        .or_else(|| server.get("SCRIPT_NAME").and_then(z_val_to_string))
        .unwrap_or_else(|| "/unknown".to_string())
}

pub fn get_request_method(server: &ZArr) -> String {
    server
        .get("REQUEST_METHOD")
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
