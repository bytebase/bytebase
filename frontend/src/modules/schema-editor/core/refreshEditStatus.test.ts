import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  ColumnMetadataSchema,
  DatabaseMetadataSchema,
  SchemaMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { EditStatusContext } from "../types";
import { refreshTableEditStatus } from "./refreshEditStatus";

describe("refreshTableEditStatus", () => {
  test("preserves created table status while notifying metadata changes", () => {
    const db = { name: "instances/test/databases/db" } as Database;
    const baselineMetadata = create(DatabaseMetadataSchema, {
      name: "db",
      schemas: [
        create(SchemaMetadataSchema, {
          name: "public",
          tables: [],
        }),
      ],
    });
    const column = create(ColumnMetadataSchema, {
      name: "id",
      type: "bigint",
    });
    const table = create(TableMetadataSchema, {
      name: "users",
      columns: [column],
    });
    const schema = create(SchemaMetadataSchema, {
      name: "public",
      tables: [table],
    });
    const editStatus = {
      getEditStatusByKey: vi.fn(() => "created"),
      markEditStatus: vi.fn(),
      removeEditStatus: vi.fn(),
      getColumnStatus: vi.fn(() => "created"),
    } as unknown as EditStatusContext;

    refreshTableEditStatus(editStatus, db, baselineMetadata, schema, table);

    expect(editStatus.markEditStatus).toHaveBeenCalledWith(
      db,
      { schema, table },
      "created"
    );
    expect(editStatus.removeEditStatus).not.toHaveBeenCalled();
  });
});
