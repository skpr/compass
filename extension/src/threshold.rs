use once_cell::sync::Lazy;
use phper::ini::ini_get;

pub const INI_CONFIG: &str = "compass.function_threshold";

static FUNCTION_THRESHOLD: Lazy<u64> = Lazy::new(|| {
    return ini_get::<i64>(INI_CONFIG) as u64;
});

#[inline]
pub fn is_under_function_threshold(elapsed: u64) -> bool {
    elapsed < *FUNCTION_THRESHOLD
}

#[inline]
pub fn is_over_function_threshold(elapsed: u64) -> bool {
    elapsed > *FUNCTION_THRESHOLD
}
