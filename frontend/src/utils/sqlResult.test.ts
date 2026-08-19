import { create as createProto } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  flattenNoSQLQueryResult,
  formatNoSQLQueryResultAsJSON,
  formatNoSQLQueryResultForJSONView,
} from "./sqlResult";

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

  test("returns undefined when any document row contains malformed JSON", () => {
    const result = createProto(QueryResultSchema, {
      columnNames: ["result"],
      rows: [
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: { case: "stringValue", value: '{"name":"Ada"}' },
            }),
          ],
        }),
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: { case: "stringValue", value: "{" },
            }),
          ],
        }),
      ],
    });

    expect(flattenNoSQLQueryResult(result)).toBeUndefined();
  });
});

describe("formatNoSQLQueryResultAsJSON", () => {
  test("formats document rows without changing field order or large numbers", () => {
    const result = createProto(QueryResultSchema, {
      columnNames: ["result"],
      columnTypeNames: ["TEXT"],
      rows: [
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: {
                case: "stringValue",
                value:
                  '{"id":"one","profile":{"tags":["a","b"]},"counter":9007199254740993}',
              },
            }),
          ],
        }),
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: {
                case: "stringValue",
                value:
                  '{"id":"two","value":{"$numberLong":"9007199254740994"}}',
              },
            }),
          ],
        }),
      ],
    });

    expect(formatNoSQLQueryResultAsJSON(result)).toBe(`[
  {
    "id": "one",
    "profile": {
      "tags": [
        "a",
        "b"
      ]
    },
    "counter": 9007199254740993
  },
  {
    "id": "two",
    "value": {
      "$numberLong": "9007199254740994"
    }
  }
]`);

    expect(formatNoSQLQueryResultForJSONView(result)?.documents).toEqual([
      {
        content: `{
  "id": "one",
  "profile": {
    "tags": [
      "a",
      "b"
    ]
  },
  "counter": 9007199254740993
}`,
        startLineNumber: 2,
        endLineNumber: 11,
      },
      {
        content: `{
  "id": "two",
  "value": {
    "$numberLong": "9007199254740994"
  }
}`,
        startLineNumber: 12,
        endLineNumber: 17,
      },
    ]);
  });

  test("rejects results that are not JSON document rows", () => {
    const wrongColumn = createProto(QueryResultSchema, {
      columnNames: ["name"],
      rows: [],
    });
    const nonStringValue = createProto(QueryResultSchema, {
      columnNames: ["result"],
      rows: [
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: { case: "int32Value", value: 1 },
            }),
          ],
        }),
      ],
    });
    const malformedJSON = createProto(QueryResultSchema, {
      columnNames: ["result"],
      rows: [
        createProto(QueryRowSchema, {
          values: [
            createProto(RowValueSchema, {
              kind: { case: "stringValue", value: "{" },
            }),
          ],
        }),
      ],
    });

    expect(formatNoSQLQueryResultAsJSON(wrongColumn)).toBeUndefined();
    expect(formatNoSQLQueryResultAsJSON(nonStringValue)).toBeUndefined();
    expect(formatNoSQLQueryResultAsJSON(malformedJSON)).toBeUndefined();
  });
});
