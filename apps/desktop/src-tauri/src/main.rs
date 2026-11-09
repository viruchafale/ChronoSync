#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]
mod bridge;
mod key_custody;
#[cfg(test)]
mod protected_keychain;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use bridge::Bridge;
use key_custody::{CustodyError, KeyCustody, PlatformCustody};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::sync::{Arc, Mutex};
use tauri::Manager;
use tauri_plugin_dialog::{DialogExt, MessageDialogButtons};

const MAX_FIXTURE_BYTES: u64 = 2 * 1024 * 1024;
const MAX_EXCHANGE_BYTES: u64 = 8 * 1024 * 1024;

type Engine = Arc<Mutex<Option<Bridge>>>;
type Custody = Arc<dyn KeyCustody>;
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct StartSpec {
    workspace: String,
    prompt: String,
    codex_path: String,
    sandbox: String,
    timeout_seconds: u32,
    consent: bool,
    excludes: Vec<String>,
}
async fn request(
    app: tauri::AppHandle,
    state: Engine,
    method: &'static str,
    params: Value,
) -> Result<Value, String> {
    tauri::async_runtime::spawn_blocking(move || {
        let mut guard = state.lock().map_err(|_| "Engine lock unavailable")?;
        if guard.is_none() {
            let executable = app
                .path()
                .resource_dir()
                .map_err(|e| e.to_string())?
                .join("binaries")
                .join(if cfg!(windows) {
                    "chronosync-engine.exe"
                } else {
                    "chronosync-engine"
                });
            let data = app
                .path()
                .app_data_dir()
                .map_err(|e| e.to_string())?
                .join("recordings");
            *guard = Some(Bridge::start(&executable, &data)?);
        }
        guard
            .as_mut()
            .ok_or("Engine unavailable")?
            .request(method, params)
    })
    .await
    .map_err(|e| e.to_string())?
}
#[tauri::command]
async fn run_start(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    spec: StartSpec,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "run.start",
        serde_json::to_value(spec).map_err(|e| e.to_string())?,
    )
    .await
}
#[tauri::command]
async fn run_list(app: tauri::AppHandle, state: tauri::State<'_, Engine>) -> Result<Value, String> {
    request(app, state.inner().clone(), "run.list", json!({})).await
}
#[tauri::command]
async fn run_cancel(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "run.cancel", json!({"id":id})).await
}
#[tauri::command]
async fn event_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
    after: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "event.list",
        json!({"id":id,"after":after}),
    )
    .await
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct EventQuery {
    id: String,
    cursor: String,
    kind: String,
    search: String,
    min: String,
    max: String,
    limit: u32,
}
#[tauri::command]
async fn event_query(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    query: EventQuery,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "event.query",
        serde_json::to_value(query).map_err(|e| e.to_string())?,
    )
    .await
}
#[tauri::command]
async fn event_get(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
    event_id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "event.get",
        json!({"id":id,"event_id":event_id}),
    )
    .await
}
#[tauri::command]
async fn run_query(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    cursor: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "run.query",
        json!({"cursor":cursor,"limit":100}),
    )
    .await
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct CaptureSpec {
    run_id: String,
    branch_id: String,
    excludes: Vec<String>,
    consent: bool,
}
#[tauri::command]
async fn snapshot_capture(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    spec: CaptureSpec,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "snapshot.capture",
        serde_json::to_value(spec).map_err(|e| e.to_string())?,
    )
    .await
}
#[tauri::command]
async fn snapshot_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    run_id: String,
    after: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "snapshot.list",
        json!({"run_id":run_id,"after":after,"limit":50}),
    )
    .await
}
#[tauri::command]
async fn branch_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    run_id: String,
    after: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "branch.list",
        json!({"run_id":run_id,"after":after,"limit":50}),
    )
    .await
}
#[tauri::command]
async fn snapshot_validate(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "snapshot.validate",
        json!({"id":id}),
    )
    .await
}
#[tauri::command]
async fn snapshot_materialize(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    snapshot_id: String,
    branch: bool,
    consent: bool,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        if branch {
            "branch.create"
        } else {
            "snapshot.restore"
        },
        json!({"snapshot_id":snapshot_id,"consent":consent}),
    )
    .await
}
#[tauri::command]
async fn object_inventory(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "object.inventory", json!({})).await
}

#[tauri::command]
async fn fixture_eligibility(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    run_id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "fixture.eligibility",
        json!({"run_id":run_id}),
    )
    .await
}

#[tauri::command]
async fn fixture_create(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    run_id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "fixture.create",
        json!({"run_id":run_id}),
    )
    .await
}

#[tauri::command]
async fn fixture_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "fixture.list", json!({})).await
}

#[tauri::command]
async fn fixture_inspect(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "fixture.inspect",
        json!({"id":id}),
    )
    .await
}

fn read_fixture(path: &std::path::Path) -> Result<String, String> {
    let metadata = fs::metadata(path).map_err(|_| "Could not inspect selected fixture")?;
    if !metadata.is_file() || metadata.len() == 0 || metadata.len() > MAX_FIXTURE_BYTES {
        return Err("Selected fixture is not a bounded regular file".into());
    }
    let mut file = fs::File::open(path).map_err(|_| "Could not open selected fixture")?;
    let mut bytes = Vec::with_capacity(metadata.len() as usize);
    Read::by_ref(&mut file)
        .take(MAX_FIXTURE_BYTES + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| "Could not read selected fixture")?;
    if bytes.len() as u64 > MAX_FIXTURE_BYTES {
        return Err("Selected fixture exceeds the import limit".into());
    }
    String::from_utf8(bytes).map_err(|_| "Fixture must be canonical UTF-8 JSON".into())
}

#[tauri::command]
async fn fixture_import(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    let dialog_app = app.clone();
    let selected = tauri::async_runtime::spawn_blocking(move || {
        dialog_app
            .dialog()
            .file()
            .add_filter("ChronoSync fixture", &["fixture", "json"])
            .blocking_pick_file()
    })
    .await
    .map_err(|_| "Fixture picker failed")?;
    let Some(selected) = selected else {
        return Ok(json!({"cancelled":true}));
    };
    let path = selected
        .into_path()
        .map_err(|_| "Selected fixture is not a local file")?;
    let bytes = read_fixture(&path)?;
    request(
        app,
        state.inner().clone(),
        "fixture.import",
        json!({"bytes":bytes}),
    )
    .await
}

fn confirm(app: tauri::AppHandle, title: &'static str, message: String) -> bool {
    app.dialog()
        .message(message)
        .title(title)
        .buttons(MessageDialogButtons::OkCancel)
        .blocking_show()
}

fn registration_payload(public_key: &[u8]) -> Vec<u8> {
    let mut payload = b"chronosync-key-custody-registration-v1\0".to_vec();
    payload.extend_from_slice(public_key);
    payload
}

fn migration_proof_payload(key_id: &str, challenge: &[u8]) -> Vec<u8> {
    let mut payload = b"chronosync-key-migration-proof-v1\0".to_vec();
    payload.extend_from_slice(key_id.as_bytes());
    payload.push(0);
    payload.extend_from_slice(challenge);
    payload
}

#[tauri::command]
async fn fixture_export(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    let record = request(
        app.clone(),
        state.inner().clone(),
        "fixture.inspect",
        json!({"id":id}),
    )
    .await?;
    let warnings = record["warnings"]
        .as_array()
        .map(|items| {
            items
                .iter()
                .filter_map(Value::as_str)
                .collect::<Vec<_>>()
                .join("\n")
        })
        .unwrap_or_default();
    let confirm_app = app.clone();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Export regression fixture",
            format!("Review before sharing:\n{warnings}"),
        )
    })
    .await
    .map_err(|_| "Export confirmation failed")?;
    if !approved {
        return Ok(json!({"exported":false,"cancelled":true}));
    }
    let exported = request(
        app.clone(),
        state.inner().clone(),
        "fixture.export",
        json!({"id":id,"consent":true}),
    )
    .await?;
    let bytes = exported["bytes"]
        .as_str()
        .ok_or("Engine returned invalid fixture bytes")?
        .as_bytes()
        .to_vec();
    let dialog_app = app.clone();
    let file_name = format!("chronosync-{}.fixture", &id[..id.len().min(12)]);
    let selected = tauri::async_runtime::spawn_blocking(move || {
        dialog_app
            .dialog()
            .file()
            .add_filter("ChronoSync fixture", &["fixture"])
            .set_file_name(file_name)
            .blocking_save_file()
    })
    .await
    .map_err(|_| "Fixture save picker failed")?;
    let Some(selected) = selected else {
        return Ok(json!({"exported":false,"cancelled":true}));
    };
    let path = selected
        .into_path()
        .map_err(|_| "Export destination is not a local file")?;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&path)
        .map_err(|_| "Export destination already exists or is unavailable")?;
    file.write_all(&bytes)
        .and_then(|_| file.sync_all())
        .map_err(|_| "Could not durably write exported fixture")?;
    if let Some(parent) = path.parent() {
        if let Ok(directory) = fs::File::open(parent) {
            let _ = directory.sync_all();
        }
    }
    Ok(json!({"exported":true}))
}

#[tauri::command]
async fn fixture_delete(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let short_id = id.chars().take(12).collect::<String>();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Delete stored fixture",
            format!("Permanently delete fixture {short_id}?"),
        )
    })
    .await
    .map_err(|_| "Deletion confirmation failed")?;
    if !approved {
        return Ok(json!({"deleted":false,"cancelled":true}));
    }
    request(
        app,
        state.inner().clone(),
        "fixture.delete",
        json!({"id":id,"consent":true}),
    )
    .await
}

#[tauri::command]
async fn replay_evaluate(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    regression_id: String,
    transport_id: String,
    plan: Value,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "replay.evaluate",
        json!({"regression_id":regression_id,"transport_id":transport_id,"plan":plan}),
    )
    .await
}

#[tauri::command]
async fn analysis_compare(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    run_ids: Vec<String>,
    policy: Value,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "analysis.compare",
        json!({"run_ids":run_ids,"policy":policy}),
    )
    .await
}

#[tauri::command]
async fn analysis_report_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "analysis.report.list",
        json!({}),
    )
    .await
}

#[tauri::command]
async fn analysis_report_get(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "analysis.report.get",
        json!({"id":id}),
    )
    .await
}

#[tauri::command]
async fn analysis_report_reevaluate(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
    policy: Value,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "analysis.report.reevaluate",
        json!({"id":id,"policy":policy}),
    )
    .await
}

#[tauri::command]
async fn benchmark_create(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    spec: Value,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "benchmark.create", spec).await
}

fn read_benchmark(path: &std::path::Path) -> Result<String, String> {
    let metadata = fs::metadata(path).map_err(|_| "Could not inspect selected benchmark")?;
    if !metadata.is_file() || metadata.len() == 0 || metadata.len() > MAX_FIXTURE_BYTES {
        return Err("Selected benchmark is not a bounded regular file".into());
    }
    let mut file = fs::File::open(path).map_err(|_| "Could not open selected benchmark")?;
    let mut bytes = Vec::with_capacity(metadata.len() as usize);
    Read::by_ref(&mut file)
        .take(MAX_FIXTURE_BYTES + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| "Could not read selected benchmark")?;
    if bytes.len() as u64 > MAX_FIXTURE_BYTES {
        return Err("Selected benchmark exceeds the bounded size limit".into());
    }
    String::from_utf8(bytes).map_err(|_| "Benchmark must be canonical UTF-8 JSON".into())
}

#[tauri::command]
async fn benchmark_import(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    let dialog_app = app.clone();
    let selected = tauri::async_runtime::spawn_blocking(move || {
        dialog_app
            .dialog()
            .file()
            .add_filter("ChronoSync benchmark", &["benchmark", "json"])
            .blocking_pick_file()
    })
    .await
    .map_err(|_| "Benchmark picker failed")?;
    let Some(selected) = selected else {
        return Ok(json!({"cancelled":true}));
    };
    let path = selected
        .into_path()
        .map_err(|_| "Selected benchmark is not a local file")?;
    let bytes = read_benchmark(&path)?;
    request(
        app,
        state.inner().clone(),
        "benchmark.import",
        json!({"bytes":bytes}),
    )
    .await
}

#[tauri::command]
async fn benchmark_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "benchmark.list", json!({})).await
}

#[tauri::command]
async fn benchmark_get(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "benchmark.get",
        json!({"id":id}),
    )
    .await
}

#[tauri::command]
async fn benchmark_export(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Export benchmark dataset",
            "The dataset contains equality-revealing fingerprints and local evidence references. SHA-256 does not prove authorship. Review before sharing.".into(),
        )
    })
    .await
    .map_err(|_| "Benchmark export confirmation failed")?;
    if !approved {
        return Ok(json!({"exported":false,"cancelled":true}));
    }
    let exported = request(
        app.clone(),
        state.inner().clone(),
        "benchmark.export",
        json!({"id":id,"consent":true}),
    )
    .await?;
    let bytes = exported["bytes"]
        .as_str()
        .ok_or("Engine returned invalid benchmark bytes")?
        .as_bytes()
        .to_vec();
    let dialog_app = app.clone();
    let file_name = format!("chronosync-{}.benchmark", &id[..id.len().min(12)]);
    let selected = tauri::async_runtime::spawn_blocking(move || {
        dialog_app
            .dialog()
            .file()
            .add_filter("ChronoSync benchmark", &["benchmark"])
            .set_file_name(file_name)
            .blocking_save_file()
    })
    .await
    .map_err(|_| "Benchmark save picker failed")?;
    let Some(selected) = selected else {
        return Ok(json!({"exported":false,"cancelled":true}));
    };
    let path = selected
        .into_path()
        .map_err(|_| "Benchmark destination is not a local file")?;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&path)
        .map_err(|_| "Benchmark destination already exists or is unavailable")?;
    file.write_all(&bytes)
        .and_then(|_| file.sync_all())
        .map_err(|_| "Could not durably write benchmark")?;
    if let Some(parent) = path.parent() {
        if let Ok(directory) = fs::File::open(parent) {
            let _ = directory.sync_all();
        }
    }
    Ok(json!({"exported":true}))
}

#[tauri::command]
async fn benchmark_compare(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    baseline_id: String,
    candidate_id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "benchmark.compare",
        json!({"baseline_id":baseline_id,"candidate_id":candidate_id}),
    )
    .await
}

#[tauri::command]
async fn benchmark_evaluate(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
    policy: Value,
    parent_lineage_id: String,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "benchmark.evaluate",
        json!({"id":id,"policy":policy,"parent_lineage_id":parent_lineage_id}),
    )
    .await
}

#[tauri::command]
async fn lineage_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "lineage.list", json!({})).await
}

#[tauri::command]
async fn lineage_get(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    id: String,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "lineage.get", json!({"id":id})).await
}

#[tauri::command]
async fn retention_set(
    app: tauri::AppHandle,
    engine: tauri::State<'_, Engine>,
    kind: String,
    id: String,
    state: String,
    policy: Value,
    reason: String,
) -> Result<Value, String> {
    request(
        app,
        engine.inner().clone(),
        "retention.set",
        json!({"kind":kind,"id":id,"state":state,"policy":policy,"reason":reason}),
    )
    .await
}

#[tauri::command]
async fn governance_audit(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "governance.audit", json!({})).await
}

#[tauri::command]
async fn governance_delete(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    kind: String,
    id: String,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let message = format!(
        "Permanently delete {kind} {}? Referenced objects are refused and no dependent is deleted automatically.",
        id.chars().take(12).collect::<String>()
    );
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(confirm_app, "Delete governed evidence", message)
    })
    .await
    .map_err(|_| "Deletion confirmation failed")?;
    if !approved {
        return Ok(json!({"deleted":false,"cancelled":true}));
    }
    request(
        app,
        state.inner().clone(),
        "governance.delete",
        json!({"kind":kind,"id":id,"consent":true}),
    )
    .await
}

fn read_exchange(path: &std::path::Path) -> Result<Vec<u8>, String> {
    let metadata = fs::symlink_metadata(path).map_err(|_| "Could not inspect selected package")?;
    if metadata.file_type().is_symlink()
        || !metadata.is_file()
        || metadata.len() == 0
        || metadata.len() > MAX_EXCHANGE_BYTES
    {
        return Err("Selected package is not a bounded regular file".into());
    }
    let mut file = fs::File::open(path).map_err(|_| "Could not open selected package")?;
    let opened = file
        .metadata()
        .map_err(|_| "Could not revalidate selected package")?;
    if !opened.is_file() || opened.len() != metadata.len() {
        return Err("Selected package changed during validation".into());
    }
    let mut bytes = Vec::with_capacity(metadata.len() as usize);
    Read::by_ref(&mut file)
        .take(MAX_EXCHANGE_BYTES + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| "Could not read selected package")?;
    if bytes.len() as u64 > MAX_EXCHANGE_BYTES {
        return Err("Selected package exceeds the bounded size limit".into());
    }
    Ok(bytes)
}

async fn pick_exchange(app: tauri::AppHandle) -> Result<Option<Vec<u8>>, String> {
    let selected = tauri::async_runtime::spawn_blocking(move || {
        app.dialog()
            .file()
            .add_filter("ChronoSync signed exchange", &["chronosync-exchange"])
            .blocking_pick_file()
    })
    .await
    .map_err(|_| "Package picker failed")?;
    let Some(selected) = selected else {
        return Ok(None);
    };
    let path = selected
        .into_path()
        .map_err(|_| "Selected package is not a local file")?;
    read_exchange(&path).map(Some)
}

#[tauri::command]
async fn identity_create(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    custody: tauri::State<'_, Custody>,
    label: String,
    supersedes: String,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let shown_label = label.clone();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Create local signing identity",
            format!(
                "Create signing identity ‘{shown_label}’ in the macOS login Keychain? Ed25519 is stored as a generic-password secret, not a Secure Enclave or hardware-backed key. This development build cannot enforce per-sign user presence."
            ),
        )
    })
    .await
    .map_err(|_| "Identity confirmation failed")?;
    if !approved {
        return Ok(json!({"cancelled":true}));
    }
    let create_custody = custody.inner().clone();
    let created = tauri::async_runtime::spawn_blocking(move || create_custody.create())
        .await
        .map_err(|_| "Keychain creation task failed")?
        .map_err(|error| error.to_string())?;
    let proof_custody = custody.inner().clone();
    let proof_key_id = created.key_id.clone();
    let proof_payload = registration_payload(&created.public_key);
    let proof = tauri::async_runtime::spawn_blocking(move || {
        proof_custody.sign(&proof_key_id, &proof_payload)
    })
    .await
    .map_err(|_| "Keychain proof task failed")?
    .map_err(|error| error.to_string())?;
    let result = request(
        app,
        state.inner().clone(),
        "identity.register_os",
        json!({"label":label,"supersedes":supersedes,"public_key":BASE64.encode(created.public_key),"proof":BASE64.encode(proof)}),
    )
    .await;
    if result
        .as_ref()
        .is_ok_and(|value| value["identity"]["key_id"].as_str() != Some(created.key_id.as_str()))
    {
        let cleanup = custody.inner().clone();
        let cleanup_key_id = created.key_id;
        let _ = tauri::async_runtime::spawn_blocking(move || cleanup.delete(&cleanup_key_id)).await;
        return Err("Engine returned a mismatched custody identity".into());
    }
    if result.is_err() {
        let cleanup = custody.inner().clone();
        let cleanup_key_id = created.key_id;
        let _ = tauri::async_runtime::spawn_blocking(move || cleanup.delete(&cleanup_key_id)).await;
    }
    result
}

#[tauri::command]
async fn identity_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(app, state.inner().clone(), "identity.list", json!({})).await
}

#[tauri::command]
async fn identity_trust(
    app: tauri::AppHandle,
    engine: tauri::State<'_, Engine>,
    key_id: String,
    trust_state: String,
    reason: String,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let short_id = key_id.chars().take(16).collect::<String>();
    let shown_state = trust_state.clone();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Change signing-identity trust",
            format!(
                "Set identity {short_id} to {shown_state}? Trust does not establish package safety. Revocation is permanent in the local trust history."
            ),
        )
    })
    .await
    .map_err(|_| "Trust confirmation failed")?;
    if !approved {
        return Ok(json!({"cancelled":true}));
    }
    request(
        app,
        engine.inner().clone(),
        "identity.trust",
        json!({"key_id":key_id,"state":trust_state,"reason":reason}),
    )
    .await
}

#[tauri::command]
async fn identity_migration_list(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "identity.migration.list",
        json!({}),
    )
    .await
}

async fn record_migration_failure(
    app: tauri::AppHandle,
    engine: Engine,
    key_id: &str,
    error: &CustodyError,
) {
    let recovery_required = matches!(
        error,
        CustodyError::Corrupt | CustodyError::Conflict | CustodyError::Backend(_)
    );
    let _ = request(
        app,
        engine,
        "identity.migration.fail",
        json!({"key_id":key_id,"reason":error.to_string(),"recovery_required":recovery_required}),
    )
    .await;
}

#[tauri::command]
async fn identity_migrate(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    custody: tauri::State<'_, Custody>,
    key_id: String,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let short_id = key_id.chars().take(16).collect::<String>();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Migrate signing key to macOS Keychain",
            format!("Migrate identity {short_id}? ChronoSync will verify a Keychain signature before retiring the legacy seed. This login-Keychain mode is not device-only, but access can still be lost after account, device, or application changes and cannot be recovered from the public identity."),
        )
    })
    .await
    .map_err(|_| "Migration confirmation failed")?;
    if !approved {
        return Ok(json!({"cancelled":true}));
    }
    let begun = request(
        app.clone(),
        state.inner().clone(),
        "identity.migration.begin",
        json!({"key_id":key_id}),
    )
    .await?;
    let migration_state = begun["record"]["state"].as_str().unwrap_or("");
    if migration_state == "migrated" {
        return Ok(begun["record"].clone());
    }
    if migration_state != "identity_verified" && migration_state != "legacy_retirement_pending" {
        let seed = zeroize::Zeroizing::new(
            BASE64
                .decode(
                    begun["seed_base64"]
                        .as_str()
                        .ok_or("Engine omitted legacy seed")?,
                )
                .map_err(|_| "Engine returned invalid legacy seed encoding")?,
        );
        if seed.len() != 32 {
            return Err("Engine returned invalid legacy seed length".into());
        }
        let import_custody = custody.inner().clone();
        let import_key_id = key_id.clone();
        let imported_public = tauri::async_runtime::spawn_blocking(move || {
            import_custody.import(&import_key_id, &seed)
        })
        .await
        .map_err(|_| "Keychain migration task failed")?;
        let imported_public = match imported_public {
            Ok(public) => public,
            Err(error) => {
                record_migration_failure(app.clone(), state.inner().clone(), &key_id, &error).await;
                return Err(error.to_string());
            }
        };
        let challenge = BASE64
            .decode(
                begun["challenge_base64"]
                    .as_str()
                    .ok_or("Engine omitted migration challenge")?,
            )
            .map_err(|_| "Engine returned invalid migration challenge")?;
        let payload = migration_proof_payload(&key_id, &challenge);
        let proof_custody = custody.inner().clone();
        let proof_key_id = key_id.clone();
        let signature = tauri::async_runtime::spawn_blocking(move || {
            proof_custody.sign(&proof_key_id, &payload)
        })
        .await
        .map_err(|_| "Keychain migration proof task failed")?;
        let signature = match signature {
            Ok(signature) => signature,
            Err(error) => {
                record_migration_failure(app.clone(), state.inner().clone(), &key_id, &error).await;
                return Err(error.to_string());
            }
        };
        if let Err(error) = request(
            app.clone(),
            state.inner().clone(),
            "identity.migration.verify",
            json!({"key_id":key_id,"public_key":BASE64.encode(imported_public),"signature":BASE64.encode(signature)}),
        )
        .await
        {
            let _ = request(
                app.clone(),
                state.inner().clone(),
                "identity.migration.fail",
                json!({"key_id":key_id,"reason":"native proof rejected","recovery_required":true}),
            )
            .await;
            return Err(error);
        }
    }
    request(
        app,
        state.inner().clone(),
        "identity.migration.retire",
        json!({"key_id":key_id}),
    )
    .await
}

#[tauri::command]
async fn identity_custody_status(
    custody: tauri::State<'_, Custody>,
    key_id: String,
) -> Result<Value, String> {
    let inspect = custody.inner().clone();
    let status = tauri::async_runtime::spawn_blocking(move || inspect.status(&key_id))
        .await
        .map_err(|_| "Keychain status task failed")?;
    serde_json::to_value(status).map_err(|error| error.to_string())
}

#[tauri::command]
async fn identity_trust_audit(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    request(
        app,
        state.inner().clone(),
        "identity.trust.audit",
        json!({}),
    )
    .await
}

#[tauri::command]
async fn exchange_export(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
    custody: tauri::State<'_, Custody>,
    dataset_content_id: String,
    identity_key_id: String,
    include_report_ids: Vec<String>,
) -> Result<Value, String> {
    let confirm_app = app.clone();
    let report_count = include_report_ids.len();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Sign and export benchmark package",
            format!(
                "Export one benchmark and {report_count} selected reports? Reports contain equality-revealing fingerprints and local identifiers. No raw command/output, credentials, environment, workspace path, or private key is intentionally included."
            ),
        )
    })
    .await
    .map_err(|_| "Exchange export confirmation failed")?;
    if !approved {
        return Ok(json!({"cancelled":true}));
    }
    let begun = request(
        app.clone(),
        state.inner().clone(),
        "exchange.export.begin",
        json!({"dataset_content_id":dataset_content_id,"identity_key_id":identity_key_id,"include_report_ids":include_report_ids}),
    )
    .await?;
    let mut exported = if begun["mode"] == "legacy_completed" {
        begun["completed"].clone()
    } else if begun["mode"] == "native_sign" {
        let session_id = begun["session_id"]
            .as_str()
            .ok_or("Engine omitted signing session")?
            .to_string();
        let key_id = begun["key_id"]
            .as_str()
            .ok_or("Engine omitted signing identity")?
            .to_string();
        let payload = BASE64
            .decode(
                begun["payload_base64"]
                    .as_str()
                    .ok_or("Engine omitted signing payload")?,
            )
            .map_err(|_| "Engine returned invalid signing payload")?;
        let sign_custody = custody.inner().clone();
        let signature =
            tauri::async_runtime::spawn_blocking(move || sign_custody.sign(&key_id, &payload))
                .await
                .map_err(|_| "Keychain signing task failed")?
                .map_err(|error| error.to_string())?;
        request(
            app.clone(),
            state.inner().clone(),
            "exchange.export.finish",
            json!({"session_id":session_id,"signature":BASE64.encode(signature)}),
        )
        .await?
    } else {
        return Err("Engine returned unsupported signing mode".into());
    };
    let encoded = exported["bytes_base64"]
        .as_str()
        .ok_or("Engine returned invalid exchange bytes")?;
    let bytes = BASE64
        .decode(encoded)
        .map_err(|_| "Engine returned invalid exchange encoding")?;
    if bytes.is_empty() || bytes.len() as u64 > MAX_EXCHANGE_BYTES {
        return Err("Engine returned an out-of-bounds package".into());
    }
    let package_id = exported["package_id"].as_str().unwrap_or("package");
    let file_name = format!(
        "chronosync-{}.chronosync-exchange",
        package_id.chars().take(12).collect::<String>()
    );
    let dialog_app = app.clone();
    let selected = tauri::async_runtime::spawn_blocking(move || {
        dialog_app
            .dialog()
            .file()
            .add_filter("ChronoSync signed exchange", &["chronosync-exchange"])
            .set_file_name(file_name)
            .blocking_save_file()
    })
    .await
    .map_err(|_| "Exchange save picker failed")?;
    let Some(selected) = selected else {
        return Ok(json!({"cancelled":true}));
    };
    let path = selected
        .into_path()
        .map_err(|_| "Exchange destination is not a local file")?;
    let mut file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&path)
        .map_err(|_| "Exchange destination already exists or is unavailable")?;
    file.write_all(&bytes)
        .and_then(|_| file.sync_all())
        .map_err(|_| "Could not durably write exchange package")?;
    if let Some(parent) = path.parent() {
        if let Ok(directory) = fs::File::open(parent) {
            let _ = directory.sync_all();
        }
    }
    if let Some(object) = exported.as_object_mut() {
        object.remove("bytes_base64");
        object.insert("exported".into(), json!(true));
    }
    Ok(exported)
}

#[tauri::command]
async fn exchange_inspect(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    let Some(bytes) = pick_exchange(app.clone()).await? else {
        return Ok(json!({"cancelled":true}));
    };
    request(
        app,
        state.inner().clone(),
        "exchange.inspect",
        json!({"bytes_base64":BASE64.encode(bytes)}),
    )
    .await
}

#[tauri::command]
async fn exchange_import(
    app: tauri::AppHandle,
    state: tauri::State<'_, Engine>,
) -> Result<Value, String> {
    let Some(bytes) = pick_exchange(app.clone()).await? else {
        return Ok(json!({"cancelled":true}));
    };
    let encoded = BASE64.encode(&bytes);
    let verification = request(
        app.clone(),
        state.inner().clone(),
        "exchange.inspect",
        json!({"bytes_base64":encoded}),
    )
    .await?;
    let status = verification["status"]
        .as_str()
        .unwrap_or("malformed_package");
    if status != "valid_trusted" && status != "valid_untrusted" {
        return Ok(json!({"imported":false,"verification":verification}));
    }
    let confirm_app = app.clone();
    let shown_status = status.to_string();
    let approved = tauri::async_runtime::spawn_blocking(move || {
        confirm(
            confirm_app,
            "Import signed benchmark package",
            format!(
                "Verification outcome: {shown_status}. Import immutable reports and benchmark evidence? A valid or trusted signature does not establish safety, quality, or execution authorization."
            ),
        )
    })
    .await
    .map_err(|_| "Exchange import confirmation failed")?;
    if !approved {
        return Ok(json!({"cancelled":true,"verification":verification}));
    }
    request(
        app,
        state.inner().clone(),
        "exchange.import",
        json!({"bytes_base64":encoded,"allow_untrusted":status=="valid_untrusted"}),
    )
    .await
}
fn main() {
    let engine: Engine = Arc::new(Mutex::new(None));
    let custody: Custody = Arc::new(PlatformCustody::new());
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .manage(engine.clone())
        .manage(custody)
        .invoke_handler(tauri::generate_handler![
            run_start,
            run_list,
            run_cancel,
            event_list,
            event_query,
            event_get,
            run_query,
            snapshot_capture,
            snapshot_list,
            branch_list,
            snapshot_validate,
            snapshot_materialize,
            object_inventory,
            fixture_eligibility,
            fixture_create,
            fixture_list,
            fixture_inspect,
            fixture_import,
            fixture_export,
            fixture_delete,
            replay_evaluate,
            analysis_compare,
            analysis_report_list,
            analysis_report_get,
            analysis_report_reevaluate,
            benchmark_create,
            benchmark_import,
            benchmark_list,
            benchmark_get,
            benchmark_export,
            benchmark_compare,
            benchmark_evaluate,
            lineage_list,
            lineage_get,
            retention_set,
            governance_audit,
            governance_delete,
            identity_create,
            identity_list,
            identity_trust,
            identity_migration_list,
            identity_migrate,
            identity_custody_status,
            identity_trust_audit,
            exchange_export,
            exchange_inspect,
            exchange_import
        ])
        .build(tauri::generate_context!())
        .expect("failed to build ChronoSync desktop host")
        .run(move |_, event| {
            if let tauri::RunEvent::Exit = event {
                if let Ok(mut guard) = engine.lock() {
                    guard.take();
                }
            }
        });
}
