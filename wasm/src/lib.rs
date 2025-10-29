use wasm_bindgen::prelude::*;
use hmac::{Hmac, Mac};
use sha2::Sha256;
use base64::{engine::general_purpose, Engine as _};
use serde_json::json;

// HMAC-SHA256
type HmacSha256 = Hmac<Sha256>;

// CLIENT_SECRET embarcado (WARNING: extraível por um atacante avançado)
const CLIENT_SECRET: &str = "client-embedded-secret-CHANGE_ME";

#[wasm_bindgen]
pub fn process(payload_b64: &str, key_hint: &str) -> String {
    // derive key: HMAC(CLIENT_SECRET, key_hint)
    let mut mac = HmacSha256::new_from_slice(CLIENT_SECRET.as_bytes()).unwrap();
    mac.update(key_hint.as_bytes());
    let key = mac.finalize().into_bytes();

    // decode payload
    let enc = general_purpose::STANDARD.decode(payload_b64).unwrap_or_default();
    let plain = xor_with_key(&enc, &key);

    // parse payload JSON
    let payload_json: serde_json::Value = serde_json::from_slice(&plain).unwrap_or(json!({}));

    // --- demo processing: iterar 0..9999, converter e checar se existe no set ---
    let mut found = false;
    if let Some(set) = payload_json.get("set") {
        if let Some(arr) = set.as_array() {
            use std::collections::HashSet;
            let s: HashSet<String> = arr.iter().filter_map(|v| v.as_str().map(|s| s.to_string())).collect();
            for i in 0..10000 {
                let si = i.to_string();
                if s.contains(&si) {
                    found = true;
                    break;
                }
            }
        }
    }

    // resultado JSON
    let res = json!({"found": found});
    let res_bytes = serde_json::to_vec(&res).unwrap();
    let out = xor_with_key(&res_bytes, &key);
    general_purpose::STANDARD.encode(out)
}

fn xor_with_key(data: &[u8], key: &[u8]) -> Vec<u8> {
    let mut out = vec![0u8; data.len()];
    for i in 0..data.len() {
        out[i] = data[i] ^ key[i % key.len()];
    }
    out
}