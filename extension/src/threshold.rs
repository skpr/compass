use once_cell::sync::Lazy;
use phper::ini::ini_get;

pub const INI_CONFIG: &str = "compass.function_threshold";

static FUNCTION_THRESHOLD: Lazy<u128> = Lazy::new(|| {
    return u128::from(ini_get::<i64>(INI_CONFIG).unsigned_abs());
});

#[inline]
pub fn is_under_function_threshold(elapsed: u128) -> bool {
    elapsed < *FUNCTION_THRESHOLD
}
