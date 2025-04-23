use once_cell::sync::Lazy;
use std::sync::RwLock;

#[derive(Debug, Clone, serde::Deserialize, serde::Serialize)]
pub struct Storage {
    pub enabled: bool,
    pub threshold: u64,
}

pub static STORAGE: Lazy<RwLock<Storage>> = Lazy::new(|| {
    RwLock::new(Storage {
        enabled: false,
        threshold: 0,
    })
});

pub fn is_enabled() -> bool {
    STORAGE.read().unwrap().enabled
}

pub fn get_threshold() -> u64 {
    STORAGE.read().unwrap().threshold
}
