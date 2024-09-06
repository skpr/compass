use once_cell::sync::Lazy;
use phper::ini::ini_get;
use std::ffi::CStr;

pub const INI_ENABLED: &str = "compass.enabled";
pub const INI_MODE: &str = "compass.mode";
pub const INI_FUNCTION_THRESHOLD: &str = "compass.function_threshold";
pub const INI_HEADER_KEY: &str = "compass.header_key";

static IS_ENABLED: Lazy<bool> = Lazy::new(|| {
    return ini_get::<bool>(INI_ENABLED);
});

#[inline]
pub fn is_enabled() -> bool {
    *IS_ENABLED
}

static MODE: Lazy<String> = Lazy::new(|| {
    let defined_instance_name = ini_get::<Option<&CStr>>(INI_MODE)
        .and_then(|s| s.to_str().ok())
        .unwrap_or_default();

    defined_instance_name.trim().to_string()
});

#[inline]
pub fn mode_is_header_only() -> bool {
    MODE.as_str() == "header_only"
}

static HEADER_KEY: Lazy<String> = Lazy::new(|| {
    let defined_instance_name = ini_get::<Option<&CStr>>(INI_HEADER_KEY)
        .and_then(|s| s.to_str().ok())
        .unwrap_or_default();

    defined_instance_name.trim().to_string()
});

#[inline]
pub fn header_key_matches(have: String) -> bool {
    have.as_str() == HEADER_KEY.as_str()
}

#[inline]
pub fn header_key_is_set() -> bool {
    HEADER_KEY.as_str() != ""
}

static FUNCTION_THRESHOLD: Lazy<u128> = Lazy::new(|| {
    return u128::from(ini_get::<i64>(INI_FUNCTION_THRESHOLD).unsigned_abs());
});

#[inline]
pub fn is_under_function_threshold(elapsed: u128) -> bool {
    elapsed < *FUNCTION_THRESHOLD
}
