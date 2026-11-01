#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]
mod bridge;
use bridge::Bridge;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::fs::{self, OpenOptions};
use std::io::{Read, Write};
use std::sync::{Arc, Mutex};
use tauri::Manager;
use tauri_plugin_dialog::{DialogExt, MessageDialogButtons};

const MAX_FIXTURE_BYTES: u64 = 2 * 1024 * 1024;

type Engine = Arc<Mutex<Option<Bridge>>>;
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
fn main() {
    let engine: Engine = Arc::new(Mutex::new(None));
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .manage(engine.clone())
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
            governance_delete
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
