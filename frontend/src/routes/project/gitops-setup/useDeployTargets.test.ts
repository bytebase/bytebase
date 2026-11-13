import { create } from "@bufbuild/protobuf";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import {
  type DeployPick,
  type DeployTargets,
  groupByEnvironment,
  useDeployTargets,
} from "./useDeployTargets";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  environmentList: [] as { name: string }[],
  getOrFetchDBGroupByName: vi.fn(),
  batchGetOrFetchDatabases: vi.fn(),
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    environmentList: mocks.environmentList,
    getOrFetchDBGroupByName: mocks.getOrFetchDBGroupByName,
    batchGetOrFetchDatabases: mocks.batchGetOrFetchDatabases,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

const db = (name: string, environment?: string): Database =>
  create(DatabaseSchema$, {
    name: `instances/pg/databases/${name}`,
    effectiveEnvironment: environment,
  });

const ORDER = [
  "environments/test",
  "environments/staging",
  "environments/prod",
];

describe("groupByEnvironment", () => {
  test("follows the workspace environment order", () => {
    const { stages, unassigned } = groupByEnvironment(
      [
        db("p1", "environments/prod"),
        db("t1", "environments/test"),
        db("p2", "environments/prod"),
      ],
      ORDER
    );
    expect(stages.map((s) => [s.id, s.databases.map((d) => d.name)])).toEqual([
      ["test", ["instances/pg/databases/t1"]],
      ["prod", ["instances/pg/databases/p1", "instances/pg/databases/p2"]],
    ]);
    expect(unassigned).toEqual([]);
  });

  test("leaves out databases with no environment", () => {
    const { stages, unassigned } = groupByEnvironment(
      [db("t1", "environments/test"), db("old"), db("older", "")],
      ORDER
    );
    expect(stages.map((s) => s.id)).toEqual(["test"]);
    expect(unassigned.map((d) => d.name)).toEqual([
      "instances/pg/databases/old",
      "instances/pg/databases/older",
    ]);
  });

  test("puts an environment missing from the order first, as the rollout does", () => {
    const { stages } = groupByEnvironment(
      [db("p1", "environments/prod"), db("x1", "environments/sandbox")],
      ORDER
    );
    expect(stages.map((s) => s.id)).toEqual(["sandbox", "prod"]);
  });
});

describe("useDeployTargets", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;
  let result: DeployTargets | undefined;

  const Probe = ({ pick }: { pick: DeployPick | undefined }) => {
    result = useDeployTargets(pick);
    return null;
  };

  const render = async (pick: DeployPick | undefined) => {
    await act(async () => {
      root.render(createElement(Probe, { pick }));
    });
  };

  beforeEach(() => {
    mocks.environmentList = ORDER.map((name) => ({ name }));
    mocks.getOrFetchDBGroupByName.mockReset();
    mocks.batchGetOrFetchDatabases.mockReset();
    container = document.createElement("div");
    root = createRoot(container);
    result = undefined;
  });

  afterEach(() => {
    act(() => root.unmount());
  });

  test("expands a group to the databases it matches", async () => {
    mocks.getOrFetchDBGroupByName.mockResolvedValue({
      matchedDatabases: [
        { name: "instances/pg/databases/t1" },
        { name: "instances/pg/databases/old" },
      ],
    });
    mocks.batchGetOrFetchDatabases.mockResolvedValue([
      db("t1", "environments/test"),
      db("old"),
    ]);
    await render({ kind: "group", name: "projects/shop/databaseGroups/all" });
    expect(mocks.getOrFetchDBGroupByName).toHaveBeenCalledWith(
      "projects/shop/databaseGroups/all",
      expect.objectContaining({ silent: true })
    );
    expect(mocks.batchGetOrFetchDatabases).toHaveBeenCalledWith(
      ["instances/pg/databases/t1", "instances/pg/databases/old"],
      true
    );
    expect(result?.loading).toBe(false);
    expect(result?.total).toBe(2);
    expect(result?.stages.map((s) => s.id)).toEqual(["test"]);
    expect(result?.unassigned).toHaveLength(1);
  });

  test("loads picked databases and nothing for an empty pick", async () => {
    mocks.batchGetOrFetchDatabases.mockResolvedValue([
      db("p1", "environments/prod"),
    ]);
    await render({ kind: "databases", names: ["instances/pg/databases/p1"] });
    expect(result?.stages.map((s) => s.id)).toEqual(["prod"]);

    await render({ kind: "databases", names: [] });
    expect(result).toMatchObject({ total: 0, loading: false, stages: [] });
    expect(mocks.batchGetOrFetchDatabases).toHaveBeenCalledTimes(1);
  });
});
