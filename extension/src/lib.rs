mod execute;
mod ini;
mod request;
mod util;

use phper::{ini::Policy, modules::Module, php_get_module};

use crate::execute::register_exec_functions;

use crate::ini::{INI_ENABLED, INI_FUNCTION_THRESHOLD, INI_HEADER_KEY, INI_HEADER_MODE};

// This is the entrypoint of the PHP extension.
#[php_get_module]
pub fn get_module() -> Module {
    let mut module = Module::new(
        env!("CARGO_CRATE_NAME"),
        env!("CARGO_PKG_VERSION"),
        env!("CARGO_PKG_AUTHORS"),
    );

    module.add_ini(INI_ENABLED, false, Policy::All);
    module.add_ini(INI_HEADER_MODE, "".to_string(), Policy::All);
    module.add_ini(INI_FUNCTION_THRESHOLD, 100000, Policy::All);
    module.add_ini(INI_HEADER_KEY, "".to_string(), Policy::All);

    module.on_module_init(on_module_init);

    module.on_request_init(on_request_init);
    module.on_request_shutdown(on_request_shutdown);

    module
}

pub fn on_module_init() {
    if !ini::is_enabled() {
        return;
    }

    register_exec_functions();
}

pub fn on_request_init() {
    if !ini::is_enabled() {
        return;
    }

    request::init();
}

pub fn on_request_shutdown() {
    if !ini::is_enabled() {
        return;
    }

    request::shutdown();
}
