#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]
mod bridge;
use bridge::Bridge;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::sync::{Arc, Mutex};
use tauri::Manager;

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
fn main() {
    let engine: Engine = Arc::new(Mutex::new(None));
    tauri::Builder::default()
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
            object_inventory
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
