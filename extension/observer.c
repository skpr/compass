#ifdef HAVE_CONFIG_H
# include "config.h"
#endif

#include "php.h"
#include "ext/standard/info.h"
#include "Zend/zend_observer.h"
#include "php_observer.h"
#include "utils.c"

ZEND_DECLARE_MODULE_GLOBALS(observer)

static zend_never_inline void compass_request_init() {
	if (OBSERVER_G(debug) == 0) {
		return;
	}
	
	php_printf("[FUNCTION BEGIN\n");
}

static zend_never_inline void compass_request_shutdown() {
	if (OBSERVER_G(debug) == 0) {
		return;
	}
	
	php_printf("[FUNCTION BEGIN\n");
}

static zend_never_inline void compass_function_begin(zend_string * hash_code, zend_string * function_name) {
	if (OBSERVER_G(debug) == 0) {
		return;
	}
	
	php_printf("[FUNCTION BEGIN %s | %s]\n", hash_code, function_name);
}

static zend_never_inline void compass_function_end(zend_string * hash_code, zend_string * function_name) {
	if (OBSERVER_G(debug) == 0) {
		return;
	}

	php_printf("[FUNCTION END %s | %s]\n", hash_code, function_name);
}

static void handler_begin(zend_execute_data *execute_data) {
	zend_string *function_name;
	zend_string *hash_code;

	function_name = compass_get_function_name(execute_data);

	hash_code = strpprintf(0, "%p", execute_data);

	compass_begin(ZSTR_VAL(hash_code), ZSTR_VAL(function_name));
}

static void handler_end(zend_execute_data *execute_data, zval *return_value) {
	zend_string *function_name;
	zend_string *hash_code;

	function_name = compass_get_function_name(execute_data);

	hash_code = strpprintf(0, "%p", execute_data);

	compass_end(ZSTR_VAL(hash_code), ZSTR_VAL(function_name));
}

// Runs once per zend_function on its first call
static zend_observer_fcall_handlers observer_instrument(zend_execute_data *execute_data) {
	zend_observer_fcall_handlers handlers = {NULL, NULL};

	if (OBSERVER_G(instrument) == 0 ||
		!execute_data->func ||
		!execute_data->func->common.function_name) {
		return handlers; // I have no handlers for this function
	}

	handlers.begin = handler_begin;
	handlers.end = handler_end;

	return handlers; // I have handlers for this function
}

static void php_observer_init_globals(zend_observer_globals *observer_globals)
{
	observer_globals->instrument = 0;
	observer_globals->debug = 0;
}

PHP_INI_BEGIN()
	STD_PHP_INI_BOOLEAN("compass.instrument", "0", PHP_INI_SYSTEM, OnUpdateBool, instrument, zend_observer_globals, observer_globals)
	STD_PHP_INI_BOOLEAN("compass.debug", "0", PHP_INI_SYSTEM, OnUpdateBool, debug, zend_observer_globals, observer_globals)
PHP_INI_END()

static PHP_MINIT_FUNCTION(observer)
{
	ZEND_INIT_MODULE_GLOBALS(observer, php_observer_init_globals, NULL);
	REGISTER_INI_ENTRIES();
	zend_observer_fcall_register(observer_instrument);
	return SUCCESS;
}

static PHP_RINIT_FUNCTION(observer)
{
#if defined(ZTS) && defined(COMPILE_DL_OBSERVER)
	ZEND_TSRMLS_CACHE_UPDATE();
#endif
	compass_request_init();
	return SUCCESS;
}

static PHP_RSHUTDOWN_FUNCTION(observer)
{
#if defined(ZTS) && defined(COMPILE_DL_OBSERVER)
	ZEND_TSRMLS_CACHE_UPDATE();
#endif
	compass_request_shutdown();
	return SUCCESS;
}

static PHP_MINFO_FUNCTION(observer)
{
	php_info_print_table_start();
	php_info_print_table_header(2, "Observer PoC extension", "enabled");
	php_info_print_table_end();

	DISPLAY_INI_ENTRIES();
}

zend_module_entry observer_module_entry = {
	STANDARD_MODULE_HEADER,
	"compass",
	NULL,
	PHP_MINIT(observer),
	NULL,
	PHP_RINIT(observer),
	PHP_RSHUTDOWN(observer),
	PHP_MINFO(observer),
	PHP_OBSERVER_VERSION,
	STANDARD_MODULE_PROPERTIES
};

#ifdef COMPILE_DL_OBSERVER
# ifdef ZTS
ZEND_TSRMLS_CACHE_DEFINE()
# endif
ZEND_GET_MODULE(observer)
#endif
