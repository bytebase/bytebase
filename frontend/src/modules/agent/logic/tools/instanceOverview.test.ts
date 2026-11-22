// @vitest-environment node

import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";

const mocks = vi.hoisted(() => ({ searchDatabases: vi.fn() }));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { searchDatabases: mocks.searchDatabases },
}));

vi.mock("@/utils/pagination", () => ({ getDefaultPagination: () => 17 }));

import {
  collectInstanceOverview,
  resolveInstanceNameFromPagePath,
} from "./instanceOverview";

describe("instanceOverview", () => {
  beforeEach(() => {
    mocks.searchDatabases.mockReset();
  });

  test.each([
    ["/instances/primary", "instances/primary"],
    [
      "/projects/orders/instances/primary?tab=overview",
      "projects/orders/instances/primary",
    ],
  ])("resolves the exact instance detail path %s", (path, expected) => {
    expect(resolveInstanceNameFromPagePath(path)).toBe(expected);
  });

  test.each([
    "/instances/primary/databases",
    "/instances/create",
    "/instances/new",
    "/projects/-/instances/primary",
    "/projects/orders/instances/primary/databases",
    "/projects/orders/instances/create",
    "/projects/orders/instances/new",
    "https://attacker.example/instances/primary",
  ])("does not resolve non-detail instance path %s", (path) => {
    expect(resolveInstanceNameFromPagePath(path)).toBeUndefined();
  });

  test("returns bounded sync status for the page instance", async () => {
    mocks.searchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/primary/databases/ready",
          syncStatus: SyncStatus.OK,
        },
        {
          name: "instances/primary/databases/broken",
          syncStatus: SyncStatus.FAILED,
        },
        {
          name: "instances/primary/databases/pending",
          syncStatus: SyncStatus.SYNC_STATUS_UNSPECIFIED,
        },
      ],
      nextPageToken: "next-page",
    });

    await expect(collectInstanceOverview("instances/primary")).resolves.toEqual(
      {
        instance: "instances/primary",
        status: "available",
        coverage: "accessible_databases_only",
        databasesOnPage: 3,
        partial: true,
        outOfSyncOnPageCount: 2,
        outOfSyncDatabasesPartial: false,
        outOfSyncDatabases: [
          {
            name: "instances/primary/databases/broken",
            syncStatus: "FAILED",
          },
          {
            name: "instances/primary/databases/pending",
            syncStatus: "SYNC_STATUS_UNSPECIFIED",
          },
        ],
      }
    );
    expect(mocks.searchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        pageSize: 17,
        filter: 'instance == "instances/primary"',
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.searchDatabases.mock.calls[0][1]?.contextValues.get(
        silentContextKey
      )
    ).toBe(true);
  });

  test("does not claim a clean instance when the single page is partial", async () => {
    mocks.searchDatabases.mockResolvedValue({
      databases: [
        {
          name: "projects/orders/instances/primary/databases/ready",
          syncStatus: SyncStatus.OK,
        },
      ],
      nextPageToken: "next-page",
    });

    await expect(
      collectInstanceOverview("projects/orders/instances/primary")
    ).resolves.toMatchObject({
      status: "available",
      partial: true,
      outOfSyncOnPageCount: 0,
    });
  });

  test("ignores databases outside the exact instance scope", async () => {
    mocks.searchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/primary-other/databases/wrong",
          syncStatus: SyncStatus.FAILED,
        },
      ],
      nextPageToken: "",
    });

    await expect(
      collectInstanceOverview("instances/primary")
    ).resolves.toMatchObject({
      databasesOnPage: 0,
      outOfSyncOnPageCount: 0,
      partial: false,
    });
  });

  test("reports the result as unavailable when the API rejects the request", async () => {
    mocks.searchDatabases.mockRejectedValue(new Error("permission denied"));

    await expect(collectInstanceOverview("instances/primary")).resolves.toEqual(
      {
        instance: "instances/primary",
        status: "unavailable",
      }
    );
  });
});
