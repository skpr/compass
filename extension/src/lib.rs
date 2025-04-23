mod config;
mod observer;
mod request;
mod util;

use phper::{ini::ini_get, ini::Policy, modules::Module, php_get_module, sys};

pub const INI_ENABLED: &str = "compass.enabled";
pub const INI_THRESHOLD: &str = "compass.threshold";

// This is the entrypoint of the PHP extension.
#[php_get_module]
pub fn get_module() -> Module {
    let mut module = Module::new(
        env!("CARGO_CRATE_NAME"),
        env!("CARGO_PKG_VERSION"),
        env!("CARGO_PKG_AUTHORS"),
    );

    module.add_ini(INI_ENABLED, false, Policy::All);
    module.add_ini(INI_THRESHOLD, 1000000, Policy::All);

    module.on_module_init(on_module_init);

    module.on_request_init(on_request_init);
    module.on_request_shutdown(on_request_shutdown);

    module
}

pub fn on_module_init() {
    {
        let mut cfg = config::STORAGE.write().unwrap();
        cfg.enabled = ini_get::<bool>(INI_ENABLED);
        cfg.threshold = ini_get::<i64>(INI_THRESHOLD) as u64;
    }

    unsafe {
        sys::zend_observer_fcall_register(Some(observer::observer_instrument));
    }
}

pub fn on_request_init() {
    if !config::is_enabled() {
        return;
    }

    request::init();
}

pub fn on_request_shutdown() {
    if !config::is_enabled() {
        return;
    }

    request::shutdown();
}
