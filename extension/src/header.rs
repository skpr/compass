use crate::mode;
use once_cell::sync::Lazy;
use phper::ini::ini_get;
use std::ffi::CStr;

pub const INI_CONFIG: &str = "compass.header";

static HEADER: Lazy<String> = Lazy::new(|| {
    let defined_instance_name = ini_get::<Option<&CStr>>(INI_CONFIG)
        .and_then(|s| s.to_str().ok())
        .unwrap_or_default();

    defined_instance_name.trim().to_string()
});

#[inline]
pub fn header_key_matches(have: String) -> bool {
    have.as_str() == HEADER.as_str()
}

pub fn block_execution(have: String) -> bool {
    if !mode::header_enabled() {
        return false;
    }

    // Do not block an execution (FPM or function call) if the header matches.
    if header_key_matches(have) {
        return false;
    }

    true
}
