import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";

const mocks = vi.hoisted(() => ({
  listRevisions: vi.fn(),
}));

vi.mock("@/connect", () => ({
  revisionServiceClientConnect: {
    listRevisions: mocks.listRevisions,
  },
}));

let useRevisionStore: typeof import("./revision").useRevisionStore;

beforeEach(async () => {
  setActivePinia(createPinia());
  mocks.listRevisions.mockReset();

  vi.resetModules();
  ({ useRevisionStore } = await import("./revision"));
});

describe("revision store", () => {
  test("fetchAllRevisionsByDatabase keeps loading until every page is fetched", async () => {
    mocks.listRevisions
      .mockResolvedValueOnce({
        nextPageToken: "page-2",
        revisions: [
          {
            name: "instances/inst1/databases/db1/revisions/1",
            version: "1.0.0",
          } as Revision,
        ],
      })
      .mockResolvedValueOnce({
        nextPageToken: "",
        revisions: [
          {
            name: "instances/inst1/databases/db1/revisions/2",
            version: "2.0.0",
          } as Revision,
        ],
      });

    const store = useRevisionStore();
    const revisions = await store.fetchAllRevisionsByDatabase(
      "instances/inst1/databases/db1",
      { pageSize: 1000 }
    );

    expect(mocks.listRevisions).toHaveBeenCalledTimes(2);
    expect(mocks.listRevisions.mock.calls[0]?.[0]).toMatchObject({
      parent: "instances/inst1/databases/db1",
      pageSize: 1000,
      pageToken: "",
    });
    expect(mocks.listRevisions.mock.calls[1]?.[0]).toMatchObject({
      parent: "instances/inst1/databases/db1",
      pageSize: 1000,
      pageToken: "page-2",
    });
    expect(revisions.map((revision) => revision.version)).toEqual([
      "1.0.0",
      "2.0.0",
    ]);
  });
});
