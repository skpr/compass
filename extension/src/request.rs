use crate::util::{
    get_header_key, get_request_id, get_request_method, get_request_server, get_request_uri,
    get_sapi_module_name, jit_initialization,
};

use crate::header;
use probe::probe;
use tracing::error;

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
        Err(_err) => {
            error!("unable to get server info: {}", _err);
            return;
        }
    };

    if header::block_execution(get_header_key(server)) {
        return;
    }

    let request_id = get_request_id(server);
    let uri = get_request_uri(server);
    let method = get_request_method(server);

    probe!(
        compass,
        request_shutdown,
        request_id.as_ptr(),
        uri.as_ptr(),
        method.as_ptr()
    );
}
