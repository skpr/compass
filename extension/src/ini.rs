use crate::{INI_ENABLED, INI_FUNCTION_THRESHOLD};
use once_cell::sync::Lazy;
use phper::ini::ini_get;

static IS_ENABLED: Lazy<bool> = Lazy::new(|| {
    return ini_get::<bool>(INI_ENABLED);
});

#[inline]
pub fn is_enabled() -> bool {
    *IS_ENABLED
}

static FUNCTION_THRESHOLD: Lazy<u128> = Lazy::new(|| {
    return u128::from(ini_get::<i64>(INI_FUNCTION_THRESHOLD).unsigned_abs());
});

#[inline]
pub fn is_under_function_threshold(elapsed: u128) -> bool {
    elapsed < *FUNCTION_THRESHOLD
}
