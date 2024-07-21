use phper::{strings::ZStr, values::ExecuteData};

pub fn get_function_and_class_name(
    execute_data: &mut ExecuteData,
) -> anyhow::Result<(Option<String>, Option<String>)> {
    let function = execute_data.func();

    let function_name = function
        .get_function_name()
        .map(ZStr::to_str)
        .transpose()?
        .map(ToOwned::to_owned);
    let class_name = function
        .get_class()
        .map(|cls| cls.get_name().to_str().map(ToOwned::to_owned))
        .transpose()?;

    Ok((function_name, class_name))
}

pub fn get_combined_name(class_name: String, function_name: String) -> String {
    if class_name != "" && function_name != "" {
        return format!("{}::{}", class_name, function_name).to_string();
    }

    if class_name != "" {
        return class_name;
    }

    return function_name;
}
