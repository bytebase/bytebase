// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { databaseServiceClientConnect } from "@/api";
import { State } from "@/types/proto-es/v1/common_pb";
import { validateResourceChoices } from "./resourceChoices";

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: vi.fn() },
}));

beforeEach(() => {
  vi.mocked(databaseServiceClientConnect.getDatabase).mockReset();
});

describe("validateResourceChoices", () => {
  test("uses fresh active database names and titles in the selected project", async () => {
    vi.mocked(databaseServiceClientConnect.getDatabase).mockImplementation(
      async (request) =>
        ({
          name: request.name,
          project: request.name!.includes("wrong")
            ? "projects/other"
            : "projects/sample",
          state: request.name!.includes("deleted")
            ? State.DELETED
            : State.ACTIVE,
          instanceResource: { title: "Production" },
        }) as never
    );
    const options = await validateResourceChoices(
      "database",
      [
        "instances/prod/databases/orders",
        "instances/prod/databases/orders",
        "instances/prod/databases/wrong",
        "instances/prod/databases/deleted",
      ],
      "projects/sample",
      "workspaces/example"
    );
    expect(options).toEqual([
      {
        label: "orders (Production)",
        value: "instances/prod/databases/orders",
      },
    ]);
    expect(databaseServiceClientConnect.getDatabase).toHaveBeenCalledTimes(3);
  });

  test("does not verify database choices without project scope or with too many values", async () => {
    expect(
      await validateResourceChoices(
        "database",
        ["instances/a/databases/b"],
        undefined,
        "workspaces/example"
      )
    ).toBeUndefined();
    expect(
      await validateResourceChoices(
        "database",
        Array.from({ length: 6 }, (_, i) => `instances/a/databases/db${i}`),
        "projects/sample",
        "workspaces/example"
      )
    ).toBeUndefined();
    expect(databaseServiceClientConnect.getDatabase).not.toHaveBeenCalled();
  });

  test("does not present denied reads as an empty database list", async () => {
    vi.mocked(databaseServiceClientConnect.getDatabase).mockRejectedValue(
      new Error("permission denied")
    );
    expect(
      await validateResourceChoices(
        "database",
        ["instances/a/databases/b"],
        "projects/sample",
        "workspaces/example"
      )
    ).toBeUndefined();
  });
});
