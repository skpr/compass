use once_cell::sync::Lazy;
use phper::ini::ini_get;

pub const INI_CONFIG: &str = "compass.enabled";

static IS_ENABLED: Lazy<bool> = Lazy::new(|| {
    return ini_get::<bool>(INI_CONFIG);
});

#[inline]
pub fn is_enabled() -> bool {
    *IS_ENABLED
}
