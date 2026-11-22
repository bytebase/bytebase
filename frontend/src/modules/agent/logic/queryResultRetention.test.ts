import { describe, expect, test } from "vitest";
import {
  boundAgentQueryResult,
  getAgentQuerySampleBytes,
  getAgentQuerySampleInfo,
  omitAgentQueryRows,
} from "./queryResultRetention";

const row = (value: string) => ({ values: [{ stringValue: value }] });
const envelope = (results: unknown[], extra = {}) =>
  JSON.stringify({ status: 200, response: { results }, ...extra });
const sampleBytes = (results: { rows?: unknown[] }[]) =>
  results.reduce(
    (bytes, result) =>
      bytes +
      (result.rows?.length
        ? new TextEncoder().encode(JSON.stringify(result.rows)).length
        : 0),
    0
  );

describe("getAgentQuerySampleInfo", () => {
  test("inspects row bytes and runtime metadata together without classifying invalid envelopes", () => {
    const raw = envelope([
      { columnNames: ["value"], rows: [row("a")] },
      { error: "later error" },
    ]);
    expect(getAgentQuerySampleInfo(raw)).toEqual({ bytes: 34 });
    expect(getAgentQuerySampleInfo(boundAgentQueryResult(raw, 33))).toEqual({
      bytes: 0,
      preview: {
        version: 1,
        level: "sampled",
        results: [
          { receivedRows: 1, retainedRows: 0 },
          { receivedRows: 0, retainedRows: 0 },
        ],
      },
    });
    expect(getAgentQuerySampleInfo(omitAgentQueryRows(raw))).toEqual({
      bytes: 0,
      preview: {
        version: 1,
        level: "omitted",
        results: [
          { receivedRows: 1, retainedRows: 0 },
          { receivedRows: 0, retainedRows: 0 },
        ],
      },
    });
    expect(getAgentQuerySampleInfo("not JSON")).toBeUndefined();
    expect(
      getAgentQuerySampleInfo(
        envelope([{ rows: [row("a")] }], {
          agentQueryPreview: { version: 2, level: "omitted", results: [] },
        })
      )
    ).toBeUndefined();
  });
});

describe("boundAgentQueryResult", () => {
  test("omits a huge first typed cell without slicing it or retaining later rows", () => {
    const raw = envelope([
      {
        columnNames: ["payload"],
        rows: [row("x".repeat(70_000)), row("small")],
      },
    ]);
    const bounded = JSON.parse(boundAgentQueryResult(raw));

    expect(bounded.response.results[0].rows).toEqual([]);
    expect(bounded.agentQueryPreview).toEqual({
      version: 1,
      level: "sampled",
      results: [{ receivedRows: 2, retainedRows: 0 }],
    });
    expect(sampleBytes(bounded.response.results)).toBeLessThanOrEqual(
      64 * 1024
    );
  });

  test("retains complete typed rows at the exact UTF-8 byte boundary", () => {
    const rows = [
      row("汉😀"),
      { values: [{ int64Value: "9007199254740993" }] },
    ];
    const raw = envelope([{ columnNames: ["value"], rows }]);
    const bytes = new TextEncoder().encode(JSON.stringify(rows)).length;

    expect(
      JSON.parse(boundAgentQueryResult(raw, bytes)).response.results[0].rows
    ).toEqual(rows);
    const bounded = JSON.parse(boundAgentQueryResult(raw, bytes - 1));
    expect(bounded.response.results[0].rows).toEqual([rows[0]]);
    expect(bounded.agentQueryPreview.results).toEqual([
      { receivedRows: 2, retainedRows: 1 },
    ]);
    expect(sampleBytes(bounded.response.results)).toBeLessThanOrEqual(
      bytes - 1
    );
  });

  test("shares one row budget across result sets and preserves later result errors", () => {
    const first = row("first");
    const second = row("second");
    const agentExecution = {
      version: 1,
      attemptId: "attempt-1",
      result: "succeeded",
      effects: "possible",
      reason: "query-result-error",
      httpStatus: 200,
    };
    const raw = envelope(
      [
        { columnNames: ["value"], rows: [first] },
        { columnNames: ["value"], rows: [second] },
        {
          error: "permission denied",
          statement: "SELECT private_value",
          rows: [],
        },
      ],
      { agentExecution }
    );
    const bounded = JSON.parse(
      boundAgentQueryResult(raw, sampleBytes([{ rows: [first] }]))
    );

    expect(
      bounded.response.results.map((result: { rows: unknown[] }) => result.rows)
    ).toEqual([[first], [], []]);
    expect(bounded.response.results[2]).toEqual({
      error: "permission denied",
      statement: "SELECT private_value",
      rows: [],
    });
    expect(bounded.agentQueryPreview.results).toEqual([
      { receivedRows: 1, retainedRows: 1 },
      { receivedRows: 1, retainedRows: 0 },
      { receivedRows: 0, retainedRows: 0 },
    ]);
    expect(bounded.agentExecution).toEqual(agentExecution);
    expect(
      JSON.parse(omitAgentQueryRows(JSON.stringify(bounded))).agentExecution
    ).toEqual(agentExecution);
  });

  test.each([
    { level: "sampled", retain: boundAgentQueryResult },
    { level: "omitted", retain: omitAgentQueryRows },
  ])(
    "preserves masking beyond displayed columns and Access Grants with $level rows",
    ({ level, retain }) => {
      const columns = Array.from(
        { length: 25 },
        (_, index) => `column_${index}`
      );
      const result = {
        statement: "SELECT * FROM records",
        columnNames: columns,
        columnTypeNames: columns.map(() => "TEXT"),
        rows: [row("x".repeat(70_000))],
        rowsCount: "2000",
        masked: columns.map((_, index) =>
          index === 24
            ? {
                semanticTypeId: "private_value",
                semanticTypeTitle: "Private Value",
                maskingRuleId: "masking-rule",
                algorithm: "full-mask",
                context: "Matched masking rule",
                classificationLevel: 2,
                semanticTypeIcon: "",
              }
            : {}
        ),
        permissionDenied: {
          method: "bytebase.v1.SQLService.Query",
          requiredPermissions: ["bb.sql.select"],
          resources: ["instances/example/databases/app"],
        },
        queryPlan: {
          format: "JSON",
          executed: true,
          content: "protected plan",
        },
        unknownFutureField: { retained: true },
      };
      const raw = JSON.stringify({
        status: 200,
        response: {
          results: [result],
          appliedAccessGrant: "projects/example/accessGrants/123",
          allowExport: false,
          advices: [{ title: "review warning" }],
        },
        traceId: "trace",
      });
      const bounded = JSON.parse(retain(raw));

      expect(bounded).toEqual({
        ...JSON.parse(raw),
        response: {
          ...JSON.parse(raw).response,
          results: [{ ...result, rows: [] }],
        },
        agentQueryPreview: {
          version: 1,
          level,
          results: [{ receivedRows: 1, retainedRows: 0 }],
        },
      });
    }
  );

  test.each([
    { level: "sampled", retain: boundAgentQueryResult },
    { level: "omitted", retain: omitAgentQueryRows },
  ])(
    "does not invent masking evidence from values or empty reasons with $level rows",
    ({ retain }) => {
      const raw = envelope([
        { columnNames: ["literal"], rows: [row("******")] },
        { columnNames: ["value"], rows: [row("plain")], masked: [{}] },
      ]);
      const retained = JSON.parse(retain(raw));

      expect(retained.response.results[0]).not.toHaveProperty("masked");
      expect(retained.response.results[1].masked).toEqual([{}]);
    }
  );

  test("preserves detailed HTTP errors byte-for-byte", () => {
    const raw = JSON.stringify(
      {
        status: 403,
        error: "Access denied",
        response: {
          code: "permission_denied",
          details: [
            {
              resource: "instances/example/databases/app",
              permissionDenied: { reason: "request an Access Grant" },
            },
          ],
        },
      },
      null,
      2
    );
    expect(boundAgentQueryResult(raw)).toBe(raw);
    expect(omitAgentQueryRows(raw)).toBe(raw);
  });

  test("is idempotent and keeps received counts when a bounded sample is reduced further", () => {
    const raw = envelope([
      { columnNames: ["value"], rows: [row("a"), row("b")] },
    ]);
    const first = boundAgentQueryResult(raw);
    const reduced = boundAgentQueryResult(
      first,
      sampleBytes([{ rows: [row("a")] }])
    );

    expect(boundAgentQueryResult(first)).toBe(first);
    expect(boundAgentQueryResult(reduced)).toBe(reduced);
    expect(JSON.parse(reduced).agentQueryPreview.results).toEqual([
      { receivedRows: 2, retainedRows: 1 },
    ]);
  });

  test.each([
    "not JSON",
    envelope([{ rows: "not rows" }]),
    envelope([{ rows: ["not a typed row"] }]),
    envelope([{ rows: [{ values: "not values" }] }]),
    envelope([{ rows: [{ values: [{ stringValue: 42 }] }] }]),
    envelope([{ rows: [{ values: [{ unknownValue: "protected" }] }] }]),
    envelope([{ columnNames: [17], rows: [] }]),
    JSON.stringify({ status: 200, response: { projects: [] } }),
  ])("protects malformed or non-query envelopes: %s", (raw) => {
    expect(boundAgentQueryResult(raw)).toBe(raw);
    expect(omitAgentQueryRows(raw)).toBe(raw);
  });

  test.each([
    {
      version: 2,
      level: "sampled",
      results: [{ receivedRows: 1, retainedRows: 1 }],
    },
    { version: 1, level: "sampled", results: [] },
    {
      version: 1,
      level: "sampled",
      results: [{ receivedRows: 0, retainedRows: 1 }],
    },
    {
      version: 1,
      level: "omitted",
      results: [{ receivedRows: 1, retainedRows: 1 }],
    },
  ])(
    "protects unknown or inconsistent preview metadata: %j",
    (agentQueryPreview) => {
      const raw = envelope(
        [{ columnNames: ["value"], rows: [row("protected")] }],
        { agentQueryPreview }
      );
      expect(boundAgentQueryResult(raw, 0)).toBe(raw);
      expect(omitAgentQueryRows(raw)).toBe(raw);
      expect(getAgentQuerySampleBytes(raw)).toBeUndefined();
    }
  );

  test("measures row samples independently of protected envelope bytes", () => {
    const raw = envelope([
      {
        columnNames: ["value"],
        statement: "x".repeat(70_000),
        rows: [row("汉😀")],
      },
    ]);
    expect(getAgentQuerySampleBytes(raw)).toBe(40);
    expect(getAgentQuerySampleBytes(omitAgentQueryRows(raw))).toBe(0);
  });
});

describe("omitAgentQueryRows", () => {
  test("irreversibly omits only rows while retaining original counts and all outcomes", () => {
    const raw = envelope([
      { columnNames: ["secret"], rows: [row("masked"), row("masked")] },
      { error: "later error", rows: [] },
    ]);
    const omitted = omitAgentQueryRows(boundAgentQueryResult(raw));
    const result = JSON.parse(omitted);

    expect(result.response.results).toEqual([
      { columnNames: ["secret"], rows: [] },
      { error: "later error", rows: [] },
    ]);
    expect(result.agentQueryPreview).toEqual({
      version: 1,
      level: "omitted",
      results: [
        { receivedRows: 2, retainedRows: 0 },
        { receivedRows: 0, retainedRows: 0 },
      ],
    });
    expect(omitAgentQueryRows(omitted)).toBe(omitted);
    expect(boundAgentQueryResult(omitted)).toBe(omitted);
  });
});
