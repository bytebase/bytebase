import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  ConstantSchema,
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { AgentDiagnosticCapability } from "../types";
import type { PreparedApiCall } from "./callApi";
import { isDiagnosticApiCallInScope } from "./diagnosticApiScope";

const api = vi.hoisted(() => ({ getDatabase: vi.fn(), batchParse: vi.fn() }));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: api.getDatabase },
  celServiceClientConnect: { batchParse: api.batchParse },
}));

const capability: AgentDiagnosticCapability = {
  kind: "diagnostic",
  resource: {
    kind: "plan",
    name: "projects/example/plans/12",
    project: "projects/example",
    label: "Selected plan",
  },
};

const prepared = (
  operation: string,
  body: Record<string, unknown>
): PreparedApiCall => ({
  operationId: `bytebase.v1.${operation.replace("/", ".")}`,
  path: `/bytebase.v1.${operation}`,
  body: JSON.stringify(body),
});

const projectFilter = (project: string): Expr =>
  create(ExprSchema, {
    exprKind: {
      case: "callExpr",
      value: {
        function: "_==_",
        args: [
          create(ExprSchema, {
            exprKind: { case: "identExpr", value: { name: "project" } },
          }),
          create(ExprSchema, {
            exprKind: {
              case: "constExpr",
              value: create(ConstantSchema, {
                constantKind: { case: "stringValue", value: project },
              }),
            },
          }),
        ],
      },
    },
  });

describe("isDiagnosticApiCallInScope", () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it("admits an annotated project resource read", async () => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("PlanService/GetPlan", {
          name: "projects/example/plans/12",
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each(["create", "plans", "tasks"])(
    "admits a task-run session in the valid %s environment",
    async (stage) => {
      await expect(
        isDiagnosticApiCallInScope(
          prepared("RolloutService/GetTaskRunSession", {
            parent: `projects/example/plans/12/rollout/stages/${stage}/tasks/4/taskRuns/5`,
          }),
          capability
        )
      ).resolves.toBe(true);
    }
  );

  it("admits a rollout selector for the valid create environment", async () => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("RolloutService/CreateRollout", {
          parent: "projects/example/plans/12",
          target: "environments/create",
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each([
    "instances/create/databases/app",
    "instances/plans/databases/create",
    "instances/tasks/databases/create",
    "projects/example/instances/plans/databases/create",
  ])(
    "admits a verified database with a named resource ID: %s",
    async (name) => {
      api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/GetDatabase", { name }),
          capability
        )
      ).resolves.toBe(true);
    }
  );

  it.each(["create", "not-an-id", "0", "-1"])(
    "rejects invalid numeric resource ID %s in its canonical position",
    async (id) => {
      const calls = [
        prepared("PlanService/GetPlan", {
          name: `projects/example/plans/${id}`,
        }),
        prepared("IssueService/ApproveIssue", {
          name: `projects/example/issues/${id}`,
        }),
        prepared("RolloutService/GetTaskRunSession", {
          parent: `projects/example/plans/${id}/rollout/stages/create/tasks/4/taskRuns/5`,
        }),
        prepared("RolloutService/ListTaskRuns", {
          parent: `projects/example/plans/12/rollout/stages/create/tasks/${id}`,
        }),
        prepared("RolloutService/GetTaskRunSession", {
          parent: `projects/example/plans/12/rollout/stages/create/tasks/4/taskRuns/${id}`,
        }),
      ];
      for (const call of calls) {
        await expect(
          isDiagnosticApiCallInScope(call, capability)
        ).resolves.toBe(false);
      }
    }
  );

  it.each([
    ["ProjectService/GetProject", { name: "projects/example" }],
    ["IssueService/ApproveIssue", { name: "projects/example/issues/4" }],
    [
      "ProjectService/UpdateProject",
      {
        project: { name: "projects/example", title: "Renamed" },
        updateMask: "title",
      },
    ],
    [
      "IssueService/CreateIssue",
      { parent: "projects/example", issue: { title: "Investigate" } },
    ],
    [
      "RolloutService/GetTaskRunSession",
      {
        parent:
          "projects/example/plans/12/rollout/stages/prod/tasks/4/taskRuns/5",
      },
    ],
    [
      "RolloutService/CreateRollout",
      { parent: "projects/example/plans/12", target: "environments/prod" },
    ],
    [
      "InstanceService/GetInstance",
      { name: "projects/example/instances/primary" },
    ],
  ])("admits scoped %s regardless of method class", async (operation, body) => {
    await expect(
      isDiagnosticApiCallInScope(prepared(operation, body), capability)
    ).resolves.toBe(true);
  });

  it.each([
    ["ProjectService/ListProjects", {}],
    ["ProjectService/SearchProjects", { filter: 'name == "projects/example"' }],
    ["ProjectService/CreateProject", { project: { name: "projects/example" } }],
    ["SettingService/GetSetting", { name: "settings/bb.workspace.profile" }],
    ["InstanceService/GetInstance", { name: "instances/shared" }],
    ["InstanceService/ListInstances", {}],
    ["PlanService/GetPlan", {}],
    ["PlanService/GetPlan", { name: "" }],
    ["PlanService/GetPlan", { name: "projects/example" }],
    ["PlanService/GetPlan", { name: "projects/example/issues/12" }],
    ["PlanService/GetPlan", { name: "projects/other/plans/12" }],
    ["PlanService/GetPlan", { name: "projects/-/plans/12" }],
    ["PlanService/GetPlan", { name: "projects/example/plans/*" }],
    ["PlanService/GetPlan", { name: "projects/example/plans/create" }],
    ["PlanService/GetPlan", { name: "projects/example/plans/../other" }],
    ["PlanService/GetPlan", { name: "projects/example/plans/%2e%2e" }],
    ["PlanService/GetPlan", { name: "projects/example/plans/%2Fother" }],
    ["ProjectService/DoesNotExist", { name: "projects/example" }],
    [
      "SettingService/GetSetting",
      { name: "settings/example", project: "projects/example" },
    ],
    ["ProjectService/ListProjects", { project: "projects/example" }],
    [
      "PlanService/GetPlan",
      { name: "projects/example/plans/12", targets: ["projects/example"] },
    ],
  ])("denies an unprovable %s request %j", async (operation, body) => {
    await expect(
      isDiagnosticApiCallInScope(prepared(operation, body), capability)
    ).resolves.toBe(false);
  });

  it.each([
    {
      kind: "diagnostic",
      resource: { ...capability.resource, project: "projects/other" },
    },
    {
      kind: "diagnostic",
      resource: { ...capability.resource, name: "projects/-/plans/12" },
    },
    {
      kind: "diagnostic",
      resource: {
        ...capability.resource,
        kind: "project",
        name: "projects/example",
      },
    },
  ])("denies an invalid capability %j", async (invalid) => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("PlanService/GetPlan", { name: "projects/example/plans/12" }),
        invalid as AgentDiagnosticCapability
      )
    ).resolves.toBe(false);
  });

  it.each([
    { path: "/bytebase.v1.PlanService/DeletePlan" },
    { operationId: "PlanService/GetPlan" },
    { body: "invalid JSON" },
    { body: "null" },
    { body: "[]" },
    { body: '{"name":42}' },
  ])("denies a malformed prepared call %j", async (override) => {
    await expect(
      isDiagnosticApiCallInScope(
        {
          ...prepared("PlanService/GetPlan", {
            name: "projects/example/plans/12",
          }),
          ...override,
        },
        capability
      )
    ).resolves.toBe(false);
  });

  it.each([
    "instances/shared/databases/app",
    "projects/example/instances/primary/databases/app",
  ])(
    "positively resolves a database target for GetDatabase: %s",
    async (name) => {
      api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/GetDatabase", { name }),
          capability
        )
      ).resolves.toBe(true);
      expect(api.getDatabase).toHaveBeenCalledWith(
        expect.objectContaining({ name }),
        expect.anything()
      );
    }
  );

  it.each([
    { name: "instances/shared/databases/app", project: "projects/other" },
    { name: "instances/shared/databases/other", project: "projects/example" },
    { name: "instances/shared/databases/app", project: "" },
    undefined,
  ])("denies unresolved or mismatched database proof %j", async (database) => {
    if (database) api.getDatabase.mockResolvedValue(database);
    else api.getDatabase.mockRejectedValue(new Error("Unavailable"));
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/GetDatabase", {
          name: "instances/shared/databases/app",
        }),
        capability
      )
    ).resolves.toBe(false);
  });

  it("rechecks a database's project instead of reusing earlier proof", async () => {
    const name = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
    const call = prepared("DatabaseService/GetDatabase", { name });
    await expect(isDiagnosticApiCallInScope(call, capability)).resolves.toBe(
      true
    );
    api.getDatabase.mockResolvedValue({ name, project: "projects/other" });
    await expect(isDiagnosticApiCallInScope(call, capability)).resolves.toBe(
      false
    );
  });

  it("resolves database descendants using their canonical database prefix", async () => {
    const name = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/GetDatabaseMetadata", {
          name: `${name}/metadata`,
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each([
    "Shop Orders",
    "percent%",
    "literal%2Fname",
    "question?mark",
    "hash#tag",
    "back\\slash",
  ])(
    "preserves the exact opaque database ID %j for metadata reads",
    async (id) => {
      for (const prefix of [
        "instances/shared",
        "projects/example/instances/primary",
      ]) {
        const name = `${prefix}/databases/${id}`;
        api.getDatabase.mockResolvedValue({
          name,
          project: "projects/example",
        });
        await expect(
          isDiagnosticApiCallInScope(
            prepared("DatabaseService/GetDatabaseMetadata", {
              name: `${name}/metadata`,
            }),
            capability
          )
        ).resolves.toBe(true);
        expect(api.getDatabase).toHaveBeenCalledWith(
          expect.objectContaining({ name }),
          expect.anything()
        );
      }
    }
  );

  it.each(["projects/other", ""])(
    "rejects an opaque database ID without matching project proof %j",
    async (project) => {
      const name = "instances/shared/databases/Shop Orders";
      api.getDatabase.mockResolvedValue({ name, project });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/GetDatabaseMetadata", {
            name: `${name}/metadata`,
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it("does not accept decoded database proof for a literal percent-encoded ID", async () => {
    api.getDatabase.mockResolvedValue({
      name: "instances/shared/databases/Shop Orders",
      project: "projects/example",
    });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/GetDatabaseMetadata", {
          name: "instances/shared/databases/Shop%20Orders/metadata",
        }),
        capability
      )
    ).resolves.toBe(false);
  });

  it.each(["instances/shared", "projects/example/instances/primary"])(
    "admits the SDL schema singleton under %s after exact database proof",
    async (prefix) => {
      const name = `${prefix}/databases/Shop Orders`;
      api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/GetDatabaseSDLSchema", {
            name: `${name}/sdlSchema`,
          }),
          capability
        )
      ).resolves.toBe(true);
    }
  );

  it.each(["app", "app/schema", "app/sdlSchema/extra"])(
    "rejects a malformed SDL schema singleton %s",
    async (suffix) => {
      const name = "instances/shared/databases/app";
      api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/GetDatabaseSDLSchema", {
            name: `instances/shared/databases/${suffix}`,
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it.each([
    {
      parent: "projects/example",
      names: [
        "instances/shared/databases/app",
        "projects/example/instances/primary/databases/app",
      ],
    },
    {
      parent: "instances/shared",
      names: [
        "instances/shared/databases/app",
        "instances/shared/databases/Shop Orders",
      ],
    },
    {
      parent: "projects/example/instances/primary",
      names: ["projects/example/instances/primary/databases/app"],
    },
  ])("admits BatchGetDatabases with supported parent $parent", async (body) => {
    api.getDatabase.mockImplementation(async ({ name }: { name: string }) => ({
      name,
      project: "projects/example",
    }));
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/BatchGetDatabases", body),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each([
    { parent: "projects/other", names: ["instances/shared/databases/app"] },
    {
      parent: "projects/example/plans/12",
      names: ["instances/shared/databases/app"],
    },
    { parent: "instances/-", names: ["instances/shared/databases/app"] },
    { parent: "projects/example", names: [] },
    {
      parent: "projects/example",
      names: [
        "instances/shared/databases/app",
        "instances/shared/databases/foreign",
      ],
    },
  ])("rejects unprovable BatchGetDatabases targets %j", async (body) => {
    api.getDatabase.mockImplementation(async ({ name }: { name: string }) => ({
      name,
      project: name.endsWith("/foreign")
        ? "projects/other"
        : "projects/example",
    }));
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/BatchGetDatabases", body),
        capability
      )
    ).resolves.toBe(false);
  });

  it.each([
    ["SQLService/Query", { statement: "SELECT * FROM foreign_db.records" }],
    ["SQLService/Query", { statement: "SELECT 1" }],
    [
      "SQLService/Query",
      { statement: "SELECT * FROM foreign_db.records", explain: true },
    ],
    [
      "SQLService/Query",
      { statement: "USE foreign_db; SELECT * FROM records" },
    ],
    [
      "SQLService/Export",
      { statement: "SELECT * FROM foreign_db.records", admin: false },
    ],
    [
      "SQLService/Export",
      { statement: "SELECT * FROM foreign_db.records", admin: true },
    ],
    [
      "SQLService/AdminExecute",
      { statement: "UPDATE foreign_db.records SET value = 1" },
    ],
  ])(
    "rejects %s when effective SQL targets cannot be proven: %j",
    async (operation, body) => {
      const name = "instances/shared/databases/app";
      api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
      await expect(
        isDiagnosticApiCallInScope(
          prepared(operation, { name, ...body }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it.each(["projects/example", "projects/example/instances/primary"])(
    "admits ListDatabases with project parent %s",
    async (parent) => {
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/ListDatabases", { parent }),
          capability
        )
      ).resolves.toBe(true);
    }
  );

  it.each(["workspaces/example", "instances/shared"])(
    "requires a proven project filter for ListDatabases under %s",
    async (parent) => {
      const filter = 'project == "projects/example"';
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/ListDatabases", { parent }),
          capability
        )
      ).resolves.toBe(false);
      api.batchParse.mockResolvedValue({
        expressions: [projectFilter("projects/example")],
      });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/ListDatabases", { parent, filter }),
          capability
        )
      ).resolves.toBe(true);
    }
  );

  it("admits SearchDatabases only with a proven project filter", async () => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/SearchDatabases", {}),
        capability
      )
    ).resolves.toBe(false);
    api.batchParse.mockResolvedValue({
      expressions: [projectFilter("projects/example")],
    });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/SearchDatabases", {
          filter: 'project == "projects/example"',
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each(["DatabaseService/ListDatabases", "DatabaseService/SearchDatabases"])(
    "denies an escaping or unverified project filter for %s",
    async (operation) => {
      const call = prepared(operation, {
        ...(operation.endsWith("ListDatabases")
          ? { parent: "instances/shared" }
          : {}),
        filter: 'project == "projects/example" || project == "projects/other"',
      });
      api.batchParse.mockResolvedValue({
        expressions: [
          create(ExprSchema, {
            exprKind: {
              case: "callExpr",
              value: {
                function: "_||_",
                args: [
                  projectFilter("projects/example"),
                  projectFilter("projects/other"),
                ],
              },
            },
          }),
        ],
      });
      await expect(isDiagnosticApiCallInScope(call, capability)).resolves.toBe(
        false
      );
      api.batchParse.mockRejectedValue(new Error("Unavailable"));
      await expect(isDiagnosticApiCallInScope(call, capability)).resolves.toBe(
        false
      );
    }
  );

  it.each([
    ["ProjectService/BatchDeleteProjects", { names: ["projects/example"] }],
    [
      "RolloutService/BatchRunTasks",
      {
        parent: "projects/example/plans/12/rollout/stages/prod",
        tasks: ["projects/example/plans/12/rollout/stages/prod/tasks/4"],
      },
    ],
    [
      "IssueService/BatchUpdateIssuesStatus",
      {
        parent: "projects/ignored",
        issues: ["projects/example/issues/4"],
        status: "OPEN",
      },
    ],
    [
      "InstanceService/BatchSyncInstances",
      {
        parent: "projects/example",
        requests: [{ name: "projects/example/instances/primary" }],
      },
    ],
  ])("admits every effective target of %s", async (operation, body) => {
    await expect(
      isDiagnosticApiCallInScope(prepared(operation, body), capability)
    ).resolves.toBe(true);
  });

  it.each([
    ["ProjectService/BatchDeleteProjects", { names: [] }],
    [
      "ProjectService/BatchDeleteProjects",
      { names: ["projects/example", "projects/other"] },
    ],
    [
      "RolloutService/BatchRunTasks",
      {
        parent: "projects/example/plans/12/rollout/stages/prod",
        tasks: [],
      },
    ],
    [
      "RolloutService/BatchRunTasks",
      {
        parent: "projects/example/plans/12/rollout/stages/prod",
        tasks: ["projects/example"],
      },
    ],
    [
      "RolloutService/BatchRunTasks",
      {
        parent: "projects/example/plans/12/rollout/stages/prod",
        tasks: ["projects/other/plans/12/rollout/stages/prod/tasks/4"],
      },
    ],
    [
      "IssueService/BatchUpdateIssuesStatus",
      { parent: "projects/example", issues: [] },
    ],
    [
      "IssueService/BatchUpdateIssuesStatus",
      { parent: "projects/example", issues: ["projects/example/plans/12"] },
    ],
    [
      "IssueService/BatchUpdateIssuesStatus",
      {
        parent: "projects/example",
        issues: ["projects/example/issues/4", "projects/other/issues/5"],
      },
    ],
    [
      "InstanceService/BatchSyncInstances",
      { parent: "projects/example", requests: [] },
    ],
  ])(
    "denies empty or mixed effective batch targets for %s",
    async (operation, body) => {
      await expect(
        isDiagnosticApiCallInScope(prepared(operation, body), capability)
      ).resolves.toBe(false);
    }
  );

  it("denies moving a verified database into another project", async () => {
    const name = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/UpdateDatabase", {
          database: { name, project: "projects/other" },
          updateMask: "project",
        }),
        capability
      )
    ).resolves.toBe(false);
  });

  it.each([undefined, ""])(
    "denies a database project update without a concrete project: %j",
    async (project) => {
      const name = "instances/shared/databases/app";
      api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("DatabaseService/UpdateDatabase", {
            database: { name, ...(project === undefined ? {} : { project }) },
            updateMask: "project",
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it("admits a verified database update that leaves its project untouched", async () => {
    const name = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("DatabaseService/UpdateDatabase", {
          database: { name, labels: { team: "example" } },
          updateMask: "labels",
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it("admits a Plan only after resolving every nested database target", async () => {
    const name = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("PlanService/CreatePlan", {
          parent: "projects/example",
          plan: {
            specs: [
              {
                changeDatabaseConfig: {
                  targets: [name],
                  sheet: "projects/example/sheets/12",
                },
              },
            ],
          },
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each(["projects/other/sheets/12", "instances/shared/databases/foreign"])(
    "denies a foreign nested Plan target %s",
    async (target) => {
      api.getDatabase.mockResolvedValue({
        name: target,
        project: "projects/other",
      });
      await expect(
        isDiagnosticApiCallInScope(
          prepared("PlanService/CreatePlan", {
            parent: "projects/example",
            plan: {
              specs: [
                {
                  changeDatabaseConfig: {
                    targets: [target],
                    sheet: "projects/example/sheets/12",
                  },
                },
              ],
            },
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it.each([{ targets: undefined }, { targets: [] }, { targets: [""] }])(
    "denies an absent or empty nested Plan database target %j",
    async ({ targets }) => {
      await expect(
        isDiagnosticApiCallInScope(
          prepared("PlanService/CreatePlan", {
            parent: "projects/example",
            plan: {
              specs: [
                {
                  changeDatabaseConfig: {
                    ...(targets ? { targets } : {}),
                    sheet: "projects/example/sheets/12",
                  },
                },
              ],
            },
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it.each(
    [
      undefined,
      [],
      ["projects/other/databaseGroups/example"],
      ["instances/shared"],
      ["instances/shared/databases/-"],
    ].map((targets) => ({ targets }))
  )(
    "denies unscoped top-level release execution targets %j",
    async ({ targets }) => {
      await expect(
        isDiagnosticApiCallInScope(
          prepared("ReleaseService/CheckRelease", {
            parent: "projects/example",
            release: {},
            ...(targets ? { targets } : {}),
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it.each(["projects/other", undefined])(
    "denies a top-level release database target with project proof %j",
    async (project) => {
      const name = "instances/shared/databases/app";
      if (project) api.getDatabase.mockResolvedValue({ name, project });
      else api.getDatabase.mockRejectedValue(new Error("Unavailable"));
      await expect(
        isDiagnosticApiCallInScope(
          prepared("ReleaseService/CheckRelease", {
            parent: "projects/example",
            release: {},
            targets: [name],
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it("positively resolves a top-level release database target", async () => {
    const name = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({ name, project: "projects/example" });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("ReleaseService/CheckRelease", {
          parent: "projects/example",
          release: {},
          targets: [name],
        }),
        capability
      )
    ).resolves.toBe(true);
    expect(api.getDatabase).toHaveBeenCalledWith(
      expect.objectContaining({ name }),
      expect.anything()
    );
  });

  it("admits a top-level release database-group target in the bound project", async () => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("ReleaseService/CheckRelease", {
          parent: "projects/example",
          release: {},
          targets: ["projects/example/databaseGroups/example"],
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it.each([undefined, "", "environments/prod"])(
    "preserves the project rollout environment selector %j",
    async (target) => {
      await expect(
        isDiagnosticApiCallInScope(
          prepared("RolloutService/CreateRollout", {
            parent: "projects/example/plans/12",
            ...(target === undefined ? {} : { target }),
          }),
          capability
        )
      ).resolves.toBe(true);
    }
  );

  it("does not treat another project as a rollout environment selector", async () => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("RolloutService/CreateRollout", {
          parent: "projects/example/plans/12",
          target: "projects/other",
        }),
        capability
      )
    ).resolves.toBe(false);
  });

  it.each(["app", "instances/shared/databases/missing"])(
    "denies an unprovable resource-owned database reference %s",
    async (database) => {
      api.getDatabase.mockRejectedValue(new Error("Unavailable"));
      await expect(
        isDiagnosticApiCallInScope(
          prepared("SavedQueryService/CreateSavedQuery", {
            parent: "projects/example",
            savedQuery: { title: "Example", database },
          }),
          capability
        )
      ).resolves.toBe(false);
    }
  );

  it("admits a resource-owned database reference after positive scope proof", async () => {
    const database = "instances/shared/databases/app";
    api.getDatabase.mockResolvedValue({
      name: database,
      project: "projects/example",
    });
    await expect(
      isDiagnosticApiCallInScope(
        prepared("SavedQueryService/CreateSavedQuery", {
          parent: "projects/example",
          savedQuery: { title: "Example", database },
        }),
        capability
      )
    ).resolves.toBe(true);
  });

  it("does not confuse a database creation name with a database resource reference", async () => {
    await expect(
      isDiagnosticApiCallInScope(
        prepared("PlanService/CreatePlan", {
          parent: "projects/example",
          plan: {
            specs: [
              {
                createDatabaseConfig: {
                  target: "projects/example/instances/primary",
                  database: "app",
                },
              },
            ],
          },
        }),
        capability
      )
    ).resolves.toBe(true);
  });
});
