import {
  BadRequestSchema,
  ErrorInfoSchema,
  RetryInfoSchema,
} from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import type { Status } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/status_pb";
import { anyUnpack } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import dayjs from "dayjs";
import type { TFunction } from "i18next";
import {
  type ImportInstancesResponse,
  type Instance,
  InstanceImportFailure_CreationOutcome,
  type NeonBranch,
} from "@/types/proto-es/v1/instance_service_pb";

export const NEON_BATCH_SIZE = 20;
export const NEON_DEFAULT_SYNC_SECONDS = 3600;
// The schedules setup offers, in seconds. Zero turns scheduled sync off.
export const NEON_SYNC_INTERVALS: readonly number[] = [3600, 21600, 86400, 0];

const RESOURCE_ID_PATTERN = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;
const RESOURCE_ID_MAX_LENGTH = 63;

// ---- Drafts ----

export type NeonSharedSettings = {
  // Environment resource name; empty means no environment.
  environment: string;
  syncAll: boolean;
  // Zero turns scheduled sync off.
  syncSeconds: number;
};

export const defaultSharedSettings = (): NeonSharedSettings => ({
  environment: "",
  syncAll: true,
  syncSeconds: NEON_DEFAULT_SYNC_SECONDS,
});

export type NeonImportDraft = {
  branch: NeonBranch;
  selected: boolean;
  title: string;
  instanceId: string;
  // An undefined setting inherits the shared value.
  environment?: string;
  syncAll?: boolean;
  syncSeconds?: number;
  // The databases synced when sync-all is off. An empty list is valid.
  databases: string[];
  role: string;
  database: string;
  passwordSource: "neon" | "typed";
  password: string;
};

export type NeonResolvedDraft = NeonImportDraft & {
  environment: string;
  syncAll: boolean;
  syncSeconds: number;
};

export function branchKey(branch: NeonBranch): string {
  return JSON.stringify([branch.projectId, branch.branchId]);
}

export function branchLabel(branch: NeonBranch): string {
  return `${branch.projectTitle || branch.projectId} / ${branch.branchTitle || branch.branchId}`;
}

// A branch needs a database to connect to before it can become an instance.
export function branchAvailable(branch: NeonBranch): boolean {
  return !branch.discoveryError && branch.databases.length > 0;
}

export function commonOwner(
  branch: NeonBranch,
  databases?: readonly string[]
): string {
  const selected = branch.databases.filter(
    (database) => databases === undefined || databases.includes(database.name)
  );
  const owner = selected[0]?.owner;
  return owner &&
    branch.roles.includes(owner) &&
    selected.every((database) => database.owner === owner)
    ? owner
    : "";
}

export function isValidInstanceId(id: string): boolean {
  return RESOURCE_ID_PATTERN.test(id);
}

const slug = (value: string) =>
  value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");

export function neonInstanceId(
  branch: NeonBranch,
  taken: ReadonlySet<string>
): string {
  const parts = [
    slug(branch.projectTitle || branch.projectId),
    slug(branch.branchTitle || branch.branchId),
  ].filter(Boolean);
  const fit = (base: string, suffix: string) =>
    `${base.slice(0, RESOURCE_ID_MAX_LENGTH - suffix.length).replace(/-+$/, "")}${suffix}`;
  const base = ["neon", ...parts].join("-");
  let id = fit(base, "");
  for (let n = 2; taken.has(id); n++) {
    id = fit(base, `-${n}`);
  }
  return id;
}

export function makeDraft(
  branch: NeonBranch,
  taken: ReadonlySet<string> = new Set()
): NeonImportDraft {
  return {
    branch,
    selected: false,
    title: branchLabel(branch),
    instanceId: neonInstanceId(branch, taken),
    databases: branch.databases.map((database) => database.name),
    role: commonOwner(branch),
    database: branch.databases[0]?.name ?? "",
    passwordSource: "neon",
    password: "",
  };
}

// Merges a discovery page into the loaded rows by Neon project and branch ID.
// "carried" holds rows from a traversal that was restarted: a branch returned
// again takes its earlier selection and settings back.
export function mergeDiscovery(
  existing: readonly NeonImportDraft[],
  branches: readonly NeonBranch[],
  carried?: ReadonlyMap<string, NeonImportDraft>
): NeonImportDraft[] {
  const rows = [...existing];
  const indices = new Map(
    rows.map((row, index) => [branchKey(row.branch), index])
  );
  const taken = new Set(rows.map((row) => row.instanceId));
  for (const row of carried?.values() ?? []) taken.add(row.instanceId);
  for (const branch of branches) {
    const key = branchKey(branch);
    const index = indices.get(key);
    const previous = index === undefined ? carried?.get(key) : rows[index];
    if (!previous) {
      const draft = makeDraft(branch, taken);
      taken.add(draft.instanceId);
      indices.set(key, rows.length);
      rows.push(draft);
      continue;
    }
    // A choice the branch no longer offers goes back to needing input.
    const names = branch.databases.map((database) => database.name);
    const roleKept = branch.roles.includes(previous.role);
    const merged = {
      ...previous,
      branch,
      selected: previous.selected && branchAvailable(branch),
      role: roleKept ? previous.role : "",
      password: roleKept ? previous.password : "",
      database: names.includes(previous.database)
        ? previous.database
        : (names[0] ?? ""),
      databases: previous.databases.filter((name) => names.includes(name)),
    };
    if (index === undefined) {
      indices.set(key, rows.length);
      rows.push(merged);
    } else {
      rows[index] = merged;
    }
  }
  return rows;
}

export function effectiveSettings(
  draft: NeonImportDraft,
  shared: NeonSharedSettings
): NeonSharedSettings {
  return {
    environment: draft.environment ?? shared.environment,
    syncAll: draft.syncAll ?? shared.syncAll,
    syncSeconds: draft.syncSeconds ?? shared.syncSeconds,
  };
}

export function resolveDraft(
  draft: NeonImportDraft,
  shared: NeonSharedSettings
): NeonResolvedDraft {
  return {
    ...draft,
    ...effectiveSettings(draft, shared),
    databases: [...draft.databases],
  };
}

export function hasOverride(draft: NeonImportDraft): boolean {
  return (
    draft.environment !== undefined ||
    draft.syncAll !== undefined ||
    draft.syncSeconds !== undefined
  );
}

export type NeonSharedKey = keyof NeonSharedSettings;

export function countOverrides(
  drafts: readonly NeonImportDraft[]
): Record<NeonSharedKey, number> {
  const count = (key: NeonSharedKey) =>
    drafts.filter((draft) => draft[key] !== undefined).length;
  return {
    environment: count("environment"),
    syncAll: count("syncAll"),
    syncSeconds: count("syncSeconds"),
  };
}

// A typed password belongs to the role it was typed for.
export function withRole<T extends NeonImportDraft>(draft: T, role: string): T {
  if (draft.role === role) return draft;
  return { ...draft, role, password: "" };
}

export function withDatabaseToggled<T extends NeonImportDraft>(
  draft: T,
  name: string,
  included: boolean
): T {
  const databases = included
    ? [...new Set([...draft.databases, name])]
    : draft.databases.filter((database) => database !== name);
  return {
    ...draft,
    databases,
    role: draft.role || commonOwner(draft.branch, databases),
  };
}

export function duplicateInstanceIds(
  drafts: readonly NeonImportDraft[]
): Set<string> {
  const seen = new Set<string>();
  const duplicates = new Set<string>();
  for (const draft of drafts) {
    if (seen.has(draft.instanceId)) duplicates.add(draft.instanceId);
    seen.add(draft.instanceId);
  }
  return duplicates;
}

export type NeonDraftNeed = "role" | "password" | "title" | "instance-id";

export function draftNeed(
  draft: NeonImportDraft,
  duplicateIds: ReadonlySet<string> = new Set()
): NeonDraftNeed | undefined {
  if (!draft.role) return "role";
  if (draft.passwordSource === "typed" && !draft.password) return "password";
  if (!draft.title.trim()) return "title";
  if (
    !isValidInstanceId(draft.instanceId) ||
    duplicateIds.has(draft.instanceId)
  ) {
    return "instance-id";
  }
  return undefined;
}

// ---- License ----

export type NeonLicense = {
  instanceCountLimit: number;
  totalInstanceCount: number;
  instanceLicenseCount: number;
  activatedInstanceCount: number;
};

const isUnlimited = (count: number) => count >= Number.MAX_SAFE_INTEGER;

const isUnifiedLicense = (license: NeonLicense) =>
  license.instanceCountLimit <= license.instanceLicenseCount;

// Mirrors the server: activation is requested whenever the license provides it.
export function importsActivated(license: NeonLicense): boolean {
  return isUnifiedLicense(license) || license.instanceLicenseCount > 0;
}

// The number of instances the workspace can still add; undefined means no limit.
export function remainingInstanceCapacity(
  license: NeonLicense
): number | undefined {
  let remaining: number | undefined;
  if (!isUnlimited(license.instanceCountLimit)) {
    remaining = Math.max(
      0,
      license.instanceCountLimit - license.totalInstanceCount
    );
  }
  if (
    importsActivated(license) &&
    !isUnifiedLicense(license) &&
    !isUnlimited(license.instanceLicenseCount)
  ) {
    const activations = Math.max(
      0,
      license.instanceLicenseCount - license.activatedInstanceCount
    );
    remaining =
      remaining === undefined ? activations : Math.min(remaining, activations);
  }
  return remaining;
}

// ---- Structured errors ----

function statusErrorInfo(status?: Status) {
  for (const detail of status?.details ?? []) {
    try {
      const info = anyUnpack(detail, ErrorInfoSchema);
      if (info) return info;
    } catch {
      // An unfamiliar or malformed detail cannot establish a retry-safe outcome.
    }
  }
  return undefined;
}

export function statusReason(status?: Status): string {
  return statusErrorInfo(status)?.reason ?? "";
}

// The connection-failure category the server attaches to
// INSTANCE_CONNECTION_FAILED, such as "auth_failed".
export function statusCategory(status?: Status): string {
  return statusErrorInfo(status)?.metadata.category ?? "";
}

export function errorReason(error: unknown): string {
  if (!(error instanceof ConnectError)) return "";
  return error.findDetails(ErrorInfoSchema)[0]?.reason ?? "";
}

// The request fields a malformed-request error names.
export function errorViolatedFields(error: unknown): string[] {
  if (!(error instanceof ConnectError)) return [];
  return error
    .findDetails(BadRequestSchema)
    .flatMap((detail) => detail.fieldViolations.map((item) => item.field));
}

export function retryDelaySeconds(error: unknown): number {
  if (!(error instanceof ConnectError)) return 0;
  const delay = error.findDetails(RetryInfoSchema)[0]?.retryDelay;
  if (!delay) return 0;
  return Math.max(0, Math.ceil(Number(delay.seconds) + delay.nanos / 1e9));
}

export type NeonRequestFailureKind =
  | "key-invalid"
  | "project-id-required"
  | "project-id-invalid"
  | "project-id-unreadable"
  | "rate-limited"
  | "unavailable"
  | "permission-denied"
  | "token-invalid"
  | "other";

export type NeonRequestFailure = {
  kind: NeonRequestFailureKind;
  retryAfterSeconds: number;
  // Shown only for a failure this UI has no wording for.
  message: string;
};

export function classifyRequestFailure(
  error: unknown,
  hasProjectFilter: boolean
): NeonRequestFailure {
  const reason = errorReason(error);
  const code = error instanceof ConnectError ? error.code : Code.Unknown;
  const kind = ((): NeonRequestFailureKind => {
    switch (reason) {
      case "VENDOR_CREDENTIAL_INVALID":
        return "key-invalid";
      case "VENDOR_RATE_LIMITED":
        return "rate-limited";
      case "VENDOR_UNAVAILABLE":
        return "unavailable";
      case "INVALID_PAGE_TOKEN":
        return "token-invalid";
      // Listing every Neon project is what a project-scoped key cannot do.
      case "VENDOR_ACCESS_DENIED":
      case "VENDOR_RESOURCE_NOT_FOUND":
        return hasProjectFilter
          ? "project-id-unreadable"
          : "project-id-required";
    }
    if (reason) return "other";
    if (errorViolatedFields(error).includes("project_id")) {
      return "project-id-invalid";
    }
    switch (code) {
      case Code.PermissionDenied:
        return "permission-denied";
      case Code.ResourceExhausted:
        return "rate-limited";
      case Code.Unavailable:
      case Code.DeadlineExceeded:
        return "unavailable";
      default:
        return "other";
    }
  })();
  return {
    kind,
    retryAfterSeconds: retryDelaySeconds(error),
    message: error instanceof ConnectError ? error.rawMessage : "",
  };
}

// Wording for a structured reason. The server message is the fallback for a
// reason this UI does not know. "category" refines a connection failure.
export function neonReasonText(
  t: TFunction,
  reason: string,
  fallback = "",
  category = ""
): string {
  switch (reason) {
    case "VENDOR_ENDPOINT_UNAVAILABLE":
      return t("instance.neon.reason.endpoint-unavailable");
    case "VENDOR_ACCESS_DENIED":
      return t("instance.neon.reason.access-denied");
    case "VENDOR_RESOURCE_NOT_FOUND":
      return t("instance.neon.reason.resource-not-found");
    case "VENDOR_CREDENTIAL_INVALID":
      return t("instance.neon.reason.key-invalid");
    case "VENDOR_RATE_LIMITED":
      return t("instance.neon.reason.rate-limited");
    case "VENDOR_UNAVAILABLE":
      return t("instance.neon.reason.unavailable");
    case "VENDOR_RESPONSE_INVALID":
      return t("instance.neon.reason.response-invalid");
    case "VENDOR_PASSWORD_REQUIRED":
      return t("instance.neon.reason.password-required");
    case "VENDOR_USER_NOT_FOUND":
      return t("instance.neon.reason.role-not-found");
    case "VENDOR_DATABASE_NOT_FOUND":
      return t("instance.neon.reason.database-not-found");
    case "INSTANCE_ID_EXISTS":
      return t("instance.neon.reason.instance-id-exists");
    case "INSTANCE_LIMIT_EXCEEDED":
      return t("instance.neon.reason.instance-limit");
    case "INSTANCE_CONNECTION_FAILED":
      switch (category) {
        case "auth_failed":
          return t("instance.neon.reason.connection-auth-failed");
        case "timeout":
        case "network_unreachable":
          return t("instance.neon.reason.connection-unreachable");
        default:
          return t("instance.neon.reason.connection-failed");
      }
    case "INSTANCE_SETTINGS_INVALID":
      return [t("instance.neon.reason.settings-invalid"), fallback]
        .filter(Boolean)
        .join(" ");
    case "DESTINATION_NOT_FOUND":
      return t("instance.neon.reason.destination-not-found");
    case "REQUEST_CANCELED":
    case "REQUEST_DEADLINE_EXCEEDED":
      return t("instance.neon.reason.not-processed");
    default:
      return fallback || t("instance.neon.reason.other");
  }
}

// Why a whole import request was refused. Each is decided from the status
// code alone, because the server returns these codes for ImportInstances only
// from its interceptors and admission checks, before any creation starts.
export type NeonRejection =
  | "invalid"
  | "unauthenticated"
  | "permission-denied"
  | "destination-not-found";

export function requestRejection(error: unknown): NeonRejection | undefined {
  if (!(error instanceof ConnectError)) return undefined;
  switch (error.code) {
    case Code.InvalidArgument:
      return "invalid";
    case Code.Unauthenticated:
      return "unauthenticated";
    case Code.PermissionDenied:
      return "permission-denied";
    case Code.NotFound:
      return "destination-not-found";
    default:
      return undefined;
  }
}

export function rejectionText(t: TFunction, rejection: NeonRejection): string {
  switch (rejection) {
    case "invalid":
      return t("instance.neon.reason.batch-rejected");
    case "unauthenticated":
      return t("instance.neon.reason.session-expired");
    case "permission-denied":
      return t("instance.neon.reason.permission-denied");
    case "destination-not-found":
      return t("instance.neon.reason.destination-not-found");
  }
}

// ---- Import run ----

export type NeonEntryState =
  | "unsent"
  | "awaiting"
  | "created"
  | "failed"
  | "unknown";

// lost: the request ended without a usable response. stopped: the user stopped
// the import while the request was active. reported: the server answered that
// it cannot tell.
export type NeonUnknownCause = "lost" | "stopped" | "reported";

export type NeonImportEntry = {
  key: string;
  // Frozen at submission; only corrections made on the Import step change it.
  draft: NeonResolvedDraft;
  batch: number;
  state: NeonEntryState;
  retrying: boolean;
  reason: string;
  message: string;
  // The connection-failure category of an INSTANCE_CONNECTION_FAILED reason.
  category?: string;
  // Set when the whole request was refused before any creation.
  rejection?: NeonRejection;
  // What the last attempt sent, to tell whether a failure's cause was corrected.
  attempted?: { role: string; database: string; instanceId: string };
  unknownCause?: NeonUnknownCause;
  // Set when a check did not find the instance, which unlocks a retry.
  checkedAt?: Date;
  checkFailed: boolean;
  instance?: Instance;
  resolvedBy?: "check" | "retry";
};

export type NeonImportBatch = {
  size: number;
  sentAt?: Date;
  cause?: "lost" | "stopped";
};

export type NeonImportRun = {
  entries: NeonImportEntry[];
  batches: NeonImportBatch[];
};

export function createImportRun(
  drafts: readonly NeonImportDraft[],
  shared: NeonSharedSettings
): NeonImportRun {
  const entries = drafts.map(
    (draft, index): NeonImportEntry => ({
      key: branchKey(draft.branch),
      draft: resolveDraft(draft, shared),
      batch: Math.floor(index / NEON_BATCH_SIZE),
      state: "unsent",
      retrying: false,
      reason: "",
      message: "",
      checkFailed: false,
    })
  );
  const batches = Array.from(
    { length: Math.ceil(entries.length / NEON_BATCH_SIZE) },
    (_, batch): NeonImportBatch => ({
      size: entries.filter((entry) => entry.batch === batch).length,
    })
  );
  return { entries, batches };
}

// The batches not sent yet, as entry indices, in their original grouping.
export function unsentGroups(run: NeonImportRun): number[][] {
  const groups = new Map<number, number[]>();
  run.entries.forEach((entry, index) => {
    if (entry.state !== "unsent") return;
    groups.set(entry.batch, [...(groups.get(entry.batch) ?? []), index]);
  });
  return [...groups.values()];
}

export function chunkIndices(indices: readonly number[]): number[][] {
  const chunks: number[][] = [];
  for (let offset = 0; offset < indices.length; offset += NEON_BATCH_SIZE) {
    chunks.push(indices.slice(offset, offset + NEON_BATCH_SIZE));
  }
  return chunks;
}

// Whether setup sets the schedule: "off" where the license activates no
// instance, so the schedule would never run; "default" where the plan cannot
// change the interval, so every instance takes the default one.
export type NeonScheduleMode = "editable" | "default" | "off";

export function scheduleMode(
  activated: boolean,
  intervalEditable: boolean
): NeonScheduleMode {
  if (!activated) return "off";
  return intervalEditable ? "editable" : "default";
}

export function toImportEntry(
  draft: NeonResolvedDraft,
  mode: NeonScheduleMode
) {
  const syncSeconds = {
    editable: draft.syncSeconds,
    default: NEON_DEFAULT_SYNC_SECONDS,
    off: 0,
  }[mode];
  return {
    instanceId: draft.instanceId,
    instance: {
      title: draft.title.trim(),
      environment: draft.environment || undefined,
      syncInterval: { seconds: BigInt(syncSeconds) },
      syncDatabases: draft.syncAll ? undefined : { databases: draft.databases },
    },
    username: draft.role,
    connectionDatabase: draft.database,
    password: draft.passwordSource === "typed" ? draft.password : undefined,
    source: {
      case: "neonBranch" as const,
      value: {
        projectId: draft.branch.projectId,
        branchId: draft.branch.branchId,
      },
    },
  };
}

export function toImportRequest(
  parent: string | undefined,
  apiKey: string,
  drafts: readonly NeonResolvedDraft[],
  mode: NeonScheduleMode
) {
  return {
    parent,
    credential: { case: "neonCredential" as const, value: { apiKey } },
    imports: drafts.map((draft) => toImportEntry(draft, mode)),
  };
}

export type SendNeonBatch = (
  entries: readonly NeonImportEntry[],
  signal?: AbortSignal
) => Promise<ImportInstancesResponse>;

// Sends each group as one request, in order, and maps results back by request
// order. It stops after a group that leaves an outcome unknown or was rejected
// whole, and never resends an entry on its own.
export async function runImportBatches(
  run: NeonImportRun,
  groups: readonly (readonly number[])[],
  sendBatch: SendNeonBatch,
  onUpdate: (run: NeonImportRun) => void,
  options: { signal?: AbortSignal; retry?: boolean; now?: () => Date } = {}
): Promise<{ run: NeonImportRun; paused: boolean }> {
  const { signal, retry = false, now = () => new Date() } = options;
  const entries = [...run.entries];
  const batches = [...run.batches];
  const snapshot = (): NeonImportRun => ({
    entries: [...entries],
    batches: [...batches],
  });
  const patch = (index: number, change: Partial<NeonImportEntry>) => {
    entries[index] = { ...entries[index], ...change };
  };
  const markBatch = (
    indices: readonly number[],
    cause?: "lost" | "stopped"
  ) => {
    if (retry) return;
    const batch = entries[indices[0]].batch;
    batches[batch] = { ...batches[batch], cause };
  };

  for (const indices of groups) {
    if (indices.length === 0) continue;
    if (signal?.aborted) return { run: snapshot(), paused: true };
    for (const index of indices) {
      const { draft } = entries[index];
      patch(index, {
        state: "awaiting",
        retrying: retry,
        reason: "",
        message: "",
        category: undefined,
        rejection: undefined,
        unknownCause: undefined,
        checkedAt: undefined,
        checkFailed: false,
        attempted: {
          role: draft.role,
          database: draft.database,
          instanceId: draft.instanceId,
        },
      });
    }
    if (!retry) {
      const batch = entries[indices[0]].batch;
      batches[batch] = { ...batches[batch], sentAt: now(), cause: undefined };
    }
    onUpdate(snapshot());

    let paused = false;
    const unknown = (index: number, unknownCause: NeonUnknownCause) => {
      patch(index, { state: "unknown", unknownCause });
      paused = true;
    };
    try {
      const response = await sendBatch(
        indices.map((index) => entries[index]),
        signal
      );
      // A mismatched response cannot be mapped to the submitted entries.
      if (response.results.length !== indices.length) {
        for (const index of indices) unknown(index, "lost");
        markBatch(indices, "lost");
      } else {
        for (const [position, index] of indices.entries()) {
          const outcome = response.results[position].outcome;
          if (outcome.case === "imported" && outcome.value.name) {
            patch(index, {
              state: "created",
              instance: outcome.value,
              resolvedBy: retry ? "retry" : undefined,
              draft: { ...entries[index].draft, password: "" },
            });
          } else if (
            outcome.case === "failure" &&
            outcome.value.creationOutcome ===
              InstanceImportFailure_CreationOutcome.NOT_CREATED
          ) {
            patch(index, {
              state: "failed",
              reason: statusReason(outcome.value.status),
              message: outcome.value.status?.message ?? "",
              category: statusCategory(outcome.value.status),
            });
          } else {
            unknown(index, "reported");
          }
        }
      }
    } catch (error) {
      const rejection = requestRejection(error);
      if (rejection) {
        for (const index of indices) {
          patch(index, {
            state: "failed",
            rejection,
            reason: errorReason(error),
            message: error instanceof ConnectError ? error.rawMessage : "",
          });
        }
        paused = true;
      } else {
        const cause = signal?.aborted ? "stopped" : "lost";
        for (const index of indices) unknown(index, cause);
        markBatch(indices, cause);
      }
    }
    onUpdate(snapshot());
    if (paused || signal?.aborted) return { run: snapshot(), paused: true };
  }
  return { run: snapshot(), paused: false };
}

export type NeonFailureKind =
  | "password"
  | "role"
  | "database"
  | "instance-id"
  | "limit"
  | "other";

export function failureKind(reason: string): NeonFailureKind {
  switch (reason) {
    case "VENDOR_PASSWORD_REQUIRED":
      return "password";
    case "VENDOR_USER_NOT_FOUND":
      return "role";
    case "VENDOR_DATABASE_NOT_FOUND":
      return "database";
    case "INSTANCE_ID_EXISTS":
      return "instance-id";
    case "INSTANCE_LIMIT_EXCEEDED":
      return "limit";
    default:
      return "other";
  }
}

// Whether a confirmed failure can be sent again: its cause is corrected, or it
// needs no correction on the entry.
export function entryRetryable(
  entry: NeonImportEntry,
  duplicateIds: ReadonlySet<string> = new Set()
): boolean {
  if (entry.state !== "failed" || draftNeed(entry.draft, duplicateIds)) {
    return false;
  }
  const { draft, attempted } = entry;
  switch (failureKind(entry.reason)) {
    case "password":
      return draft.passwordSource === "typed";
    case "role":
      return draft.role !== attempted?.role;
    case "database":
      return draft.database !== attempted?.database;
    case "instance-id":
      return draft.instanceId !== attempted?.instanceId;
    default:
      return true;
  }
}

export type NeonInstanceCheck =
  | { key: string; result: "found"; instance: Instance }
  | { key: string; result: "missing" }
  | { key: string; result: "error" };

// Applies an instance-list check to the unknown entries: a found instance
// resolves the entry as created, and only a confirmed absence records a check
// time.
export function applyInstanceChecks(
  run: NeonImportRun,
  checks: readonly NeonInstanceCheck[],
  at: Date
): NeonImportRun {
  const byKey = new Map(checks.map((check) => [check.key, check]));
  const entries = run.entries.map((entry): NeonImportEntry => {
    const check = byKey.get(entry.key);
    if (entry.state !== "unknown" || !check) return entry;
    switch (check.result) {
      case "found":
        return {
          ...entry,
          state: "created",
          instance: check.instance,
          resolvedBy: "check",
          checkFailed: false,
          draft: { ...entry.draft, password: "" },
        };
      case "missing":
        return { ...entry, checkedAt: at, checkFailed: false };
      case "error":
        return { ...entry, checkFailed: true };
    }
  });
  return { ...run, entries };
}

export function updateEntryDraft(
  run: NeonImportRun,
  key: string,
  update: (draft: NeonResolvedDraft) => NeonResolvedDraft
): NeonImportRun {
  return {
    ...run,
    entries: run.entries.map((entry) =>
      entry.key === key && entry.state === "failed"
        ? { ...entry, draft: update(entry.draft) }
        : entry
    ),
  };
}

export type NeonRunSummary = Record<NeonEntryState, number> & {
  total: number;
  processed: number;
};

export function summarizeRun(run: NeonImportRun): NeonRunSummary {
  const count = (state: NeonEntryState) =>
    run.entries.filter((entry) => entry.state === state).length;
  const summary = {
    unsent: count("unsent"),
    awaiting: count("awaiting"),
    created: count("created"),
    failed: count("failed"),
    unknown: count("unknown"),
    total: run.entries.length,
  };
  return {
    ...summary,
    processed: summary.total - summary.unsent - summary.awaiting,
  };
}

export function instanceResourceName(
  parent: string | undefined,
  instanceId: string
): string {
  return parent
    ? `${parent}/instances/${instanceId}`
    : `instances/${instanceId}`;
}

// ---- Manual entry ----

export type NeonConnectionString = {
  host: string;
  port: string;
  role: string;
  password: string;
  database: string;
};

const decode = (value: string) => {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
};

// Reads the connection string the Neon console shows, with or without the
// "psql" prefix and quotes it copies along.
export function parseNeonConnectionString(
  value: string
): NeonConnectionString | undefined {
  const match = value
    .trim()
    .replace(/^psql\s+/, "")
    .replace(/^['"]|['"]$/g, "")
    .match(
      /^postgres(?:ql)?:\/\/([^:@/]+)(?::([^@/]*))?@([^/:?#]+)(?::(\d+))?\/([^?#]+)/
    );
  if (!match) return undefined;
  return {
    role: decode(match[1]),
    password: decode(match[2] ?? ""),
    host: match[3],
    port: match[4] ?? "5432",
    database: decode(match[5]),
  };
}

// Neon's pooled endpoints run PgBouncer, which schema sync and migrations
// cannot use.
export function isNeonPooledHost(host: string): boolean {
  return host.includes("-pooler");
}

// ---- Presentation helpers ----

export const formatClock = (date: Date) => dayjs(date).format("HH:mm:ss");

export function formatCountdown(seconds: number): string {
  const clamped = Math.max(0, Math.ceil(seconds));
  return `${Math.floor(clamped / 60)}:${String(clamped % 60).padStart(2, "0")}`;
}

// The label of one schedule choice, such as "Every 6 hours" or "Off".
export function syncIntervalLabel(t: TFunction, seconds: number): string {
  if (seconds <= 0) return t("instance.neon.interval.off");
  if (seconds === 3600) return t("instance.neon.interval.hourly");
  return seconds % 3600 === 0
    ? t("instance.neon.interval.every-n-hours", { hours: seconds / 3600 })
    : t("instance.neon.sync.scheduled", {
        interval: formatSyncInterval(t, seconds),
      });
}

function formatSyncInterval(t: TFunction, seconds: number): string {
  const minutes = Math.round(seconds / 60);
  if (minutes % (24 * 60) === 0) {
    return t("instance.neon.interval.days", { count: minutes / (24 * 60) });
  }
  if (minutes % 60 === 0) {
    return t("instance.neon.interval.hours", { count: minutes / 60 });
  }
  return t("instance.neon.interval.minutes", { count: minutes });
}

export function syncScheduleText(t: TFunction, seconds: number): string {
  return seconds > 0
    ? t("instance.neon.sync.scheduled", {
        interval: formatSyncInterval(t, seconds),
      })
    : t("instance.neon.sync.manual");
}

export function syncScopeText(t: TFunction, draft: NeonResolvedDraft): string {
  if (draft.syncAll) return t("instance.neon.scope.all");
  return draft.databases.length > 0
    ? t("instance.neon.scope.some", {
        count: draft.databases.length,
        total: draft.branch.databases.length,
      })
    : t("instance.neon.scope.none");
}
