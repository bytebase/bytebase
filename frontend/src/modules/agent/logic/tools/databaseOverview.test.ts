// @vitest-environment node

import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";

const mocks = vi.hoisted(() => ({
  getDatabase: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: {
    getDatabase: mocks.getDatabase,
  },
}));

import {
  collectDatabaseOverview,
  createDatabaseOverviewTool,
  extractDatabaseOverviewEvidence,
} from "./databaseOverview";

const projectName = "projects/orders";
const databaseName = "projects/orders/instances/prod/databases/orders";
const workspaceInstanceDatabaseName = "instances/prod/databases/orders";

describe("databaseOverview", () => {
  beforeEach(() => {
    mocks.getDatabase.mockReset();
  });

  test("collects a bounded view of a database's operational state", async () => {
    mocks.getDatabase.mockResolvedValue({
      name: databaseName,
      project: projectName,
      state: State.ACTIVE,
      instanceResource: {
        name: "instances/prod",
        title: "Production PostgreSQL",
        engine: Engine.POSTGRES,
        engineVersion: "16",
      },
      environment: "environments/production",
      effectiveEnvironment: "environments/production",
      release: "projects/orders/releases/release-20261022",
      backupAvailable: true,
      syncStatus: SyncStatus.OK,
      successfulSyncTime: { seconds: 1_792_648_000n, nanos: 0 },
    });

    await expect(
      collectDatabaseOverview(projectName, databaseName)
    ).resolves.toEqual({
      project: projectName,
      database: {
        status: "available",
        name: databaseName,
        state: "ACTIVE",
        instance: {
          status: "available",
          name: "instances/prod",
          title: "Production PostgreSQL",
          engine: "POSTGRES",
          engineVersion: "16",
        },
        environment: {
          status: "available",
          name: "environments/production",
        },
        release: {
          status: "available",
          name: "projects/orders/releases/release-20261022",
        },
        backupAvailable: true,
        sync: {
          status: "available",
          syncStatus: "OK",
          successfulSyncTime: {
            status: "available",
            value: "2026-10-22T05:46:40.000Z",
          },
        },
      },
      evidence: {
        project: projectName,
        references: [{ kind: "database", name: databaseName }],
      },
    });
    expect(mocks.getDatabase).toHaveBeenCalledWith(
      expect.objectContaining({ name: databaseName })
    );
  });

  test("rejects a database outside the chat-bound project before calling the API", async () => {
    await expect(
      collectDatabaseOverview(
        projectName,
        "projects/orders-archive/instances/prod/databases/orders"
      )
    ).resolves.toBeNull();
    expect(mocks.getDatabase).not.toHaveBeenCalled();
  });

  test("accepts a workspace-instance database after checking its fetched project", async () => {
    mocks.getDatabase.mockResolvedValue({
      name: workspaceInstanceDatabaseName,
      project: projectName,
      state: State.ACTIVE,
      instanceResource: {
        name: "instances/prod",
        title: "Production PostgreSQL",
        engine: Engine.POSTGRES,
        engineVersion: "16",
      },
      backupAvailable: false,
      syncStatus: SyncStatus.OK,
    });

    await expect(
      collectDatabaseOverview(projectName, workspaceInstanceDatabaseName)
    ).resolves.toMatchObject({
      project: projectName,
      database: {
        status: "available",
        name: workspaceInstanceDatabaseName,
      },
      evidence: {
        project: projectName,
        references: [{ kind: "database", name: workspaceInstanceDatabaseName }],
      },
    });
    expect(mocks.getDatabase).toHaveBeenCalledWith(
      expect.objectContaining({ name: workspaceInstanceDatabaseName })
    );
  });

  test("marks a mismatched database response unavailable and withholds evidence", async () => {
    mocks.getDatabase.mockResolvedValue({
      name: "projects/orders/instances/prod/databases/orders_archive",
      project: projectName,
    });

    await expect(
      collectDatabaseOverview(projectName, databaseName)
    ).resolves.toEqual({
      project: projectName,
      database: { status: "unavailable" },
      evidence: { project: projectName, references: [] },
    });
  });

  test("marks a response from a different project unavailable and withholds evidence", async () => {
    mocks.getDatabase.mockResolvedValue({
      name: databaseName,
      project: "projects/orders-archive",
    });

    await expect(
      collectDatabaseOverview(projectName, databaseName)
    ).resolves.toEqual({
      project: projectName,
      database: { status: "unavailable" },
      evidence: { project: projectName, references: [] },
    });
  });

  test("marks a workspace-instance response from a different project unavailable", async () => {
    mocks.getDatabase.mockResolvedValue({
      name: workspaceInstanceDatabaseName,
      project: "projects/orders-archive",
    });

    await expect(
      collectDatabaseOverview(projectName, workspaceInstanceDatabaseName)
    ).resolves.toEqual({
      project: projectName,
      database: { status: "unavailable" },
      evidence: { project: projectName, references: [] },
    });
  });

  test("marks an API failure unavailable rather than empty", async () => {
    mocks.getDatabase.mockRejectedValue(new Error("unavailable"));

    await expect(
      collectDatabaseOverview(projectName, databaseName)
    ).resolves.toEqual({
      project: projectName,
      database: { status: "unavailable" },
      evidence: { project: projectName, references: [] },
    });
  });

  test("only accepts canonical database evidence in the bound project", async () => {
    expect(
      extractDatabaseOverviewEvidence(
        JSON.stringify({
          evidence: {
            project: projectName,
            references: [{ kind: "database", name: databaseName }],
          },
        })
      )
    ).toEqual({
      project: projectName,
      references: [{ kind: "database", name: databaseName }],
    });
    expect(
      extractDatabaseOverviewEvidence(
        JSON.stringify({
          evidence: {
            project: projectName,
            references: [
              { kind: "database", name: workspaceInstanceDatabaseName },
            ],
          },
        })
      )
    ).toEqual({
      project: projectName,
      references: [{ kind: "database", name: workspaceInstanceDatabaseName }],
    });
    expect(
      extractDatabaseOverviewEvidence(
        JSON.stringify({
          evidence: {
            project: projectName,
            references: [
              {
                kind: "database",
                name: "projects/orders-archive/instances/prod/databases/orders",
              },
            ],
          },
        })
      )
    ).toBeUndefined();
  });

  test("rejects a malformed tool argument without calling the API", async () => {
    await expect(
      createDatabaseOverviewTool({ projectName })({
        databaseName: "instances/prod/orders",
      })
    ).resolves.toBe(
      JSON.stringify({
        error: "The database must belong to the project bound to this chat.",
      })
    );
    expect(mocks.getDatabase).not.toHaveBeenCalled();
  });
});
