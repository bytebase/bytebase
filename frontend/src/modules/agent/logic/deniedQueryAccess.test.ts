import { create } from "@bufbuild/protobuf";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SQL_EDITOR_DATABASE_MODULE } from "@/app/router/handles";
import {
  consumeAccessGrantDraft,
  type RequestRoleDraft,
} from "@/modules/sql-editor/model/accessGrantDraft";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  extractDeniedQueryAccessRequest,
  prepareAgentAccessGrantDraftRoute,
  prepareAgentDeniedQueryRequestRoute,
  resolveDeniedQueryAccessDraft,
} from "./deniedQueryAccess";

const api = vi.hoisted(() => ({ getDatabase: vi.fn(), getProject: vi.fn() }));
const app = vi.hoisted(() => ({
  loadSubscription: vi.fn(),
  hasFeature: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: api.getDatabase },
  projectServiceClientConnect: { getProject: api.getProject },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => app },
}));

const database = "instances/prod/databases/app";
const project = "projects/sample";
const query = {
  operationId: "SQLService/Query",
  body: { name: database, statement: "SELECT id FROM users", schema: "public" },
};
const denied = (
  resources: string[] = [database],
  permissions = ["bb.sql.select"]
) =>
  JSON.stringify({
    status: 200,
    response: {
      results: [
        {
          error: "permission denied",
          permissionDenied: { requiredPermissions: permissions, resources },
        },
      ],
    },
  });

beforeEach(() => {
  vi.clearAllMocks();
  sessionStorage.clear();
  api.getDatabase.mockImplementation(async ({ name }: { name: string }) =>
    create(DatabaseSchema$, { name, project, state: State.ACTIVE })
  );
  api.getProject.mockResolvedValue(
    create(ProjectSchema, {
      name: project,
      state: State.ACTIVE,
      allowJustInTimeAccess: true,
    })
  );
  app.loadSubscription.mockResolvedValue({});
  app.hasFeature.mockReturnValue(true);
});

afterEach(() => vi.unstubAllGlobals());

describe("prepareAgentAccessGrantDraftRoute", () => {
  const draft = {
    project,
    sourceDatabase: database,
    targets: [database],
    statement: "SELECT id FROM users",
    schema: "public",
  };

  test("opens SQL Editor on the source database with a one-time draft key", async () => {
    const route = await prepareAgentAccessGrantDraftRoute(draft);
    expect(route).toMatchObject({
      name: SQL_EDITOR_DATABASE_MODULE,
      params: { project: "sample", instance: "prod", database: "app" },
      query: { parent: "instances/prod" },
    });
    if (typeof route === "string") throw new Error("Expected a route target");
    expect(route.query).not.toHaveProperty("statement");
    const key = route.query?.accessGrantDraftKey;
    expect(typeof key).toBe("string");
    expect(consumeAccessGrantDraft(String(key))).toEqual(draft);
  });

  test("does not store an obsolete draft after a database moves projects", async () => {
    api.getDatabase.mockResolvedValue(
      create(DatabaseSchema$, {
        name: database,
        project: "projects/other",
        state: State.ACTIVE,
      })
    );
    await expect(prepareAgentAccessGrantDraftRoute(draft)).rejects.toThrow();
    expect(sessionStorage.length).toBe(0);
  });

  test("reports storage failure without opening an empty drawer", async () => {
    const storage = sessionStorage;
    vi.stubGlobal("sessionStorage", {
      get length() {
        return storage.length;
      },
      key: (index: number) => storage.key(index),
      getItem: (key: string) => storage.getItem(key),
      removeItem: (key: string) => storage.removeItem(key),
      clear: () => storage.clear(),
      setItem: () => {
        throw new Error("storage disabled");
      },
    });
    await expect(
      prepareAgentAccessGrantDraftRoute(draft)
    ).rejects.toMatchObject({
      name: "DeniedQueryRequestDraftStorageError",
    });
  });

  test("opens a Request Role draft when JIT is off", async () => {
    api.getProject.mockResolvedValue(
      create(ProjectSchema, {
        name: project,
        state: State.ACTIVE,
        allowRequestRole: true,
      })
    );
    const draft: RequestRoleDraft = {
      kind: "requestRole",
      project,
      sourceDatabase: database,
      targets: [database],
      resources: [`${database}/schemas/public/tables/users`],
      statement: "SELECT id FROM users",
      requiredPermissions: ["bb.sql.select"],
    };
    const route = await prepareAgentDeniedQueryRequestRoute(draft);
    if (typeof route === "string") throw new Error("Expected a route target");
    const key = route.query?.accessGrantDraftKey;
    expect(typeof key).toBe("string");
    expect(consumeAccessGrantDraft(String(key))).toEqual(draft);
    expect(JSON.stringify(route)).not.toContain(draft.statement);
  });

  test("preserves a project instance source when route IDs can collide", async () => {
    const sourceDatabase = "projects/sample/instances/prod/databases/app";
    const draft = {
      project,
      sourceDatabase,
      targets: [sourceDatabase],
      statement: "SELECT id FROM users",
    };

    const route = await prepareAgentDeniedQueryRequestRoute(draft);

    expect(route).toMatchObject({
      name: SQL_EDITOR_DATABASE_MODULE,
      params: { project: "sample", instance: "prod", database: "app" },
      query: { parent: "projects/sample/instances/prod" },
    });
  });
});

describe("extractDeniedQueryAccessRequest", () => {
  test("uses the exact failed SQL and typed denied resources, not a prose error", () => {
    expect(
      extractDeniedQueryAccessRequest(
        query,
        denied([`${database}/schemas/public/tables/users`, database])
      )
    ).toEqual({
      sourceDatabase: database,
      targets: [database],
      resources: [`${database}/schemas/public/tables/users`, database],
      statement: "SELECT id FROM users",
      schema: "public",
      requiredPermissions: ["bb.sql.select"],
    });
  });

  test("does not offer a draft for a batch with earlier successful results", () => {
    const result = JSON.stringify({
      status: 200,
      response: {
        results: [
          { rows: [{ values: [1] }] },
          {
            error: "permission denied",
            permissionDenied: {
              requiredPermissions: ["bb.sql.select"],
              resources: [database],
            },
          },
        ],
      },
    });

    expect(extractDeniedQueryAccessRequest(query, result)).toBeUndefined();
  });

  test.each([
    [
      "client preflight",
      JSON.stringify({
        status: 403,
        error: "Not called",
        response: { code: "permission_denied" },
      }),
    ],
    [
      "generic 403",
      JSON.stringify({ status: 403, error: "permission denied" }),
    ],
    [
      "untyped query error",
      JSON.stringify({
        status: 200,
        response: { results: [{ error: "permission denied" }] },
      }),
    ],
    [
      "successful query",
      JSON.stringify({
        status: 200,
        response: { results: [{ columnNames: ["id"] }] },
      }),
    ],
    ["unsupported permission", denied([database], ["bb.sql.dml"])],
    ["non-database target", denied(["projects/sample"])],
  ])("rejects %s", (_name, result) => {
    expect(extractDeniedQueryAccessRequest(query, result)).toBeUndefined();
  });

  test("rejects denials from other operations and missing statements", () => {
    expect(
      extractDeniedQueryAccessRequest(
        { ...query, operationId: "SQLService/Export" },
        denied()
      )
    ).toBeUndefined();
    expect(
      extractDeniedQueryAccessRequest(
        { ...query, body: { name: database, statement: "  " } },
        denied()
      )
    ).toBeUndefined();
  });
});

describe("resolveDeniedQueryAccessDraft", () => {
  test("checks every target belongs to one active JIT-enabled project", async () => {
    const otherDatabase = "instances/prod/databases/other";
    await expect(
      resolveDeniedQueryAccessDraft(query, denied([database, otherDatabase]))
    ).resolves.toEqual({
      project,
      sourceDatabase: database,
      targets: [database, otherDatabase],
      statement: "SELECT id FROM users",
      schema: "public",
    });
    expect(api.getDatabase).toHaveBeenCalledTimes(2);
    expect(api.getProject).toHaveBeenCalledOnce();
  });

  test("does not offer a draft for a target in another project", async () => {
    api.getDatabase.mockImplementation(async ({ name }: { name: string }) =>
      create(DatabaseSchema$, {
        name,
        project: name === database ? project : "projects/other",
        state: State.ACTIVE,
      })
    );
    await expect(
      resolveDeniedQueryAccessDraft(
        query,
        denied([database, "instances/prod/databases/other"])
      )
    ).resolves.toBeUndefined();
  });

  test("does not offer an Access Grant draft when JIT is disabled", async () => {
    api.getProject.mockResolvedValue(
      create(ProjectSchema, { name: project, state: State.ACTIVE })
    );
    await expect(
      resolveDeniedQueryAccessDraft(query, denied())
    ).resolves.toBeUndefined();
  });

  test("offers a scoped Request Role draft when JIT is off and role requests are enabled", async () => {
    api.getProject.mockResolvedValue(
      create(ProjectSchema, {
        name: project,
        state: State.ACTIVE,
        allowRequestRole: true,
      })
    );
    await expect(
      resolveDeniedQueryAccessDraft(
        query,
        denied([`${database}/schemas/public/tables/users`])
      )
    ).resolves.toEqual({
      kind: "requestRole",
      project,
      sourceDatabase: database,
      targets: [database],
      resources: [`${database}/schemas/public/tables/users`],
      statement: "SELECT id FROM users",
      requiredPermissions: ["bb.sql.select"],
    });
  });

  test("does not offer Request Role when its feature is unavailable", async () => {
    api.getProject.mockResolvedValue(
      create(ProjectSchema, {
        name: project,
        state: State.ACTIVE,
        allowRequestRole: true,
      })
    );
    app.hasFeature.mockReturnValue(false);
    await expect(
      resolveDeniedQueryAccessDraft(query, denied())
    ).resolves.toBeUndefined();
  });
});
