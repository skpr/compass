use once_cell::sync::Lazy;
use phper::ini::ini_get;
use std::ffi::CStr;

pub const INI_CONFIG: &str = "compass.mode";

static MODE: Lazy<String> = Lazy::new(|| {
    let defined_instance_name = ini_get::<Option<&CStr>>(INI_CONFIG)
        .and_then(|s| s.to_str().ok())
        .unwrap_or_default();

    defined_instance_name.trim().to_string()
});

#[inline]
pub fn header_enabled() -> bool {
    MODE.as_str() == "header"
}
