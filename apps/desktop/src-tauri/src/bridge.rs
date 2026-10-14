use serde_json::{json, Value};
use std::io::{BufRead, BufReader, Read, Write};
use std::path::Path;
use std::process::{Child, ChildStdin, Command, Stdio};
use std::sync::mpsc::{self, Receiver};
use std::time::{Duration, Instant};

const LIMIT: usize = 1024 * 1024;
pub struct Bridge {
    child: Child,
    input: Option<ChildStdin>,
    replies: Receiver<Result<Value, String>>,
    next: u64,
    healthy: bool,
}
impl Bridge {
    pub fn start(executable: &Path, data: &Path) -> Result<Self, String> {
        let mut child = Command::new(executable)
            .arg("serve")
            .arg("--data-dir")
            .arg(data)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .map_err(|_| "Could not start bundled engine".to_string())?;
        let input = child.stdin.take().ok_or("Missing engine stdin")?;
        let stdout = child.stdout.take().ok_or("Missing engine stdout")?;
        let (tx, rx) = mpsc::sync_channel(2);
        std::thread::spawn(move || {
            let mut reader = BufReader::new(stdout);
            loop {
                let mut frame = Vec::new();
                let result = (&mut reader)
                    .take((LIMIT + 1) as u64)
                    .read_until(b'\n', &mut frame);
                let response = match result {
                    Ok(n) if n > 0 && n <= LIMIT && frame.last() == Some(&b'\n') => {
                        serde_json::from_slice(&frame)
                            .map_err(|_| "Invalid engine response".to_string())
                    }
                    _ => Err("Engine disconnected or exceeded frame limit".to_string()),
                };
                let failed = response.is_err();
                if tx.send(response).is_err() || failed {
                    break;
                }
            }
        });
        let mut bridge = Self {
            child,
            input: Some(input),
            replies: rx,
            next: 0,
            healthy: true,
        };
        let hello = bridge.request("hello", json!({}))?;
        if hello["protocol_version"] != 1
            || !hello["capabilities"].as_array().is_some_and(|c| {
                c.contains(&json!("stdio-recorder-v1"))
                    && c.contains(&json!("timeline-v1"))
                    && c.contains(&json!("snapshots-v1"))
            })
        {
            return Err("Incompatible recorder engine".into());
        }
        Ok(bridge)
    }
    pub fn request(&mut self, method: &str, params: Value) -> Result<Value, String> {
        if !self.healthy {
            return Err("Engine disconnected; restart ChronoSync".into());
        }
        self.next += 1;
        let id = self.next.to_string();
        let mut bytes = serde_json::to_vec(
            &json!({"protocol_version":1,"id":id,"method":method,"params":params}),
        )
        .map_err(|e| e.to_string())?;
        if bytes.len() > LIMIT {
            return Err("Request exceeds frame limit".into());
        }
        bytes.push(b'\n');
        let result = self.exchange(
            &bytes,
            &id,
            if method == "hello" {
                120
            } else if method.starts_with("snapshot.")
                || method.starts_with("branch.")
                || method == "object.inventory"
            {
                30
            } else {
                15
            },
        );
        if result.as_ref().is_err_and(|e| e.starts_with("Transport:")) {
            self.healthy = false;
            self.input.take();
        }
        result
    }
    fn exchange(&mut self, bytes: &[u8], id: &str, timeout_seconds: u64) -> Result<Value, String> {
        self.input
            .as_mut()
            .ok_or("Transport: engine input closed")?
            .write_all(bytes)
            .map_err(|_| "Transport: engine write failed")?;
        let response = self
            .replies
            .recv_timeout(Duration::from_secs(timeout_seconds))
            .map_err(|_| "Transport: engine response timed out")?
            .map_err(|e| format!("Transport: {e}"))?;
        if response["protocol_version"] != 1 || response["id"] != id {
            return Err("Transport: mismatched engine response".into());
        }
        if let Some(error) = response.get("error") {
            return Err(error["message"]
                .as_str()
                .unwrap_or("Engine request failed")
                .to_string());
        }
        response
            .get("result")
            .cloned()
            .ok_or("Transport: missing engine result".into())
    }
}
impl Drop for Bridge {
    fn drop(&mut self) {
        self.input.take(); // EOF asks the engine to cancel and reap its owned group.
        let until = Instant::now() + Duration::from_secs(8);
        while Instant::now() < until {
            if matches!(self.child.try_wait(), Ok(Some(_))) {
                return;
            }
            std::thread::sleep(Duration::from_millis(50));
        }
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    use std::fs;
    use std::os::unix::fs::PermissionsExt;
    use std::time::{SystemTime, UNIX_EPOCH};
    #[test]
    fn real_engine_negotiation_and_cancel() {
        let root = std::env::temp_dir().join(format!(
            "chronosync-bridge-{}",
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        fs::create_dir(&root).unwrap();
        fs::set_permissions(&root, fs::Permissions::from_mode(0o700)).unwrap();
        let workspace = root.join("workspace");
        fs::create_dir(&workspace).unwrap();
        let fake = root.join("synthetic-codex");
        fs::write(&fake, "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codex-cli 0.163.0'; exit 0; fi\nprintf '%s\\n' '{\"type\":\"turn.started\"}'\nsleep 30\n").unwrap();
        fs::set_permissions(&fake, fs::Permissions::from_mode(0o700)).unwrap();
        let executable = Path::new(env!("CARGO_MANIFEST_DIR")).join("binaries/chronosync-engine");
        let data = root.join("store");
        {
            let mut bridge = Bridge::start(&executable, &data).unwrap();
            assert_eq!(bridge.request("run.list", json!({})).unwrap(), json!([]));
            assert!(bridge
                .request("event.list", json!({"id":"../outside","after":"0"}))
                .is_err());
            let run = bridge.request("run.start", json!({"workspace":workspace,"prompt":"synthetic","codex_path":fake,"sandbox":"read-only","timeout_seconds":10,"consent":true,"excludes":[]})).unwrap();
            let id = run["id"].clone();
            // Wait until the subprocess is actually running before cancellation.
            let until = Instant::now() + Duration::from_secs(5);
            loop {
                let page = bridge
                    .request("event.list", json!({"id":id,"after":"0"}))
                    .unwrap();
                if page["run"]["state"] == "running" {
                    break;
                }
                assert!(Instant::now() < until, "run never reached running");
                std::thread::sleep(Duration::from_millis(20));
            }
            bridge.request("run.cancel", json!({"id":id})).unwrap();
            loop {
                let page = bridge
                    .request("event.list", json!({"id":id,"after":"0"}))
                    .unwrap();
                if page["run"]["state"] == "cancelled" {
                    break;
                }
                assert!(Instant::now() < until, "cancel never confirmed");
                std::thread::sleep(Duration::from_millis(20));
            }
        }
        let mut reopened = Bridge::start(&executable, &data).unwrap();
        assert_eq!(
            reopened.request("run.list", json!({})).unwrap()[0]["state"],
            "cancelled"
        );
        let runs = reopened
            .request("run.query", json!({"cursor":"","limit":1}))
            .unwrap();
        let id = runs["runs"][0]["id"].as_str().unwrap();
        let page = reopened
            .request("event.query", json!({"id":id,"limit":2}))
            .unwrap();
        assert_eq!(page["events"].as_array().unwrap().len(), 2);
        let detail = reopened
            .request(
                "event.get",
                json!({"id":id,"event_id":page["events"][0]["event_id"]}),
            )
            .unwrap();
        assert_eq!(detail["sequence"], "1");
        let snap = reopened
            .request(
                "snapshot.capture",
                json!({"run_id":id,"excludes":[],"consent":true}),
            )
            .unwrap();
        assert_eq!(snap["state"], "committed");
        let branch = reopened
            .request(
                "branch.create",
                json!({"snapshot_id":snap["id"],"consent":true}),
            )
            .unwrap();
        assert_eq!(branch["state"], "ready");
        let verified = reopened
            .request("snapshot.validate", json!({"id":snap["id"]}))
            .unwrap();
        assert_eq!(verified["state"], "committed");
        drop(reopened);
        fs::remove_dir_all(root).unwrap();
    }
}
