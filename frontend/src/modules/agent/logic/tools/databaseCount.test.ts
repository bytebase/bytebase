// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";

const databaseApi = vi.hoisted(() => ({
  listDatabases: vi.fn(),
  searchDatabases: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: {
    listDatabases: databaseApi.listDatabases,
    searchDatabases: databaseApi.searchDatabases,
  },
}));

import { createDatabaseCountTool } from "./databaseCount";

const databases = (count: number) =>
  Array.from({ length: count }, (_, index) => ({
    name: `instances/test/databases/private-${index}`,
  }));

beforeEach(() => {
  databaseApi.listDatabases.mockReset();
  databaseApi.searchDatabases.mockReset();
});

describe("get_database_count", () => {
  test("counts all workspace pages without returning database records", async () => {
    databaseApi.searchDatabases
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "2" })
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "3" })
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "4" })
      .mockResolvedValueOnce({ databases: databases(200), nextPageToken: "" });

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "available",
      count: 3200,
    });
    expect(result).not.toContain("private-");
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(4);
    expect(
      databaseApi.searchDatabases.mock.calls.map(
        ([request]) => request.pageToken
      )
    ).toEqual(["", "2", "3", "4"]);
    for (const [request] of databaseApi.searchDatabases.mock.calls) {
      expect(request).toMatchObject({
        pageSize: 1000,
      });
    }
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("counts the bound project instead of the workspace", async () => {
    databaseApi.searchDatabases.mockResolvedValue({
      databases: databases(2),
      nextPageToken: "",
    });

    const result = await createDatabaseCountTool({
      scopeName: "projects/sample",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "projects/sample",
      status: "available",
      count: 2,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({ filter: 'project == "projects/sample"' }),
      expect.anything()
    );
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("counts all projects on the selected workspace-instance database's instance", async () => {
    databaseApi.listDatabases
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "2" })
      .mockResolvedValueOnce({ databases: databases(25), nextPageToken: "" });

    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase: "instances/shared/databases/private-db",
    })({ scope: "referenced-instance" });

    expect(JSON.parse(result)).toEqual({
      scope: "instances/shared",
      status: "available",
      count: 1025,
    });
    expect(result).not.toContain("private-db");
    expect(
      databaseApi.listDatabases.mock.calls.map(([request]) => request.parent)
    ).toEqual(["instances/shared", "instances/shared"]);
  });

  test("preserves the full parent of a selected project-instance database", async () => {
    databaseApi.listDatabases.mockResolvedValue({
      databases: databases(3),
      nextPageToken: "",
    });

    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase:
        "projects/selected/instances/dedicated/databases/private-db",
    })({ scope: "referenced-instance" });

    expect(JSON.parse(result)).toEqual({
      scope: "projects/selected/instances/dedicated",
      status: "available",
      count: 3,
    });
    expect(databaseApi.listDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/selected/instances/dedicated",
      }),
      expect.anything()
    );
  });

  test("rejects an instance count without the current matching database reference", async () => {
    for (const reference of [
      {},
      {
        referencedProject: "projects/selected",
        referencedDatabase:
          "projects/other/instances/dedicated/databases/private-db",
      },
    ]) {
      const result = await createDatabaseCountTool({
        scopeName: "projects/selected",
        ...reference,
      })({ scope: "referenced-instance" });
      expect(JSON.parse(result)).toHaveProperty("error");
    }
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("rejects an arbitrary parent passed as the requested count scope", async () => {
    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase: "instances/shared/databases/private-db",
    })({ scope: "instances/another" });

    expect(JSON.parse(result)).toHaveProperty("error");
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("does not report a partial instance count after permission failure", async () => {
    databaseApi.listDatabases
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "2" })
      .mockRejectedValueOnce(new Error("permission denied"));

    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase: "instances/shared/databases/private-db",
    })({ scope: "referenced-instance" });

    expect(JSON.parse(result)).toEqual({
      scope: "instances/shared",
      status: "unavailable",
    });
    expect(result).not.toContain("1000");
  });

  test("reports zero only after a complete empty page", async () => {
    databaseApi.searchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });

    const result = await createDatabaseCountTool({
      scopeName: "projects/empty",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "projects/empty",
      status: "available",
      count: 0,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({ filter: 'project == "projects/empty"' }),
      expect.anything()
    );
  });

  test("does not claim an exact count if a later page fails", async () => {
    databaseApi.searchDatabases
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "2" })
      .mockRejectedValueOnce(new Error("permission denied"));

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "unavailable",
    });
    expect(result).not.toContain("1000");
  });

  test("does not claim an exact count if pagination repeats a token", async () => {
    databaseApi.searchDatabases
      .mockResolvedValueOnce({ databases: databases(1000), nextPageToken: "2" })
      .mockResolvedValueOnce({
        databases: databases(1000),
        nextPageToken: "2",
      });

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "unavailable",
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(2);
  });

  test("rejects an invalid or missing chat scope", async () => {
    const result = await createDatabaseCountTool({
      scopeName: "instances/test",
    })();

    expect(JSON.parse(result)).toHaveProperty("error");
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });
});
