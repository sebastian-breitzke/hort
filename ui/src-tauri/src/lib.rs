use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::io::{BufRead, BufReader, Write};
use std::path::PathBuf;
use std::process::Command;

#[cfg(unix)]
use std::os::unix::net::UnixStream;

/// Path to the daemon's Unix socket: honour HORT_SOCKET_PATH first (matches
/// the Go daemon's override), then fall back to ~/.hort/daemon.sock.
fn socket_path() -> PathBuf {
    if let Ok(override_path) = std::env::var("HORT_SOCKET_PATH") {
        return PathBuf::from(override_path);
    }
    let mut p = dirs::home_dir().unwrap_or_default();
    p.push(".hort");
    p.push("daemon.sock");
    p
}

#[derive(Deserialize, Serialize)]
struct DaemonRequest {
    method: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    params: Option<Value>,
}

#[derive(Deserialize)]
struct DaemonResponse {
    ok: bool,
    #[serde(default)]
    error: String,
    #[serde(default)]
    result: Option<Value>,
}

/// Send one JSON line to the daemon, read one JSON line back.
fn daemon_call(method: &str, params: Value) -> Result<Value, String> {
    #[cfg(not(unix))]
    {
        let _ = (method, params);
        return Err("daemon not supported on this platform".to_string());
    }
    #[cfg(unix)]
    {
        let path = socket_path();
        let mut stream =
            UnixStream::connect(&path).map_err(|e| format!("connect {}: {}", path.display(), e))?;
        let req = DaemonRequest {
            method: method.to_string(),
            params: if params.is_null() { None } else { Some(params) },
        };
        let mut body = serde_json::to_vec(&req).map_err(|e| e.to_string())?;
        body.push(b'\n');
        stream.write_all(&body).map_err(|e| e.to_string())?;
        stream.flush().map_err(|e| e.to_string())?;

        let mut reader = BufReader::new(stream);
        let mut line = String::new();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;
        if line.is_empty() {
            return Err("empty daemon response".to_string());
        }
        let resp: DaemonResponse = serde_json::from_str(&line).map_err(|e| e.to_string())?;
        if !resp.ok {
            return Err(resp.error);
        }
        Ok(resp.result.unwrap_or(Value::Null))
    }
}

/// Shape the generic `list`/`describe` params, then unwrap the daemon's map.
#[tauri::command]
fn hort_rpc(method: String, params: Value) -> Result<Value, String> {
    daemon_call(&method, params)
}

/// Run the hort CLI for side-effect commands the daemon doesn't own (unlock,
/// lock, daemon start, source mount/unmount). We don't shell out for reads.
#[tauri::command]
fn hort_shell(args: Vec<String>, passphrase: Option<String>) -> Result<String, String> {
    let mut cmd = Command::new("hort");
    cmd.args(&args);
    if let Some(pw) = passphrase {
        cmd.env("HORT_PASSPHRASE", pw);
    }
    let output = cmd
        .output()
        .map_err(|e| format!("exec hort: {}. Is `hort` on PATH?", e))?;
    if !output.status.success() {
        let err = String::from_utf8_lossy(&output.stderr).trim().to_string();
        return Err(if err.is_empty() {
            format!("hort exited with status {}", output.status)
        } else {
            err
        });
    }
    Ok(String::from_utf8_lossy(&output.stdout).into_owned())
}

#[tauri::command]
fn hort_copy(text: String) -> Result<(), String> {
    let mut cb = arboard::Clipboard::new().map_err(|e| e.to_string())?;
    cb.set_text(text).map_err(|e| e.to_string())
}

/// Post-init reachability ping — the frontend also checks via hort_rpc and
/// shows a one-shot "starting daemon…" toast if unreachable.
#[tauri::command]
fn hort_ping() -> bool {
    daemon_call("status", json!({})).is_ok()
}

pub fn run() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            hort_rpc, hort_shell, hort_copy, hort_ping
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
