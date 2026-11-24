import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  command,
  failedAttempt,
  passedAttempt,
  releaseFile,
  ts,
} from "@/components/task-run-log/taskRunLogEntries";
import {
  TaskRun_Status as Status,
  type TaskRunLogEntry,
  TaskRunLogEntrySchema,
  TaskSchema,
  TaskRunLogEntry_Type as Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import { buildAgentDiagnosticPrompt } from "./diagnosticHelp";
import {
  type AgentTaskRunDiagnosticInput,
  getTaskRunDiagnostic,
} from "./taskRunDiagnostics";

const project = "projects/example";
const planName = `${project}/plans/42`;
const taskName = `${planName}/rollout/stages/prod/tasks/5`;
const name = `${taskName}/taskRuns/8`;
const issue = {
  kind: "issue",
  project,
  name: `${project}/issues/19`,
  label: "Issue",
} as const;
const input = (status = Status.FAILED): AgentTaskRunDiagnosticInput => ({
  resource: { kind: "plan", name: planName, project, label: "Plan" },
  planName,
  run: { name, status, detail: "" },
  provenance: "latest",
});
const log = (entries: TaskRunLogEntry[], status = Status.DONE) => ({
  entries,
  fetchStatus: "success" as const,
  snapshot: { taskRunName: name, status },
});
const step = (
  type: Type,
  field:
    | "schemaDump"
    | "databaseSync"
    | "priorBackup"
    | "computeDiff"
    | "ghostMigration",
  { endTime = ts(3), error = "Recorded step error", replicaId = "" } = {}
) =>
  create(TaskRunLogEntrySchema, {
    type,
    logTime: ts(2),
    replicaId,
    [field]: { startTime: ts(2), endTime, error },
  });
const diffError = (replicaId = "") =>
  step(Type.COMPUTE_DIFF, "computeDiff", { replicaId });
const facts = (ctx: ReturnType<typeof getTaskRunDiagnostic>) =>
  Object.fromEntries(
    ctx?.facts?.map(({ label, value }) => [label, value]) ?? []
  );

describe("task run diagnostic evidence", () => {
  test("a failed Plan run remains eligible without task metadata, detail, or logs", () => {
    const ctx = getTaskRunDiagnostic(input());
    expect(ctx).toMatchObject({
      kind: "task-run",
      sourceId: name,
      resource: { kind: "plan", name: planName, project, label: planName },
    });
    expect(facts(ctx)).toMatchObject({
      Status: "FAILED",
      Provenance: "latest",
      Logs: "Unavailable",
    });
    expect(ctx).not.toHaveProperty("sqlFixSource");
    expect(ctx).not.toHaveProperty("sql");
  });

  test.each(["latest", "explicit-history"] as const)(
    "accepts the valid create environment for %s failed Plan runs",
    (provenance) => {
      const createTask = taskName.replace("/stages/prod/", "/stages/create/");
      const createRun = `${createTask}/taskRuns/8`;
      const ctx = getTaskRunDiagnostic({
        ...input(),
        provenance,
        run: { ...input().run, name: createRun },
        task: create(TaskSchema, { name: createTask }),
      });
      expect(ctx).toMatchObject({
        sourceId: createRun,
        resource: { name: planName },
      });
      expect(facts(ctx)).toMatchObject({
        Status: "FAILED",
        Provenance: provenance,
      });
    }
  );

  test.each(["latest", "explicit-history"] as const)(
    "accepts the valid create environment for %s completed runs with step errors",
    (provenance) => {
      const createRun = name.replace("/stages/prod/", "/stages/create/");
      const ctx = getTaskRunDiagnostic({
        ...input(Status.DONE),
        provenance,
        run: { ...input(Status.DONE).run, name: createRun },
        log: {
          ...log([diffError()]),
          snapshot: { taskRunName: createRun, status: Status.DONE },
        },
      });
      expect(ctx).toMatchObject({
        sourceId: createRun,
        detail: "Recorded step error",
      });
      expect(facts(ctx)).toMatchObject({
        Status: "DONE",
        Provenance: provenance,
      });
    }
  );

  test.each(["latest", "explicit-history"] as const)(
    "accepts the valid create environment for %s database-creation failures",
    (provenance) => {
      const createTask = taskName.replace("/stages/prod/", "/stages/create/");
      const createRun = `${createTask}/taskRuns/8`;
      const ctx = getTaskRunDiagnostic({
        ...input(),
        resource: issue,
        provenance,
        run: { ...input().run, name: createRun },
        task: create(TaskSchema, {
          name: createTask,
          target: "instances/example",
          payload: {
            case: "databaseCreate",
            value: { sheet: `${project}/sheets/2` },
          },
        }),
      });
      expect(ctx).toMatchObject({
        sourceId: createRun,
        resource: { kind: "issue", name: issue.name },
      });
      expect(facts(ctx)).toMatchObject({
        Status: "FAILED",
        Provenance: provenance,
      });
    }
  );

  test.each([
    [Type.SCHEMA_DUMP, "schemaDump"],
    [Type.DATABASE_SYNC, "databaseSync"],
    [Type.PRIOR_BACKUP, "priorBackup"],
    [Type.COMPUTE_DIFF, "computeDiff"],
    [Type.GHOST_MIGRATION, "ghostMigration"],
  ] as const)("DONE includes completed one-time %s errors", (type, field) => {
    expect(
      getTaskRunDiagnostic({
        ...input(Status.DONE),
        log: log([step(type, field)]),
      })
    ).toMatchObject({
      detail: "Recorded step error",
    });
  });

  test("DONE ignores missing or unfinished one-time payloads", () => {
    const incomplete = step(Type.COMPUTE_DIFF, "computeDiff");
    incomplete.computeDiff!.endTime = undefined;
    expect(
      getTaskRunDiagnostic({
        ...input(Status.DONE),
        log: log([
          create(TaskRunLogEntrySchema, { type: Type.COMPUTE_DIFF }),
          incomplete,
        ]),
      })
    ).toBeUndefined();
  });

  test("DONE ignores failed command, transaction, and retry history", () => {
    const attempt = failedAttempt(2, 1);
    attempt[2].transactionControl!.error = "Historical transaction error";
    expect(
      getTaskRunDiagnostic({
        ...input(Status.DONE),
        log: log([...attempt, ...passedAttempt(5)]),
      })
    ).toBeUndefined();
  });

  test.each([
    Status.CANCELED,
    Status.RUNNING,
    Status.PENDING,
    Status.AVAILABLE,
    Status.STATUS_UNSPECIFIED,
  ])("status %s is not eligible even with error logs", (status) => {
    expect(
      getTaskRunDiagnostic({
        ...input(status),
        log: log([diffError()], status),
      })
    ).toBeUndefined();
  });

  test.each([
    {
      fetchStatus: "success",
      snapshot: { taskRunName: name, status: Status.RUNNING },
    },
    {
      fetchStatus: "error",
      snapshot: { taskRunName: name, status: Status.DONE },
    },
    {
      fetchStatus: "success",
      snapshot: { taskRunName: `${name}9`, status: Status.DONE },
    },
    { fetchStatus: "success", snapshot: undefined },
  ] as const)(
    "DONE refuses a non-final or unrelated snapshot: %j",
    (presentation) => {
      expect(
        getTaskRunDiagnostic({
          ...input(Status.DONE),
          log: { entries: [diffError()], ...presentation },
        })
      ).toBeUndefined();
    }
  );

  test.each([
    ["blue", "green"],
    ["", "unknown"],
  ])(
    "replica streams %s and %s cannot establish DONE attribution",
    (first, second) => {
      expect(
        getTaskRunDiagnostic({
          ...input(Status.DONE),
          log: log([diffError(first), command(4, 5, { replicaId: second })]),
        })
      ).toBeUndefined();
    }
  );

  test("FAILED does not attribute a log from before its terminal status", () => {
    const ctx = getTaskRunDiagnostic({
      ...input(),
      log: log([diffError("blue")], Status.RUNNING),
    });
    expect(ctx?.detail).toBeUndefined();
    expect(facts(ctx)).not.toHaveProperty("Recorded replica");
  });

  test("an unrelated later file success does not erase a diff error in a legacy stream; tied timestamps use append order", () => {
    const ctx = getTaskRunDiagnostic({
      ...input(Status.DONE),
      log: log([
        releaseFile(2, "v1", "migrations/1.sql"),
        diffError(),
        releaseFile(2, "v2", "migrations/2.sql"),
        command(2, 3),
      ]),
    });
    expect(facts(ctx)).toMatchObject({
      "Recorded file": "migrations/1.sql",
      "Recorded version": "v1",
    });
    expect(facts(ctx)).not.toHaveProperty("Recorded replica");
  });

  test("a task-level sync after a file marker does not borrow its file identity", () => {
    const ctx = getTaskRunDiagnostic({
      ...input(Status.DONE),
      log: log([
        releaseFile(2, "v1", "migrations/1.sql"),
        step(Type.DATABASE_SYNC, "databaseSync"),
      ]),
    });
    expect(ctx?.detail).toBe("Recorded step error");
    expect(facts(ctx)).not.toHaveProperty("Recorded file");
  });

  test("records the failed command's historical file, replica, and byte range without its SQL", () => {
    const entry = command(2, 3, { error: "Recorded error", replicaId: "blue" });
    entry.commandExecute!.range = create(TaskRunLogEntrySchema, {
      commandExecute: { range: { start: 7, end: 29 } },
    }).commandExecute?.range;
    const ctx = getTaskRunDiagnostic({
      ...input(),
      run: {
        ...input().run,
        detail:
          "failed to execute release file migrations/1.sql (version v1): Recorded error",
      },
      provenance: "explicit-history",
      task: create(TaskSchema, {
        name: taskName,
        target: "instances/example/databases/books",
      }),
      log: log(
        [
          releaseFile(2, "v1", "migrations/1.sql", { replicaId: "blue" }),
          entry,
        ],
        Status.FAILED
      ),
    });
    expect(facts(ctx)).toMatchObject({
      Status: "FAILED",
      Provenance: "explicit-history",
      "Recorded target": "instances/example/databases/books",
      "Recorded file": "migrations/1.sql",
      "Recorded version": "v1",
      "Recorded replica": "blue",
      "Recorded byte range": "[7, 29)",
    });
    expect(JSON.stringify(ctx)).not.toContain("ALTER TABLE");
  });

  test.each([
    name.replace("/rollout/", "/rollouts/"),
    name.replace("/plans/42/", "/plans/-/"),
    name.replace("/plans/42/", "/plans/create/"),
    name.replace("/plans/42/", "/plans/43/"),
    name.replace("/example/", "/other/"),
    name.replace("/stages/prod/", "/stages/*/"),
    name.replace("/tasks/5/", "/tasks/0/"),
    name.replace("/tasks/5/", "/tasks/create/"),
    name.replace("/taskRuns/8", "/taskRuns/create"),
    `${name}/extra`,
    `${name}\n`,
  ])("rejects malformed or foreign run names: %s", (invalid) => {
    expect(
      getTaskRunDiagnostic({
        ...input(),
        run: { ...input().run, name: invalid },
      })
    ).toBeUndefined();
  });

  test("rejects a foreign task with a colliding UID", () => {
    const task = create(TaskSchema, {
      name: taskName.replace("/stages/prod/", "/stages/test/"),
    });
    expect(getTaskRunDiagnostic({ ...input(), task })).toBeUndefined();
  });

  test("Issue evidence requires the linked Plan and exact create-database task", () => {
    const task = create(TaskSchema, {
      name: taskName,
      target: "instances/example",
      payload: {
        case: "databaseCreate",
        value: { sheet: `${project}/sheets/2` },
      },
    });
    const bound = { ...input(), resource: issue };
    expect(getTaskRunDiagnostic(bound)).toBeUndefined();
    expect(
      getTaskRunDiagnostic({
        ...bound,
        task: create(TaskSchema, { name: taskName }),
      })
    ).toBeUndefined();
    expect(
      getTaskRunDiagnostic({ ...bound, task, planName: `${project}/plans/43` })
    ).toBeUndefined();
    expect(getTaskRunDiagnostic({ ...bound, task })).toMatchObject({
      sourceId: name,
      resource: { name: issue.name, kind: "issue", project },
    });
  });

  test("escape-heavy execution detail cannot crowd out status or provenance", () => {
    const ctx = getTaskRunDiagnostic({
      ...input(),
      provenance: "explicit-history",
      run: {
        ...input().run,
        detail: `password='never-persist' ${"\u0001".repeat(1900)}`,
      },
    });
    const prompt = buildAgentDiagnosticPrompt(ctx!);
    const snapshot = JSON.parse(prompt.slice(prompt.indexOf("\n") + 1));
    expect(snapshot).toMatchObject({
      kind: "task-run",
      sourceId: name,
      resource: { name: planName },
      truncated: expect.arrayContaining(["detail", "facts"]),
    });
    expect(snapshot.summary).toContain("FAILED");
    expect(snapshot.summary).toContain("explicit-history");
    expect(snapshot.detail).toContain("[redacted]");
    expect(prompt).not.toContain("never-persist");
    expect(new TextEncoder().encode(prompt).byteLength).toBeLessThanOrEqual(
      6000
    );
  });

  test("prompt retains status, binding and provenance and bounds redacted Unicode log evidence", () => {
    const ctx = getTaskRunDiagnostic({
      ...input(),
      provenance: "explicit-history",
      run: {
        ...input().run,
        detail: `password='never-persist' ${"例🙂".repeat(3000)}`,
      },
    });
    const prompt = buildAgentDiagnosticPrompt(ctx!);
    const snapshot = JSON.parse(prompt.slice(prompt.indexOf("\n") + 1));
    expect(snapshot).toMatchObject({
      kind: "task-run",
      sourceId: name,
      resource: { name: planName },
      truncated: ["detail"],
    });
    expect(
      Object.fromEntries(
        snapshot.facts.map(
          ({ label, value }: { label: string; value: string }) => [label, value]
        )
      )
    ).toMatchObject({
      Status: "FAILED",
      Provenance: "explicit-history",
    });
    expect(snapshot.detail).toContain("[redacted]");
    expect(prompt).not.toContain("never-persist");
    expect(new TextEncoder().encode(prompt).byteLength).toBeLessThanOrEqual(
      6000
    );
  });
});
