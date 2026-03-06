import { create as createProto } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { flattenNoSQLQueryResult } from "./utils";

describe("flattenNoSQLQueryResult", () => {
  test("flattens raw MongoDB document rows into table columns on demand", () => {
    const result = createProto(QueryResultSchema, {
      columnNames: ["result"],
      columnTypeNames: ["TEXT"],
      rows: [
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: {
                case: "stringValue",
                value: JSON.stringify({
                  _id: { $oid: "507f1f77bcf86cd799439011" },
                  name: "Ada",
                  profile: {
                    age: { $numberInt: "36" },
                  },
                }),
              },
            }),
          ],
        }),
      ],
    });

    const flattened = flattenNoSQLQueryResult(result);

    expect(flattened?.columnNames).toEqual(["_id", "name", "profile"]);
    expect(flattened?.columnTypeNames).toEqual(["TEXT", "TEXT", "TEXT"]);
    expect(flattened?.rows[0]?.values[0]?.kind.value).toBe(
      "507f1f77bcf86cd799439011"
    );
    expect(flattened?.rows[0]?.values[1]?.kind.value).toBe("Ada");
    expect(flattened?.rows[0]?.values[2]?.kind.value).toBe('{"age":36}');
  });

  test("returns undefined for non-document result sets", () => {
    const result = createProto(QueryResultSchema, {
      columnNames: ["name"],
      columnTypeNames: ["TEXT"],
      rows: [
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: {
                case: "stringValue",
                value: "Ada",
              },
            }),
          ],
        }),
      ],
    });

    expect(flattenNoSQLQueryResult(result)).toBeUndefined();
  });
});
