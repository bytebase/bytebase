import {
  BadRequestSchema,
  ErrorInfoSchema,
  RetryInfoSchema,
} from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { create } from "@bufbuild/protobuf";
import { anyPack } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it, vi } from "vitest";
import {
  ImportInstancesResponseSchema,
  InstanceImportFailure_CreationOutcome,
  InstanceImportResultSchema,
  InstanceSchema,
  NeonBranchSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  applyInstanceChecks,
  branchKey,
  chunkIndices,
  classifyRequestFailure,
  commonOwner,
  countOverrides,
  createImportRun,
  defaultSharedSettings,
  draftNeed,
  duplicateInstanceIds,
  effectiveSettings,
  entryRetryable,
  importsActivated,
  isNeonPooledHost,
  makeDraft,
  mergeDiscovery,
  type NeonImportEntry,
  type NeonImportRun,
  neonInstanceId,
  neonReasonText,
  parseNeonConnectionString,
  remainingInstanceCapacity,
  requestRejection,
  runImportBatches,
  scheduleMode,
  summarizeRun,
  syncIntervalLabel,
  toImportRequest,
  unsentGroups,
  updateEntryDraft,
  withDatabaseToggled,
  withRole,
} from "./neon-import";

const UNLIMITED = Number.MAX_VALUE;

const branch = (id: string, projectId = "project") =>
  create(NeonBranchSchema, {
    projectId,
    projectTitle: "storefront",
    branchId: id,
    branchTitle: `branch-${id}`,
    databases: [{ name: "db", owner: "owner" }],
    roles: ["owner", "other"],
  });

const run = (count: number): NeonImportRun =>
  createImportRun(
    Array.from({ length: count }, (_, index) => ({
      ...makeDraft(branch(String(index))),
      instanceId: `neon-${index}`,
      selected: true,
    })),
    defaultSharedSettings()
  );

const imported = (entry: NeonImportEntry) =>
  create(InstanceImportResultSchema, {
    outcome: {
      case: "imported",
      value: create(InstanceSchema, {
        name: `instances/${entry.draft.instanceId}`,
      }),
    },
  });

const failure = (
  reason: string,
  creationOutcome = InstanceImportFailure_CreationOutcome.NOT_CREATED,
  metadata: Record<string, string> = {}
) =>
  create(InstanceImportResultSchema, {
    outcome: {
      case: "failure",
      value: {
        creationOutcome,
        status: {
          code: 9,
          message: "Server wording that must not decide anything",
          details: [
            anyPack(
              ErrorInfoSchema,
              create(ErrorInfoSchema, { reason, metadata })
            ),
          ],
        },
      },
    },
  });

const respond = (
  entries: readonly NeonImportEntry[],
  override: Record<number, ReturnType<typeof failure>> = {}
) =>
  create(ImportInstancesResponseSchema, {
    results: entries.map((entry, index) => override[index] ?? imported(entry)),
  });

const connectError = (
  code: Code,
  reason?: string,
  retrySeconds?: number,
  violatedField?: string
) =>
  new ConnectError("Server wording", code, undefined, [
    ...(reason
      ? [{ desc: ErrorInfoSchema, value: create(ErrorInfoSchema, { reason }) }]
      : []),
    ...(violatedField
      ? [
          {
            desc: BadRequestSchema,
            value: create(BadRequestSchema, {
              fieldViolations: [{ field: violatedField, description: "bad" }],
            }),
          },
        ]
      : []),
    ...(retrySeconds
      ? [
          {
            desc: RetryInfoSchema,
            value: create(RetryInfoSchema, {
              retryDelay: { seconds: BigInt(retrySeconds) },
            }),
          },
        ]
      : []),
  ]);

describe("discovery drafts", () => {
  it("merges pages by Neon project and branch ID and keeps selections and edits", () => {
    const original = {
      ...makeDraft(branch("one")),
      title: "Edited",
      selected: true,
    };
    const updated = branch("one");
    updated.roles.push("new-role");
    const merged = mergeDiscovery(
      [original],
      [updated, branch("one", "other-project"), updated]
    );
    expect(merged).toHaveLength(2);
    expect(merged[0]).toMatchObject({
      title: "Edited",
      selected: true,
      instanceId: original.instanceId,
    });
    expect(merged[0].branch.roles).toContain("new-role");
    expect(merged[1].selected).toBe(false);
  });

  it("deselects a branch that comes back with a discovery error", () => {
    const original = { ...makeDraft(branch("one")), selected: true };
    const broken = branch("one");
    broken.discoveryError = {
      $typeName: "google.rpc.Status",
      code: 9,
      message: "",
      details: [],
    };
    expect(mergeDiscovery([original], [broken])[0].selected).toBe(false);
  });

  it("drops a role or database the branch no longer offers", () => {
    const original = {
      ...makeDraft(branch("one")),
      role: "other",
      passwordSource: "typed" as const,
      password: "example-password",
    };
    const changed = branch("one");
    changed.roles = ["owner"];
    changed.databases[0].name = "renamed";
    expect(mergeDiscovery([original], [changed])[0]).toMatchObject({
      role: "",
      password: "",
      database: "renamed",
      databases: [],
    });
  });

  it("restores a carried row only when a restarted traversal returns its branch again", () => {
    const kept = {
      ...makeDraft(branch("kept")),
      selected: true,
      title: "Mine",
    };
    const gone = { ...makeDraft(branch("gone")), selected: true };
    const carried = new Map(
      [kept, gone].map((row) => [branchKey(row.branch), row])
    );
    const merged = mergeDiscovery([], [branch("kept"), branch("new")], carried);
    expect(merged.map((row) => row.branch.branchId)).toEqual(["kept", "new"]);
    expect(merged[0]).toMatchObject({ selected: true, title: "Mine" });
    expect(merged[1].selected).toBe(false);
  });

  it("gives each row a readable, valid, unique instance ID", () => {
    const first = neonInstanceId(branch("one"), new Set());
    expect(first).toBe("neon-storefront-branch-one");
    expect(neonInstanceId(branch("one"), new Set([first]))).toBe(
      "neon-storefront-branch-one-2"
    );
    const long = branch("one");
    long.branchTitle = "Feature/".concat("x".repeat(80));
    const id = neonInstanceId(long, new Set());
    expect(id).toMatch(/^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/);
    const [a, b] = mergeDiscovery([], [branch("same"), branch("same", "p2")]);
    expect(a.instanceId).not.toBe(b.instanceId);
  });

  it("prefills a role only when the databases chosen for sync share one owner", () => {
    const candidate = branch("one");
    candidate.databases.push({
      ...candidate.databases[0],
      name: "reporting",
      owner: "other",
    });
    expect(commonOwner(candidate)).toBe("");
    expect(commonOwner(candidate, ["db"])).toBe("owner");
    expect(commonOwner(candidate, [])).toBe("");
    expect(makeDraft(candidate).role).toBe("");
    const narrowed = withDatabaseToggled(
      { ...makeDraft(candidate), syncAll: false },
      "reporting",
      false
    );
    expect(narrowed).toMatchObject({ databases: ["db"], role: "owner" });
  });

  it("inherits shared settings until a row sets its own", () => {
    const shared = {
      environment: "environments/test",
      syncAll: true,
      syncSeconds: 3600,
    };
    const inherits = makeDraft(branch("one"));
    const overrides = {
      ...makeDraft(branch("two")),
      environment: "",
      syncSeconds: 0,
    };
    expect(effectiveSettings(inherits, shared)).toEqual(shared);
    expect(effectiveSettings(overrides, shared)).toEqual({
      environment: "",
      syncAll: true,
      syncSeconds: 0,
    });
    expect(countOverrides([inherits, overrides])).toEqual({
      environment: 1,
      syncAll: 0,
      syncSeconds: 1,
    });
  });

  it("clears a typed password when the role changes", () => {
    const draft = {
      ...makeDraft(branch("one")),
      passwordSource: "typed" as const,
      password: "example-password",
    };
    expect(withRole(draft, "owner").password).toBe("example-password");
    expect(withRole(draft, "other")).toMatchObject({
      role: "other",
      password: "",
    });
  });

  it("reports what a row still needs before import", () => {
    const ready = makeDraft(branch("one"));
    expect(draftNeed(ready)).toBeUndefined();
    expect(draftNeed({ ...ready, role: "" })).toBe("role");
    expect(draftNeed({ ...ready, passwordSource: "typed" })).toBe("password");
    expect(draftNeed({ ...ready, title: " " })).toBe("title");
    expect(draftNeed({ ...ready, instanceId: "Not Valid" })).toBe(
      "instance-id"
    );
    const twin = { ...makeDraft(branch("two")), instanceId: ready.instanceId };
    expect(draftNeed(ready, duplicateInstanceIds([ready, twin]))).toBe(
      "instance-id"
    );
  });
});

describe("instance limits", () => {
  const license = {
    instanceCountLimit: 20,
    totalInstanceCount: 5,
    instanceLicenseCount: 20,
    activatedInstanceCount: 5,
  };

  it("counts what the workspace can still add under a unified license", () => {
    expect(remainingInstanceCapacity(license)).toBe(15);
    expect(importsActivated(license)).toBe(true);
  });

  it("caps a split license by its remaining activations", () => {
    const split = {
      ...license,
      instanceCountLimit: 50,
      instanceLicenseCount: 10,
      activatedInstanceCount: 7,
    };
    expect(remainingInstanceCapacity(split)).toBe(3);
    expect(
      remainingInstanceCapacity({ ...split, instanceCountLimit: UNLIMITED })
    ).toBe(3);
  });

  it("treats an unlimited license as no limit", () => {
    expect(
      remainingInstanceCapacity({
        ...license,
        instanceCountLimit: UNLIMITED,
        instanceLicenseCount: UNLIMITED,
      })
    ).toBeUndefined();
  });

  it("imports without activation where the license provides none", () => {
    const free = {
      ...license,
      instanceCountLimit: 10,
      instanceLicenseCount: 0,
      activatedInstanceCount: 0,
    };
    expect(importsActivated(free)).toBe(false);
    expect(remainingInstanceCapacity(free)).toBe(5);
    expect(remainingInstanceCapacity({ ...free, totalInstanceCount: 12 })).toBe(
      0
    );
  });
});

describe("structured request failures", () => {
  it.each([
    [
      connectError(Code.FailedPrecondition, "VENDOR_CREDENTIAL_INVALID"),
      false,
      "key-invalid",
    ],
    [
      connectError(Code.ResourceExhausted, "VENDOR_RATE_LIMITED", 25),
      false,
      "rate-limited",
    ],
    [
      connectError(Code.Unavailable, "VENDOR_UNAVAILABLE"),
      false,
      "unavailable",
    ],
    [
      connectError(Code.DeadlineExceeded, "VENDOR_UNAVAILABLE"),
      false,
      "unavailable",
    ],
    [connectError(Code.PermissionDenied), false, "permission-denied"],
    [
      connectError(Code.FailedPrecondition, "VENDOR_ACCESS_DENIED"),
      false,
      "project-id-required",
    ],
    [
      connectError(Code.FailedPrecondition, "VENDOR_RESOURCE_NOT_FOUND"),
      true,
      "project-id-unreadable",
    ],
    [
      connectError(Code.InvalidArgument, undefined, undefined, "project_id"),
      true,
      "project-id-invalid",
    ],
    [
      connectError(Code.InvalidArgument, undefined, undefined, "page_size"),
      true,
      "other",
    ],
    [
      connectError(Code.InvalidArgument, "INVALID_PAGE_TOKEN"),
      false,
      "token-invalid",
    ],
    [connectError(Code.NotFound, "DESTINATION_NOT_FOUND"), false, "other"],
    [new Error("offline"), false, "other"],
  ])("classifies %s", (error, hasProjectFilter, kind) => {
    expect(classifyRequestFailure(error, hasProjectFilter).kind).toBe(kind);
  });

  it("carries the retry delay Neon asked for", () => {
    const limited = connectError(
      Code.ResourceExhausted,
      "VENDOR_RATE_LIMITED",
      25
    );
    expect(classifyRequestFailure(limited, false).retryAfterSeconds).toBe(25);
    expect(
      classifyRequestFailure(connectError(Code.Unavailable), false)
        .retryAfterSeconds
    ).toBe(0);
  });
});

describe("automatic import batches", () => {
  it("sends 45 entries as sequential batches of 20, 20, and 5", async () => {
    const base = run(45);
    const sizes: number[] = [];
    let active = 0;
    const send = vi.fn(async (entries: readonly NeonImportEntry[]) => {
      expect(active++).toBe(0);
      sizes.push(entries.length);
      await Promise.resolve();
      active--;
      return respond(entries);
    });
    const updates: NeonImportRun[] = [];
    const result = await runImportBatches(
      base,
      unsentGroups(base),
      send,
      (next) => updates.push(next)
    );
    expect(sizes).toEqual([20, 20, 5]);
    expect(result.paused).toBe(false);
    expect(summarizeRun(result.run)).toMatchObject({
      created: 45,
      processed: 45,
    });
    expect(base.entries.every((entry) => entry.state === "unsent")).toBe(true);
    // The active batch is awaiting as a whole; nothing in it reports early.
    expect(summarizeRun(updates[0])).toMatchObject({
      awaiting: 20,
      unsent: 25,
      processed: 0,
    });
    expect(result.run.batches.every((batch) => batch.sentAt)).toBe(true);
  });

  it("continues past confirmed failures and reads the reason from ErrorInfo", async () => {
    const base = run(21);
    const send = vi.fn(async (entries: readonly NeonImportEntry[]) =>
      respond(
        entries,
        send.mock.calls.length === 1
          ? { 0: failure("VENDOR_PASSWORD_REQUIRED") }
          : {}
      )
    );
    const result = await runImportBatches(
      base,
      unsentGroups(base),
      send,
      vi.fn()
    );
    expect(send).toHaveBeenCalledTimes(2);
    expect(result.paused).toBe(false);
    expect(result.run.entries[0]).toMatchObject({
      state: "failed",
      reason: "VENDOR_PASSWORD_REQUIRED",
    });
    expect(result.run.entries[20].state).toBe("created");
  });

  it.each(["lost", "incomplete", "reported"] as const)(
    "pauses on a %s outcome and leaves later batches unsent",
    async (kind) => {
      const base = run(21);
      const send = vi.fn(async (entries: readonly NeonImportEntry[]) => {
        if (kind === "lost") throw new Error("Disconnected");
        const response = respond(
          entries,
          kind === "reported"
            ? {
                0: failure(
                  "INSTANCE_CREATION_FAILED",
                  InstanceImportFailure_CreationOutcome.UNKNOWN
                ),
              }
            : {}
        );
        if (kind === "incomplete") response.results.pop();
        return response;
      });
      const result = await runImportBatches(
        base,
        unsentGroups(base),
        send,
        vi.fn()
      );
      expect(result.paused).toBe(true);
      expect(send).toHaveBeenCalledTimes(1);
      expect(result.run.entries[0]).toMatchObject({
        state: "unknown",
        unknownCause: kind === "reported" ? "reported" : "lost",
      });
      expect(result.run.entries[20].state).toBe("unsent");
      // Continuing sends only what was never sent.
      expect(unsentGroups(result.run)).toEqual([[20]]);
    }
  );

  it.each([
    [Code.InvalidArgument, "invalid"],
    [Code.Unauthenticated, "unauthenticated"],
    [Code.PermissionDenied, "permission-denied"],
    [Code.NotFound, "destination-not-found"],
  ] as const)(
    "treats a request rejected with code %s as confirmed not created and pauses",
    async (code, rejection) => {
      const base = run(21);
      const result = await runImportBatches(
        base,
        unsentGroups(base),
        async () => {
          // An interceptor rejection carries no ErrorInfo reason.
          throw connectError(code);
        },
        vi.fn()
      );
      expect(result.paused).toBe(true);
      expect(result.run.entries[0]).toMatchObject({
        state: "failed",
        rejection,
        reason: "",
      });
      expect(result.run.batches[0].cause).toBeUndefined();
      expect(entryRetryable(result.run.entries[0])).toBe(true);
      expect(result.run.entries[20].state).toBe("unsent");
    }
  );

  it.each([
    Code.Internal,
    Code.Unavailable,
    Code.DeadlineExceeded,
    Code.Canceled,
    Code.Unknown,
    Code.ResourceExhausted,
    Code.FailedPrecondition,
    Code.AlreadyExists,
  ])(
    "leaves entries unknown when the request fails with code %s",
    async (code) => {
      expect(requestRejection(connectError(code))).toBeUndefined();
      const base = run(2);
      const result = await runImportBatches(
        base,
        unsentGroups(base),
        async () => {
          throw connectError(code);
        },
        vi.fn()
      );
      expect(result.run.entries[0]).toMatchObject({
        state: "unknown",
        unknownCause: "lost",
      });
    }
  );

  it("clears an earlier rejection when the entry is sent again", async () => {
    const base = run(1);
    const rejected = await runImportBatches(
      base,
      unsentGroups(base),
      async () => {
        throw connectError(Code.PermissionDenied);
      },
      vi.fn()
    );
    const retried = await runImportBatches(
      rejected.run,
      chunkIndices([0]),
      async (entries) =>
        respond(entries, {
          0: failure("INSTANCE_CONNECTION_FAILED", undefined, {
            category: "auth_failed",
          }),
        }),
      vi.fn(),
      { retry: true }
    );
    expect(retried.run.entries[0]).toMatchObject({
      state: "failed",
      reason: "INSTANCE_CONNECTION_FAILED",
      category: "auth_failed",
    });
    expect(retried.run.entries[0].rejection).toBeUndefined();
  });

  it("marks the active batch unknown when the user stops, and sends nothing more", async () => {
    const base = run(21);
    const controller = new AbortController();
    const send = vi.fn(async () => {
      controller.abort();
      throw new DOMException("Aborted", "AbortError");
    });
    const result = await runImportBatches(
      base,
      unsentGroups(base),
      send,
      vi.fn(),
      {
        signal: controller.signal,
      }
    );
    expect(result.paused).toBe(true);
    expect(send).toHaveBeenCalledTimes(1);
    expect(result.run.entries[0]).toMatchObject({
      state: "unknown",
      unknownCause: "stopped",
    });
    expect(result.run.batches[0].cause).toBe("stopped");
    expect(result.run.entries[20].state).toBe("unsent");
  });
});

describe("recovery", () => {
  const failedRun = async (reason: string) => {
    const base = run(1);
    const result = await runImportBatches(
      base,
      unsentGroups(base),
      async (entries) => respond(entries, { 0: failure(reason) }),
      vi.fn()
    );
    return result.run;
  };

  it("retries a missing password only after one is typed", async () => {
    let current = await failedRun("VENDOR_PASSWORD_REQUIRED");
    const { key } = current.entries[0];
    expect(entryRetryable(current.entries[0])).toBe(false);
    current = updateEntryDraft(current, key, (draft) => ({
      ...draft,
      passwordSource: "typed",
      password: "example-password",
    }));
    expect(entryRetryable(current.entries[0])).toBe(true);
    const retried = await runImportBatches(
      current,
      chunkIndices([0]),
      async (entries) => {
        expect(entries[0].draft.password).toBe("example-password");
        return respond(entries);
      },
      vi.fn(),
      { retry: true }
    );
    expect(retried.run.entries[0]).toMatchObject({
      state: "created",
      resolvedBy: "retry",
      draft: { instanceId: "neon-0", password: "" },
    });
  });

  it.each([
    ["VENDOR_USER_NOT_FOUND", { role: "other" }],
    ["VENDOR_DATABASE_NOT_FOUND", { database: "replacement" }],
    ["INSTANCE_ID_EXISTS", { instanceId: "neon-renamed" }],
  ])("retries %s only after its cause is corrected", async (reason, change) => {
    let current = await failedRun(reason);
    expect(entryRetryable(current.entries[0])).toBe(false);
    current = updateEntryDraft(current, current.entries[0].key, (draft) => ({
      ...draft,
      ...change,
    }));
    expect(entryRetryable(current.entries[0])).toBe(true);
  });

  it("retries a limit rejection and an unfamiliar reason without correction", async () => {
    expect(
      entryRetryable((await failedRun("INSTANCE_LIMIT_EXCEEDED")).entries[0])
    ).toBe(true);
    expect(entryRetryable((await failedRun("SOMETHING_NEW")).entries[0])).toBe(
      true
    );
  });

  it("holds back an entry whose instance ID another entry of the run uses", async () => {
    let current = await failedRun("INSTANCE_ID_EXISTS");
    current = updateEntryDraft(current, current.entries[0].key, (draft) => ({
      ...draft,
      instanceId: "neon-taken",
    }));
    expect(entryRetryable(current.entries[0], new Set(["neon-taken"]))).toBe(
      false
    );
    expect(entryRetryable(current.entries[0], new Set(["neon-other"]))).toBe(
      true
    );
  });

  it("resolves a found unknown entry as created and records a check time only for a confirmed absence", async () => {
    const base = run(3);
    const lost = await runImportBatches(
      base,
      unsentGroups(base),
      async () => {
        throw new Error("Disconnected");
      },
      vi.fn()
    );
    const [found, missing, failedCheck] = lost.run.entries;
    const at = new Date("2026-11-10T06:02:07Z");
    const checked = applyInstanceChecks(
      lost.run,
      [
        {
          key: found.key,
          result: "found",
          instance: create(InstanceSchema, { name: "instances/neon-0" }),
        },
        { key: missing.key, result: "missing" },
        { key: failedCheck.key, result: "error" },
      ],
      at
    );
    expect(checked.entries[0]).toMatchObject({
      state: "created",
      resolvedBy: "check",
    });
    expect(checked.entries[1]).toMatchObject({
      state: "unknown",
      checkedAt: at,
    });
    expect(checked.entries[2]).toMatchObject({
      state: "unknown",
      checkFailed: true,
    });
    expect(checked.entries[2].checkedAt).toBeUndefined();
  });
});

describe("import requests", () => {
  const draft = () => run(1).entries[0].draft;

  it("names the credential once and gives each entry its Neon branch source", () => {
    const request = toImportRequest(
      "projects/payments",
      "example-key",
      [draft()],
      "editable"
    );
    expect(request).toMatchObject({
      parent: "projects/payments",
      credential: { case: "neonCredential", value: { apiKey: "example-key" } },
    });
    expect(request.imports).toHaveLength(1);
    expect(request.imports[0]).not.toHaveProperty("activation");
    expect(request.imports[0]).toMatchObject({
      source: {
        case: "neonBranch",
        value: { projectId: "project", branchId: "0" },
      },
      username: "owner",
      connectionDatabase: "db",
      password: undefined,
      instanceId: "neon-0",
      instance: {
        environment: undefined,
        syncInterval: { seconds: 3600n },
        syncDatabases: undefined,
      },
    });
  });

  it("sends a typed password, a chosen subset, and a schedule that is off", () => {
    const [entry] = toImportRequest(
      undefined,
      "example-key",
      [
        {
          ...draft(),
          passwordSource: "typed",
          password: "example-password",
          syncAll: false,
          databases: [],
          syncSeconds: 0,
          environment: "environments/prod",
        },
      ],
      "editable"
    ).imports;
    expect(entry).toMatchObject({
      password: "example-password",
      instance: {
        environment: "environments/prod",
        syncInterval: { seconds: 0n },
        syncDatabases: { databases: [] },
      },
    });
  });

  it("turns the schedule off where instances are not activated, and sends the default interval where the plan fixes it", () => {
    expect(scheduleMode(false, true)).toBe("off");
    expect(scheduleMode(true, false)).toBe("default");
    expect(scheduleMode(true, true)).toBe("editable");
    const interval = (mode: "off" | "default") =>
      toImportRequest(
        undefined,
        "example-key",
        [{ ...draft(), syncSeconds: 21600 }],
        mode
      ).imports[0].instance.syncInterval;
    expect(interval("off")).toEqual({ seconds: 0n });
    expect(interval("default")).toEqual({ seconds: 3600n });
  });
});

describe("structured reasons", () => {
  const t = ((key: string) => key) as never;

  it("refines a connection failure by its category", () => {
    const text = (category: string) =>
      neonReasonText(t, "INSTANCE_CONNECTION_FAILED", "server", category);
    expect(text("auth_failed")).toBe(
      "instance.neon.reason.connection-auth-failed"
    );
    expect(text("timeout")).toBe("instance.neon.reason.connection-unreachable");
    expect(text("unknown")).toBe("instance.neon.reason.connection-failed");
  });

  it("falls back to the server message only for a reason it does not know", () => {
    expect(neonReasonText(t, "VENDOR_USER_NOT_FOUND", "server")).toBe(
      "instance.neon.reason.role-not-found"
    );
    expect(neonReasonText(t, "SOMETHING_NEW", "server")).toBe("server");
    expect(neonReasonText(t, "", "")).toBe("instance.neon.reason.other");
  });

  it("labels the schedule choices", () => {
    expect(syncIntervalLabel(t, 3600)).toBe("instance.neon.interval.hourly");
    expect(syncIntervalLabel(t, 21600)).toBe(
      "instance.neon.interval.every-n-hours"
    );
    expect(syncIntervalLabel(t, 0)).toBe("instance.neon.interval.off");
  });
});

describe("manual entry", () => {
  it("reads the connection string the Neon console shows", () => {
    expect(
      parseNeonConnectionString(
        "psql 'postgresql://app_owner:p%40ss@ep-example-123456.us-east-2.aws.neon.tech/storefront?sslmode=require'"
      )
    ).toEqual({
      role: "app_owner",
      password: "p@ss",
      host: "ep-example-123456.us-east-2.aws.neon.tech",
      port: "5432",
      database: "storefront",
    });
    expect(
      parseNeonConnectionString("postgres://role@db.example.com:6543/app")
    ).toMatchObject({ password: "", port: "6543" });
    expect(
      parseNeonConnectionString("mysql://role@db.example.com/app")
    ).toBeUndefined();
    expect(parseNeonConnectionString("ep-example.neon.tech")).toBeUndefined();
  });

  it("recognizes a pooled endpoint host", () => {
    expect(
      isNeonPooledHost("ep-example-123456-pooler.us-east-2.aws.neon.tech")
    ).toBe(true);
    expect(isNeonPooledHost("ep-example-123456.us-east-2.aws.neon.tech")).toBe(
      false
    );
  });
});
