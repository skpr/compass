use crate::util::get_sapi_module_name;

use once_cell::sync::Lazy;

static IS_FPM: Lazy<bool> = Lazy::new(|| get_sapi_module_name().to_bytes() == b"fpm-fcgi");

#[inline]
pub fn is_fpm() -> bool {
    *IS_FPM
}
