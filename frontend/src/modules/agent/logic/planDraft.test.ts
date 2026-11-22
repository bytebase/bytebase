import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/app/router/handles";
import { consumeInitialSQL } from "@/lib/plan/initialSQLStorage";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { prepareAgentPlanDraftRoute } from "./planDraft";

const api = vi.hoisted(() => ({ getDatabase: vi.fn(), getProject: vi.fn() }));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: api.getDatabase },
  projectServiceClientConnect: { getProject: api.getProject },
}));

const action = {
  kind: "draft-plan" as const,
  project: "projects/sample",
  database: "instances/prod/databases/app",
};
const answer = "Review this DDL:\n\n```sql\nCREATE TABLE notes (id INT);\n```";

const requireQuery = (
  route: Awaited<ReturnType<typeof prepareAgentPlanDraftRoute>>
) => {
  if (typeof route === "string" || !route.query) {
    throw new Error("Expected a Plan creation route with a query");
  }
  return route.query;
};

const storage = (): Storage => {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    key: (index) => [...values.keys()][index] ?? null,
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => {
      values.set(key, value);
    },
    removeItem: (key) => {
      values.delete(key);
    },
    clear: () => values.clear(),
  };
};

beforeEach(() => {
  vi.stubGlobal("localStorage", storage());
  api.getDatabase.mockReset();
  api.getProject.mockReset();
  api.getDatabase.mockResolvedValue(
    create(DatabaseSchema$, {
      name: action.database,
      project: action.project,
      state: State.ACTIVE,
    })
  );
  api.getProject.mockResolvedValue(
    create(ProjectSchema, { name: action.project, state: State.ACTIVE })
  );
});

describe("prepareAgentPlanDraftRoute", () => {
  test("opens a local Plan draft with the exact displayed SQL and database", async () => {
    const route = await prepareAgentPlanDraftRoute(
      action,
      answer,
      "Proposed database change"
    );

    expect(route).toMatchObject({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "sample", planId: "create" },
      query: {
        databaseList: action.database,
        name: "[app] Proposed database change",
      },
    });
    const query = requireQuery(route);
    expect(query).not.toHaveProperty("sql");
    expect(consumeInitialSQL<string>(String(query.sqlStorageKey))).toBe(
      "CREATE TABLE notes (id INT);"
    );
    expect(api.getDatabase).toHaveBeenCalledOnce();
    expect(api.getProject).toHaveBeenCalledOnce();
  });

  test("leaves the title blank when the project requires a manual title", async () => {
    api.getProject.mockResolvedValue(
      create(ProjectSchema, {
        name: action.project,
        state: State.ACTIVE,
        enforceIssueTitle: true,
      })
    );
    const route = await prepareAgentPlanDraftRoute(
      action,
      answer,
      "Proposed database change"
    );
    expect(requireQuery(route)).not.toHaveProperty("name");
  });

  test("does not prepare a draft if the database moved to another project", async () => {
    api.getDatabase.mockResolvedValue(
      create(DatabaseSchema$, {
        name: action.database,
        project: "projects/other",
        state: State.ACTIVE,
      })
    );
    await expect(
      prepareAgentPlanDraftRoute(action, answer, "Proposed database change")
    ).rejects.toThrow();
    expect(api.getProject).not.toHaveBeenCalled();
    expect(localStorage.length).toBe(0);
  });

  test("does not prepare a draft without exactly one visible SQL block", async () => {
    await expect(
      prepareAgentPlanDraftRoute(
        action,
        "The table needs an id column.",
        "Proposed database change"
      )
    ).rejects.toThrow();
    expect(api.getDatabase).not.toHaveBeenCalled();
  });

  test("reports browser storage failure without opening an empty draft", async () => {
    vi.stubGlobal("localStorage", {
      ...storage(),
      setItem: () => {
        throw new Error("quota exceeded");
      },
    });
    await expect(
      prepareAgentPlanDraftRoute(action, answer, "Proposed database change")
    ).rejects.toMatchObject({ name: "PlanDraftStorageError" });
  });
});
