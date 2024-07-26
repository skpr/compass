mod execute;
mod request;
mod util;

use phper::{
    ini::{ini_get, Policy},
    modules::Module,
    php_get_module,
};

use once_cell::sync::Lazy;

use crate::execute::register_exec_functions;

// Used to enable Compass.
const COMPASS_TRACE_ENABLED: &str = "compass.enabled";

// This is the entrypoint of the PHP extension.
#[php_get_module]
pub fn get_module() -> Module {
    let mut module = Module::new(
        env!("CARGO_CRATE_NAME"),
        env!("CARGO_PKG_VERSION"),
        env!("CARGO_PKG_AUTHORS"),
    );

    module.add_ini(COMPASS_TRACE_ENABLED, false, Policy::All);

    module.on_module_init(on_module_init);

    module.on_request_shutdown(on_request_shutdown);

    module
}

pub fn on_module_init() {
    if !is_enabled() {
        return;
    }

    register_exec_functions();
}

pub fn on_request_shutdown() {
    if !is_enabled() {
        return;
    }

    request::shutdown();
}

static IS_ENABLED: Lazy<bool> = Lazy::new(|| {
    return ini_get::<bool>(COMPASS_TRACE_ENABLED);
});

#[inline]
pub fn is_enabled() -> bool {
    *IS_ENABLED
}
