use crate::util::{
    get_request_id, get_request_method, get_request_server, get_request_uri, jit_initialization,
};

use crate::fpm::is_fpm;

use probe::probe_lazy;
use tracing::error;

pub fn init() {
    if !is_fpm() {
        return;
    }

    jit_initialization();

    let server_result = get_request_server();

    let server = match server_result {
        Ok(carrier) => carrier,
        Err(_err) => {
            error!("unable to get server info: {}", _err);
            return;
        }
    };

    let request_id = get_request_id(server);
    let uri = get_request_uri(server);
    let method = get_request_method(server);

    probe_lazy!(
        compass,
        request_init,
        request_id.as_ptr(),
        uri.as_ptr(),
        method.as_ptr()
    );
}

pub fn shutdown() {
    if !is_fpm() {
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

    let request_id = get_request_id(server);

    probe_lazy!(compass, request_shutdown, request_id.as_ptr());
}
