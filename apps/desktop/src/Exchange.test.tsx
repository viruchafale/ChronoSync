// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Exchange } from "./Exchange";
import type {
  ExchangeAPI,
  ExchangeVerification,
  IdentityRecord,
} from "../../../contracts/exchange";
import type {
  BenchmarkDataset,
  DatasetRecord,
} from "../../../contracts/governance";

const identity: IdentityRecord = {
  identity: {
    format_version: 1,
    key_id: "a".repeat(64),
    algorithm: "ed25519",
    public_key: "synthetic-public-key",
    label: "Personal signer",
    created_at: "2026-11-01T00:00:00Z",
  },
  trust: "trusted",
  has_private_key: true,
  trust_revision: 1,
  custody_backend: "macos_keychain",
  custody_status: "unchecked",
  warnings: [],
};

const datasetRecord: DatasetRecord = {
  content_id: "b".repeat(64),
  benchmark_id: "c".repeat(32),
  version: 1,
  name: "Portable benchmark",
  cases: 1,
  bytes: 512,
  integrity: "verified",
  lifecycle: "active",
  warnings: [],
};

const dataset: BenchmarkDataset = {
  format_version: 1,
  benchmark_id: datasetRecord.benchmark_id!,
  version: 1,
  name: datasetRecord.name!,
  created_at: "2026-11-01T00:00:00Z",
  cases: [
    {
      case_id: "case-001",
      source_report_id: "d".repeat(32),
      comparison_index: 0,
      baseline_observation_id: "baseline",
      observed_observation_id: "observed",
      comparison_sha256: "e".repeat(64),
      expected_status: "partial_match",
      supported_dimensions: ["event_type", "ordering"],
    },
  ],
  supported_dimensions: ["event_type", "ordering"],
  required_dimensions: ["event_type", "ordering"],
  compatibility_model: "case-id-v1",
  provenance_limitations: ["Synthetic evidence only."],
};

const verification: ExchangeVerification = {
  format_version: 1,
  status: "valid_untrusted",
  package_id: "f".repeat(64),
  signature_valid: true,
  trust: "untrusted",
  identity: { ...identity.identity, label: "External signer" },
  manifest: {
    format_version: 1,
    domain: "chronosync.benchmark-exchange",
    dataset_content_id: datasetRecord.content_id,
    benchmark_id: dataset.benchmark_id,
    benchmark_version: 1,
    compatibility_model: "case-id-v1",
    entries: [],
    included_report_ids: [dataset.cases[0]!.source_report_id],
    omitted_report_ids: [],
    privacy: {
      raw_commands: false,
      raw_outputs: false,
      workspace_paths: false,
      credentials: false,
      environment: false,
      private_keys: false,
      included_evidence: ["benchmark_dataset"],
      omitted_evidence: [],
    },
  },
  importable: true,
  warnings: ["Valid does not mean trusted or safe."],
  detail: "Signature valid; identity is not locally trusted.",
};

function mockAPI(overrides: Partial<ExchangeAPI> = {}): ExchangeAPI {
  return {
    createIdentity: vi.fn(),
    identities: vi.fn().mockResolvedValue([identity]),
    migrations: vi.fn().mockResolvedValue([]),
    migrate: vi.fn(),
    custodyStatus: vi.fn().mockResolvedValue("available"),
    auditTrust: vi.fn().mockResolvedValue({
      format_version: 1,
      verified: true,
      identities: 1,
      events: 1,
      revoked: 0,
      issues: [],
    }),
    setTrust: vi.fn(),
    exportPackage: vi.fn(),
    inspectPackage: vi.fn(),
    importPackage: vi.fn(),
    ...overrides,
  };
}

const evidence = {
  listDatasets: vi.fn().mockResolvedValue([datasetRecord]),
  getDataset: vi.fn().mockResolvedValue(dataset),
};

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("Exchange", () => {
  it("selects governed evidence and sends explicit report inclusions for export", async () => {
    const exportPackage = vi.fn().mockResolvedValue({
      exported: true,
      package_id: "f".repeat(64),
      bytes: 1024,
      verification: {
        ...verification,
        status: "valid_trusted",
        trust: "trusted",
      },
    });
    render(<Exchange api={mockAPI({ exportPackage })} evidence={evidence} />);
    fireEvent.click(
      screen.getByRole("button", { name: "Refresh exchange inventory" }),
    );
    await screen.findByRole("button", { name: "Reject identity" });
    fireEvent.change(screen.getByLabelText("Governed benchmark"), {
      target: { value: datasetRecord.content_id },
    });
    await screen.findByText(`Include ${dataset.cases[0]!.source_report_id}`);
    fireEvent.change(screen.getByLabelText("Eligible local signer"), {
      target: { value: identity.identity.key_id },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Sign and export package" }),
    );
    await waitFor(() =>
      expect(exportPackage).toHaveBeenCalledWith(
        datasetRecord.content_id,
        identity.identity.key_id,
        [dataset.cases[0]!.source_report_id],
      ),
    );
    expect(await screen.findByText("valid_trusted")).not.toBeNull();
  });

  it("displays cryptographic validity separately from local trust and privacy", async () => {
    render(
      <Exchange
        api={mockAPI({
          inspectPackage: vi.fn().mockResolvedValue(verification),
        })}
        evidence={evidence}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Inspect signed package" }),
    );
    expect(await screen.findByText("valid_untrusted")).not.toBeNull();
    expect(screen.getByText(/Signature valid: yes/).textContent).toContain(
      "local trust: untrusted",
    );
    expect(screen.getByText(/Privacy flags/)).not.toBeNull();
  });

  it("exposes explicit trust and permanent revocation actions", async () => {
    const untrusted = {
      ...identity,
      trust: "untrusted" as const,
      has_private_key: false,
    };
    const setTrust = vi
      .fn()
      .mockResolvedValue({ ...untrusted, trust: "revoked" });
    render(
      <Exchange
        api={mockAPI({
          identities: vi.fn().mockResolvedValue([untrusted]),
          setTrust,
        })}
        evidence={evidence}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Refresh exchange inventory" }),
    );
    await screen.findByRole("button", { name: "Trust identity" });
    fireEvent.click(screen.getByRole("button", { name: "Revoke identity" }));
    await waitFor(() =>
      expect(setTrust).toHaveBeenCalledWith(
        untrusted.identity.key_id,
        "revoked",
        "explicit desktop revoked decision",
      ),
    );
  });

  it("requires explicit legacy migration and surfaces trust audit results", async () => {
    const legacy = {
      ...identity,
      custody_backend: "legacy_file" as const,
      custody_status: "available" as const,
      migration_state: "legacy_detected" as const,
    };
    const migrate = vi.fn().mockResolvedValue({
      key_id: legacy.identity.key_id,
      state: "migrated",
      revision: 5,
      updated_at: "2026-11-03T00:00:00Z",
      eligible: false,
      custody_backend: "macos_keychain",
      detail: "verified migration complete",
    });
    const auditTrust = vi.fn().mockResolvedValue({
      format_version: 1,
      verified: true,
      identities: 1,
      events: 2,
      revoked: 0,
      issues: [],
    });
    render(
      <Exchange
        api={mockAPI({
          identities: vi.fn().mockResolvedValue([legacy]),
          migrations: vi.fn().mockResolvedValue([]),
          migrate,
          auditTrust,
        })}
        evidence={evidence}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Refresh exchange inventory" }),
    );
    await screen.findByRole("button", { name: "Migrate to macOS Keychain" });
    fireEvent.click(
      screen.getByRole("button", { name: "Migrate to macOS Keychain" }),
    );
    await waitFor(() =>
      expect(migrate).toHaveBeenCalledWith(legacy.identity.key_id),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Audit trust history" }),
    );
    await screen.findByText(/Trust audit: verified/);
    expect(auditTrust).toHaveBeenCalledOnce();
  });

  it("surfaces a native custody access failure without claiming availability", async () => {
    const custodyStatus = vi
      .fn()
      .mockRejectedValue(new Error("KEY_ACCESS_DENIED"));
    render(<Exchange api={mockAPI({ custodyStatus })} evidence={evidence} />);
    fireEvent.click(
      screen.getByRole("button", { name: "Refresh exchange inventory" }),
    );
    await screen.findByRole("button", { name: "Check Keychain access" });
    fireEvent.click(
      screen.getByRole("button", { name: "Check Keychain access" }),
    );
    expect((await screen.findByRole("alert")).textContent).toContain(
      "KEY_ACCESS_DENIED",
    );
    expect(screen.queryByText(/Keychain access: available/)).toBeNull();
  });
});
