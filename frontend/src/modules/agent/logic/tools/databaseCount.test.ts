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

vi.mock("@/utils/pagination", () => ({
  getDefaultPagination: () => 17,
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
  test("returns one workspace page without returning database records", async () => {
    databaseApi.searchDatabases.mockResolvedValueOnce({
      databases: databases(17),
      nextPageToken: "2",
    });

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "available",
      count: 17,
      partial: true,
    });
    expect(result).not.toContain("private-");
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(1);
    expect(databaseApi.searchDatabases.mock.calls[0][0]).toMatchObject({
      pageSize: 17,
      pageToken: "",
    });
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
      partial: false,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({ filter: 'project == "projects/sample"' }),
      expect.anything()
    );
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("returns one page for the selected workspace-instance database", async () => {
    databaseApi.searchDatabases.mockResolvedValueOnce({
      databases: databases(17),
      nextPageToken: "2",
    });

    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase: "instances/shared/databases/private-db",
    })({ scope: "referenced-instance" });

    expect(JSON.parse(result)).toEqual({
      scope: "instances/shared",
      status: "available",
      count: 17,
      partial: true,
    });
    expect(result).not.toContain("private-db");
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(1);
    expect(databaseApi.searchDatabases.mock.calls[0][0]).toMatchObject({
      filter: 'instance == "instances/shared"',
      pageSize: 17,
      pageToken: "",
    });
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("preserves the full parent of a selected project-instance database", async () => {
    databaseApi.searchDatabases.mockResolvedValue({
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
      partial: false,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        filter: 'instance == "projects/selected/instances/dedicated"',
      }),
      expect.anything()
    );
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
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
    expect(databaseApi.searchDatabases).not.toHaveBeenCalled();
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("rejects an arbitrary parent passed as the requested count scope", async () => {
    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase: "instances/shared/databases/private-db",
    })({ scope: "instances/another" });

    expect(JSON.parse(result)).toHaveProperty("error");
    expect(databaseApi.searchDatabases).not.toHaveBeenCalled();
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });

  test("reports unavailable when the instance request fails", async () => {
    databaseApi.searchDatabases.mockRejectedValueOnce(
      new Error("permission denied")
    );

    const result = await createDatabaseCountTool({
      scopeName: "projects/selected",
      referencedProject: "projects/selected",
      referencedDatabase: "instances/shared/databases/private-db",
    })({ scope: "referenced-instance" });

    expect(JSON.parse(result)).toEqual({
      scope: "instances/shared",
      status: "unavailable",
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(1);
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
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
      partial: false,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({ filter: 'project == "projects/empty"' }),
      expect.anything()
    );
  });

  test("reports unavailable when the first workspace request fails", async () => {
    databaseApi.searchDatabases.mockRejectedValueOnce(
      new Error("permission denied")
    );

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "unavailable",
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(1);
  });

  test("marks an underfilled page as partial when the API returns a page token", async () => {
    databaseApi.searchDatabases.mockResolvedValueOnce({
      databases: databases(4),
      nextPageToken: "2",
    });

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "available",
      count: 4,
      partial: true,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(1);
  });

  test("does not infer another page solely from a full page", async () => {
    databaseApi.searchDatabases.mockResolvedValueOnce({
      databases: databases(17),
      nextPageToken: "",
    });

    const result = await createDatabaseCountTool({
      scopeName: "workspaces/demo",
    })();

    expect(JSON.parse(result)).toEqual({
      scope: "workspaces/demo",
      status: "available",
      count: 17,
      partial: false,
    });
    expect(databaseApi.searchDatabases).toHaveBeenCalledTimes(1);
  });

  test("rejects an invalid or missing chat scope", async () => {
    const result = await createDatabaseCountTool({
      scopeName: "instances/test",
    })();

    expect(JSON.parse(result)).toHaveProperty("error");
    expect(databaseApi.listDatabases).not.toHaveBeenCalled();
  });
});
