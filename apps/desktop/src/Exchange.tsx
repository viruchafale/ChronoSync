import { useState } from "react";
import { SectionNav, Badge, CopyButton } from "./Workbench";
import type {
  ExchangeAPI,
  ExchangeVerification,
  ExchangeViewDependencies,
  IdentityRecord,
  MigrationRecord,
  TrustAudit,
  TrustState,
} from "../../../contracts/exchange";
import type { DatasetRecord } from "../../../contracts/governance";

export function Exchange({
  api,
  evidence,
}: {
  api: ExchangeAPI;
  evidence: ExchangeViewDependencies;
}) {
  const [identities, setIdentities] = useState<IdentityRecord[]>([]);
  const [datasets, setDatasets] = useState<DatasetRecord[]>([]);
  const [migrations, setMigrations] = useState<MigrationRecord[]>([]);
  const [trustAudit, setTrustAudit] = useState<TrustAudit | null>(null);
  const [label, setLabel] = useState("");
  const [supersedes, setSupersedes] = useState("");
  const [identityID, setIdentityID] = useState("");
  const [datasetID, setDatasetID] = useState("");
  const [availableReportIDs, setAvailableReportIDs] = useState<string[]>([]);
  const [reportIDs, setReportIDs] = useState<string[]>([]);
  const [verification, setVerification] = useState<ExchangeVerification | null>(
    null,
  );
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function action(operation: () => Promise<void>) {
    setError("");
    setMessage("");
    try {
      await operation();
    } catch (value) {
      setError(String(value));
    }
  }

  async function refresh() {
    const [nextIdentities, nextDatasets, nextMigrations] = await Promise.all([
      api.identities(),
      evidence.listDatasets(),
      api.migrations(),
    ]);
    setIdentities(nextIdentities);
    setDatasets(nextDatasets);
    setMigrations(nextMigrations);
  }

  async function selectDataset(id: string) {
    setDatasetID(id);
    if (!id) {
      setAvailableReportIDs([]);
      setReportIDs([]);
      return;
    }
    const dataset = await evidence.getDataset(id);
    const ids = [
      ...new Set(dataset.cases.map((item) => item.source_report_id)),
    ].sort();
    setAvailableReportIDs(ids);
    setReportIDs(ids);
  }

  async function changeTrust(keyID: string, state: TrustState) {
    const result = await api.setTrust(
      keyID,
      state,
      `explicit desktop ${state} decision`,
    );
    if ("cancelled" in result) {
      setMessage("Trust change cancelled.");
      return;
    }
    await refresh();
    setMessage(`Identity is now ${result.trust}.`);
  }

  return (
    <section aria-labelledby="exchange-heading">
      <h2 id="exchange-heading">Signed benchmark exchange</h2>
      <SectionNav
        items={[
          ["identity-heading", "Identities & trust"],
          ["export-heading", "Sign & export"],
          ["import-heading", "Inspect & import"],
        ]}
      />
      <p>
        Exchange non-executable benchmark evidence. Signature validity, local
        trust, content safety, benchmark quality, and execution authorization
        are separate. Private signing keys never enter this interface.
      </p>
      {error && <p role="alert">{error}</p>}
      {message && <p role="status">{message}</p>}
      <button
        onClick={() =>
          void action(async () => {
            await refresh();
            setMessage("Exchange identities and benchmarks refreshed.");
          })
        }
      >
        Refresh exchange inventory
      </button>

      <section aria-labelledby="identity-heading">
        <h3 id="identity-heading">Personal signing identities</h3>
        <p>
          New identities use the macOS login Keychain. Ed25519 is not a Secure
          Enclave key: signing material exists briefly in trusted native memory.
          The current unsigned development build cannot enforce per-sign user
          presence; legacy file seeds require explicit migration.
        </p>
        <label>
          Identity label
          <input
            value={label}
            onChange={(event) => setLabel(event.target.value)}
          />
        </label>
        <label>
          Optional superseded key ID
          <input
            value={supersedes}
            onChange={(event) => setSupersedes(event.target.value)}
          />
        </label>
        <button
          disabled={!label}
          onClick={() =>
            void action(async () => {
              const result = await api.createIdentity(label, supersedes);
              if ("cancelled" in result) {
                setMessage("Identity creation cancelled.");
                return;
              }
              setIdentityID(result.identity.key_id);
              setLabel("");
              await refresh();
              setMessage("Personal signing identity created locally.");
            })
          }
        >
          Create signing identity
        </button>
        <ul aria-label="Signing identities">
          {identities.map((record) => (
            <li key={record.identity.key_id}>
              <strong>{record.identity.label}</strong> ·{" "}
              <Badge value={record.trust} /> ·{" "}
              {record.has_private_key ? "local signer" : "public identity"} ·{" "}
              {record.custody_backend} · {record.custody_status} ·{" "}
              <code>{record.identity.key_id}</code>
              <CopyButton value={record.identity.key_id} label="Copy key ID" />
              {record.migration_state && (
                <span> · migration {record.migration_state}</span>
              )}
              {record.identity.supersedes && (
                <span> · supersedes {record.identity.supersedes}</span>
              )}
              {record.trust === "untrusted" && (
                <button
                  onClick={() =>
                    void action(() =>
                      changeTrust(record.identity.key_id, "trusted"),
                    )
                  }
                >
                  Trust identity
                </button>
              )}
              {record.trust === "trusted" && (
                <button
                  onClick={() =>
                    void action(() =>
                      changeTrust(record.identity.key_id, "untrusted"),
                    )
                  }
                >
                  Reject identity
                </button>
              )}
              {record.trust !== "revoked" && (
                <button
                  onClick={() =>
                    void action(() =>
                      changeTrust(record.identity.key_id, "revoked"),
                    )
                  }
                >
                  Revoke identity
                </button>
              )}
              {record.custody_backend === "legacy_file" && (
                <button
                  onClick={() =>
                    void action(async () => {
                      const result = await api.migrate(record.identity.key_id);
                      if ("cancelled" in result) {
                        setMessage(
                          "Key migration cancelled; legacy key retained.",
                        );
                        return;
                      }
                      await refresh();
                      setMessage(`Migration state: ${result.state}.`);
                    })
                  }
                >
                  Migrate to macOS Keychain
                </button>
              )}
              {record.custody_backend === "macos_keychain" && (
                <button
                  onClick={() =>
                    void action(async () => {
                      const status = await api.custodyStatus(
                        record.identity.key_id,
                      );
                      setMessage(`Keychain access status: ${status}.`);
                    })
                  }
                >
                  Check Keychain access
                </button>
              )}
            </li>
          ))}
        </ul>
        <button
          onClick={() =>
            void action(async () => {
              const result = await api.auditTrust();
              setTrustAudit(result);
              setMessage(
                result.verified
                  ? "Trust history verified."
                  : "Trust history is corrupt; automatic recovery is refused.",
              );
            })
          }
        >
          Audit trust history
        </button>
        {trustAudit && (
          <p aria-label="Trust audit result">
            Trust audit: {trustAudit.verified ? "verified" : "failed"} ·{" "}
            {trustAudit.identities} identities · {trustAudit.events} events ·{" "}
            {trustAudit.revoked} revoked
            {trustAudit.issues.length > 0 &&
              ` · ${trustAudit.issues.join("; ")}`}
          </p>
        )}
        {migrations.length > 0 && (
          <ul aria-label="Key migrations">
            {migrations.map((migration) => (
              <li key={migration.key_id}>
                {migration.key_id} · {migration.state} · {migration.detail}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby="export-heading">
        <h3 id="export-heading">Sign and export</h3>
        <label>
          Governed benchmark
          <select
            value={datasetID}
            onChange={(event) =>
              void action(() => selectDataset(event.target.value))
            }
          >
            <option value="">Select benchmark</option>
            {datasets
              .filter((item) => item.integrity === "verified")
              .map((item) => (
                <option key={item.content_id} value={item.content_id}>
                  {item.name} v{item.version}
                </option>
              ))}
          </select>
        </label>
        <label>
          Eligible local signer
          <select
            value={identityID}
            onChange={(event) => setIdentityID(event.target.value)}
          >
            <option value="">Select signer</option>
            {identities
              .filter(
                (item) => item.has_private_key && item.trust !== "revoked",
              )
              .map((item) => (
                <option key={item.identity.key_id} value={item.identity.key_id}>
                  {item.identity.label} · {item.trust}
                </option>
              ))}
          </select>
        </label>
        <fieldset>
          <legend>Required provenance reports</legend>
          {availableReportIDs.length === 0 ? (
            <p>Select a benchmark to inspect its report references.</p>
          ) : (
            availableReportIDs.map((id) => (
              <label key={id}>
                <input
                  type="checkbox"
                  checked={reportIDs.includes(id)}
                  onChange={(event) =>
                    setReportIDs((current) =>
                      event.target.checked
                        ? [...new Set([...current, id])].sort()
                        : current.filter((value) => value !== id),
                    )
                  }
                />
                Include {id}
              </label>
            ))
          )}
        </fieldset>
        <button
          disabled={!datasetID || !identityID}
          onClick={() =>
            void action(async () => {
              const result = await api.exportPackage(
                datasetID,
                identityID,
                reportIDs,
              );
              if ("cancelled" in result) {
                setMessage("Signed export cancelled.");
                return;
              }
              setVerification(result.verification);
              setMessage(
                `Signed package ${result.package_id} exported and reverified.`,
              );
            })
          }
        >
          Sign and export package
        </button>
      </section>

      <section aria-labelledby="import-heading">
        <h3 id="import-heading">Inspect and import</h3>
        <button
          onClick={() =>
            void action(async () => {
              const result = await api.inspectPackage();
              if ("cancelled" in result) {
                setMessage("Package inspection cancelled.");
                return;
              }
              setVerification(result);
              setMessage("Package inspected without importing it.");
            })
          }
        >
          Inspect signed package
        </button>
        <button
          onClick={() =>
            void action(async () => {
              const result = await api.importPackage();
              if ("cancelled" in result && result.cancelled) {
                setMessage("Package import cancelled.");
                if (result.verification) setVerification(result.verification);
                return;
              }
              if ("verification" in result && result.verification) {
                setVerification(result.verification);
                if ("dataset" in result) {
                  await refresh();
                  setMessage(
                    `Imported benchmark and ${result.reports} reports.`,
                  );
                } else {
                  setMessage("Package was inspected but not importable.");
                }
              }
            })
          }
        >
          Import signed package
        </button>
      </section>

      {verification && (
        <section aria-label="Exchange verification">
          <h3>{verification.status}</h3>
          <p>{verification.detail}</p>
          <p>
            Signature valid: {verification.signature_valid ? "yes" : "no"} ·
            local trust: {verification.trust} · importable:{" "}
            {verification.importable ? "yes" : "no"}
          </p>
          {verification.identity && (
            <p>
              Signer: {verification.identity.label} ·{" "}
              {verification.identity.key_id}
            </p>
          )}
          {verification.manifest && (
            <>
              <p>
                Benchmark {verification.manifest.benchmark_id} v
                {verification.manifest.benchmark_version} ·{" "}
                {verification.manifest.included_report_ids.length} included
                reports · {verification.manifest.omitted_report_ids.length}{" "}
                omitted reports
              </p>
              <p>
                Privacy flags: no raw commands, raw outputs, workspace paths,
                credentials, environment variables, or private keys.
              </p>
            </>
          )}
          {verification.warnings.map((warning) => (
            <p key={warning}>{warning}</p>
          ))}
        </section>
      )}
    </section>
  );
}
