// Inspired by: https://github.com/longxinH/xhprof/blob/master/extension/trace.h#L54
static zend_always_inline zend_string *compass_get_function_name(zend_execute_data *execute_data)
{
    zend_function *curr_func;
    zend_string *real_function_name;

    if (!execute_data) {
        return NULL;
    }

    curr_func = execute_data->func;

    if (!curr_func->common.function_name) {
        return NULL;
    }

    if (curr_func->common.scope != NULL) {
        real_function_name = strpprintf(0, "%s::%s", curr_func->common.scope->name->val, ZSTR_VAL(curr_func->common.function_name));
    } else {
        real_function_name = zend_string_copy(curr_func->common.function_name);
    }

    return real_function_name;
}
