use crate::util::{get_request_id, get_request_server, get_sapi_module_name, jit_initialization};

use probe::probe;

pub fn init() {
    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        return;
    }

    jit_initialization();
}

pub fn shutdown() {
    if get_sapi_module_name().to_bytes() != b"fpm-fcgi" {
        return;
    }

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        // @todo, This should not panic.
        Err(error) => panic!("Problem getting the server: {:?}", error),
    };

    let request_id = get_request_id(server);

    probe!(compass, request_shutdown, request_id.as_ptr());
}