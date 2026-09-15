// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  DatabaseCatalogSchema,
  ObjectSchemaSchema,
  TableCatalog_ColumnsSchema,
  TableCatalogSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  DatabaseMetadataSchema,
  SchemaMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { findGuideQueryTarget } from "./queryTarget";

const metadata = create(DatabaseMetadataSchema, {
  schemas: [
    create(SchemaMetadataSchema, {
      name: "public",
      tables: [
        create(TableMetadataSchema, { name: "users" }),
        create(TableMetadataSchema, { name: "events" }),
      ],
    }),
  ],
});

describe("findGuideQueryTarget", () => {
  test("selects the first table for the query-data guide", () => {
    expect(findGuideQueryTarget(metadata)).toEqual({
      schema: "public",
      table: "users",
    });
  });

  test("selects the table containing a marked relational column", () => {
    const catalog = create(DatabaseCatalogSchema, {
      schemas: [
        {
          name: "public",
          tables: [
            create(TableCatalogSchema, {
              name: "events",
              kind: {
                case: "columns",
                value: create(TableCatalog_ColumnsSchema, {
                  columns: [{ name: "payload", semanticType: "bb.default" }],
                }),
              },
            }),
          ],
        },
      ],
    });

    expect(findGuideQueryTarget(metadata, catalog)).toEqual({
      schema: "public",
      table: "events",
    });
  });

  test("selects the table containing a nested marked document field", () => {
    const catalog = create(DatabaseCatalogSchema, {
      schemas: [
        {
          name: "public",
          tables: [
            create(TableCatalogSchema, {
              name: "users",
              kind: {
                case: "objectSchema",
                value: create(ObjectSchemaSchema, {
                  kind: {
                    case: "structKind",
                    value: {
                      properties: {
                        profile: create(ObjectSchemaSchema, {
                          semanticType: "bb.default",
                        }),
                      },
                    },
                  },
                }),
              },
            }),
          ],
        },
      ],
    });

    expect(findGuideQueryTarget(metadata, catalog)).toEqual({
      schema: "public",
      table: "users",
    });
  });

  test("ignores marked catalog tables missing from current metadata", () => {
    const catalog = create(DatabaseCatalogSchema, {
      schemas: [
        {
          name: "public",
          tables: [
            create(TableCatalogSchema, {
              name: "removed",
              kind: {
                case: "columns",
                value: create(TableCatalog_ColumnsSchema, {
                  columns: [{ name: "secret", semanticType: "bb.default" }],
                }),
              },
            }),
          ],
        },
      ],
    });

    expect(findGuideQueryTarget(metadata, catalog)).toBeUndefined();
  });
});
