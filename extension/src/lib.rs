mod enabled;
mod observer;
mod request;
mod threshold;
mod util;

use phper::{ini::Policy, modules::Module, php_get_module, sys};
use probe::probe_lazy;

// This is the entrypoint of the PHP extension.
#[php_get_module]
pub fn get_module() -> Module {
    let mut module = Module::new(
        env!("CARGO_CRATE_NAME"),
        env!("CARGO_PKG_VERSION"),
        env!("CARGO_PKG_AUTHORS"),
    );

    module.add_ini(enabled::INI_CONFIG, false, Policy::All);
    module.add_ini(threshold::INI_CONFIG, 1000000, Policy::All);

    module.on_module_init(on_module_init);

    module.on_request_init(on_request_init);
    module.on_request_shutdown(on_request_shutdown);

    module
}

pub fn on_module_init() {
    if !enabled::is_enabled() {
        return;
    }

    unsafe {
        sys::zend_observer_fcall_register(Some(observer::observer_instrument));
    }
}

pub fn on_request_init() {
    if !enabled::is_enabled() {
        return;
    }

    let probe_enabled = probe_lazy!(skpr, enable_request_init);

    if !probe_enabled {
        return;
    }

    request::init();
}

pub fn on_request_shutdown() {
    if !enabled::is_enabled() {
        return;
    }

    let probe_enabled = probe_lazy!(skpr, enable_request_shutdown);

    if !probe_enabled {
        return;
    }

    request::shutdown();
}
