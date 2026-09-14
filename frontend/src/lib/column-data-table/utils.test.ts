import { create } from "@bufbuild/protobuf";
import { expect, test, vi } from "vitest";
import {
  type DatabaseCatalog,
  DatabaseCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { updateColumnCatalog } from "./utils";

const mocks = vi.hoisted(() => ({
  getOrFetchDatabaseCatalog: vi.fn(),
  updateDatabaseCatalog: vi.fn<(catalog: DatabaseCatalog) => Promise<void>>(
    async () => {}
  ),
  pushNotification: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks },
}));
vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));
vi.mock("@/lib/i18n", () => ({ default: { t: (key: string) => key } }));

test("updates multiple columns together without losing existing catalog data", async () => {
  const catalog = create(DatabaseCatalogSchema, {
    name: "instances/instance/databases/database/catalog",
    schemas: [
      {
        name: "public",
        tables: [
          {
            name: "customers",
            classification: "sensitive",
            kind: {
              case: "columns",
              value: {
                columns: [
                  { name: "email", labels: { owner: "sales" } },
                  { name: "id", semanticType: "existing" },
                ],
              },
            },
          },
        ],
      },
    ],
  });
  mocks.getOrFetchDatabaseCatalog.mockResolvedValue(catalog);

  await updateColumnCatalog({
    database: "instances/instance/databases/database",
    schema: "public",
    table: "customers",
    column: ["email", "phone"],
    columnCatalog: { semanticType: "bb.default" },
    notification: "common.updated",
  });

  expect(mocks.updateDatabaseCatalog).toHaveBeenCalledTimes(1);
  expect(mocks.updateDatabaseCatalog.mock.calls[0]?.[0]).toMatchObject({
    schemas: [
      {
        tables: [
          {
            classification: "sensitive",
            kind: {
              case: "columns",
              value: {
                columns: [
                  {
                    name: "email",
                    semanticType: "bb.default",
                    labels: { owner: "sales" },
                  },
                  { name: "id", semanticType: "existing" },
                  { name: "phone", semanticType: "bb.default" },
                ],
              },
            },
          },
        ],
      },
    ],
  });
  expect(catalog.schemas[0]?.tables[0]?.kind).toMatchObject({
    case: "columns",
    value: {
      columns: [
        { name: "email", semanticType: "" },
        { name: "id", semanticType: "existing" },
      ],
    },
  });
  expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
});
