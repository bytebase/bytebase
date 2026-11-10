import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import {
  convertFromExpr,
  getGuaranteedExpiration,
  readableCondition,
  stringifyConditionExpression,
} from "./cel";

// Trees shaped as the server's parser returns them.
const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const select = (operand: Expr, field: string) =>
  create(ExprSchema, {
    exprKind: { case: "selectExpr", value: { operand, field } },
  });
const call = (fn: string, ...args: Expr[]) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args } },
  });
const stringCall = (fn: string, attribute: string, value: string) =>
  create(ExprSchema, {
    exprKind: {
      case: "callExpr",
      value: {
        function: fn,
        target: resource(attribute),
        args: [str(value)],
      },
    },
  });
const constant = (
  constantKind:
    | { case: "stringValue"; value: string }
    | { case: "boolValue"; value: boolean }
    | { case: "int64Value"; value: bigint }
) =>
  create(ExprSchema, {
    exprKind: { case: "constExpr", value: { constantKind } },
  });
const str = (value: string) => constant({ case: "stringValue", value });
const bool = (value: boolean) => constant({ case: "boolValue", value });
const int = (value: number) =>
  constant({ case: "int64Value", value: BigInt(value) });
const list = (...elements: Expr[]) =>
  create(ExprSchema, { exprKind: { case: "listExpr", value: { elements } } });
const resource = (field: string) => select(ident("resource"), field);
const eq = (attribute: string, value: string) =>
  call("_==_", resource(attribute), str(value));
const envIn = (...ids: string[]) =>
  call("@in", resource("environment_id"), list(...ids.map(str)));
const timeBefore = (iso: string) =>
  call("_<_", select(ident("request"), "time"), call("timestamp", str(iso)));
const and = (...exprs: Expr[]) =>
  exprs.reduce((left, right) => call("_&&_", left, right));
const or = (...exprs: Expr[]) =>
  exprs.reduce((left, right) => call("_||_", left, right));

const environments = (expr: Expr) => readableCondition(expr).environments;
const A = "instances/i/databases/a";

describe("stringifyConditionExpression", () => {
  test("the switch off is byte-for-byte the empty picker's clause", () => {
    expect(stringifyConditionExpression({ environments: [] })).toBe(
      "resource.environment_id in []"
    );
    expect(stringifyConditionExpression({ environments: undefined })).toBe("");
    expect(
      stringifyConditionExpression({
        environments: ["environments/staging", "environments/prod"],
      })
    ).toBe('resource.environment_id in ["staging", "prod"]');
  });
});

describe("readableCondition proves the environments and keeps the rest best-effort", () => {
  test("the environment clause alone, and beside the other clauses the forms write", () => {
    expect(environments(envIn())).toEqual([]);
    expect(environments(envIn("test", "test"))).toEqual(["environments/test"]);
    expect(
      readableCondition(
        and(
          eq("database", A),
          timeBefore("2026-10-01T00:00:00Z"),
          envIn("prod")
        )
      )
    ).toEqual({
      databaseResources: [{ databaseFullName: A }],
      expiredTime: "2026-10-01T00:00:00.000Z",
      environments: ["environments/prod"],
    });
  });

  test("the frontend builder's dotted ident is read too", () => {
    expect(
      environments(
        call("@in", ident("resource.environment_id"), list(str("prod")))
      )
    ).toEqual(["environments/prod"]);
  });

  test("a clause under || or ! does not count: `in [] || true` is not the switch off", () => {
    expect(convertFromExpr(or(envIn(), bool(true)))?.environments).toEqual([]);
    expect(environments(or(envIn(), bool(true)))).toBeUndefined();
    expect(environments(or(envIn("prod"), envIn("dev")))).toBeUndefined();
    expect(
      environments(and(bool(true), or(envIn("prod"), bool(true))))
    ).toBeUndefined();
    expect(environments(call("!_", envIn()))).toBeUndefined();
  });

  test("a computed or non-string environment list does not count", () => {
    expect(
      environments(
        call(
          "@in",
          resource("environment_id"),
          call("_+_", list(str("prod")), list(str("dev")))
        )
      )
    ).toBeUndefined();
    expect(
      environments(call("@in", resource("environment_id"), list(int(1))))
    ).toBeUndefined();
  });

  test("two root environment clauses intersect", () => {
    expect(environments(and(envIn("prod", "dev"), envIn("dev")))).toEqual([
      "environments/dev",
    ]);
  });
});

describe("getGuaranteedExpiration", () => {
  test("finds an expiration beside independent disjunctions", () => {
    const expiration = "2020-01-01T00:00:00Z";
    expect(
      getGuaranteedExpiration(
        and(
          or(eq("database", "instances/i/databases/a"), eq("database", A)),
          or(eq("schema_name", "public"), eq("schema_name", "internal")),
          timeBefore(expiration)
        )
      )?.toISOString()
    ).toBe("2020-01-01T00:00:00.000Z");
  });

  test("ignores an expiration that is only one OR branch", () => {
    expect(
      getGuaranteedExpiration(
        or(timeBefore("2020-01-01T00:00:00Z"), eq("database", A))
      )
    ).toBeUndefined();
  });
});

describe("convertFromExpr with a column condition", () => {
  const columnIn = (...names: string[]) =>
    call("@in", resource("column_name"), list(...names.map(str)));
  const databaseResources = (expr: Expr) =>
    convertFromExpr(expr)?.databaseResources;

  test("keeps the columns of a column-only in condition", () => {
    expect(databaseResources(columnIn("email", "phone"))).toEqual([
      { databaseFullName: "", columns: ["email", "phone"] },
    ]);
  });

  test.each(["schema_name", "table_name"])(
    "keeps a %s-only in scope rather than all databases",
    (attribute) => {
      expect(
        databaseResources(
          call("@in", resource(attribute), list(str("archive"), str("events")))
        )
      ).toEqual([
        {
          databaseFullName: "",
          [attribute === "schema_name" ? "schema" : "table"]: "archive",
        },
        {
          databaseFullName: "",
          [attribute === "schema_name" ? "schema" : "table"]: "events",
        },
      ]);
    }
  );

  test("collects columns from equality conditions joined by or", () => {
    expect(
      databaseResources(
        or(eq("column_name", "email"), eq("column_name", "phone"))
      )
    ).toEqual([{ databaseFullName: "", columns: ["email", "phone"] }]);
  });

  test("keeps disjunctive column scopes in separate resources", () => {
    expect(
      databaseResources(
        or(
          columnIn("a", "b"),
          and(
            eq("instance_id", "mysql-instance"),
            eq("database_name", "bytebase_210"),
            eq("column_name", "c")
          )
        )
      )
    ).toEqual([
      { databaseFullName: "", columns: ["a", "b"] },
      {
        databaseFullName: "instances/mysql-instance/databases/bytebase_210",
        instanceResourceId: "mysql-instance",
        databaseResourceId: "bytebase_210",
        columns: ["c"],
      },
    ]);
  });
  test("carries shared conditions into nested OR branches", () => {
    expect(
      databaseResources(
        and(
          timeBefore("2026-10-01T00:00:00Z"),
          or(
            columnIn("a"),
            and(
              eq("instance_id", "mysql-instance"),
              eq("database_name", "bytebase_210"),
              eq("column_name", "c")
            )
          )
        )
      )
    ).toEqual([
      { databaseFullName: "", columns: ["a"] },
      {
        databaseFullName: "instances/mysql-instance/databases/bytebase_210",
        instanceResourceId: "mysql-instance",
        databaseResourceId: "bytebase_210",
        columns: ["c"],
      },
    ]);
  });

  test("keeps a broad OR branch broad when another branch names columns", () => {
    expect(
      databaseResources(
        or(eq("database", A), and(eq("database", A), eq("column_name", "c")))
      )
    ).toEqual([{ databaseFullName: A }]);
  });

  test("falls back to the raw condition for independent OR groups", () => {
    expect(
      convertFromExpr(
        and(
          or(
            eq("database", "instances/i/databases/a"),
            eq("database", "instances/i/databases/b")
          ),
          or(eq("column_name", "email"), eq("column_name", "phone"))
        )
      )
    ).toBeUndefined();
  });
});

describe("convertFromExpr preserves unsupported scopes as raw conditions", () => {
  test.each(["contains", "startsWith", "endsWith", "matches"])(
    "does not omit a %s predicate from a database scope",
    (operator) => {
      const predicate = stringCall(operator, "table_name", "archive");
      expect(convertFromExpr(predicate)).toBeUndefined();
      expect(
        convertFromExpr(and(eq("database", A), predicate))
      ).toBeUndefined();
    }
  );

  test("does not omit a negated string predicate", () => {
    expect(
      convertFromExpr(
        call("!_", stringCall("contains", "column_name", "email"))
      )
    ).toBeUndefined();
  });

  test("does not omit an inequality or negative list membership", () => {
    expect(
      convertFromExpr(call("_!=_", resource("table_name"), str("archive")))
    ).toBeUndefined();
    expect(
      convertFromExpr(
        call("!_", call("@in", resource("table_name"), list(str("archive"))))
      )
    ).toBeUndefined();
  });

  test("does not turn a computed exact resource into a literal scope", () => {
    expect(
      convertFromExpr(
        call("_==_", resource("table_name"), call("_+_", str("a"), str("b")))
      )
    ).toBeUndefined();
  });

  test("retains classification comparisons and expiration beside an exact scope", () => {
    expect(
      convertFromExpr(
        and(
          eq("database", A),
          call("_<=_", resource("classification_level"), int(3)),
          timeBefore("2026-10-01T00:00:00Z")
        )
      )
    ).toEqual({
      databaseResources: [{ databaseFullName: A }],
      expiredTime: "2026-10-01T00:00:00.000Z",
    });
  });

  test("keeps a guaranteed expiration when the resource scope stays raw", () => {
    const result = readableCondition(
      and(
        stringCall("contains", "table_name", "archive"),
        timeBefore("2026-10-01T00:00:00Z"),
        envIn("prod")
      )
    );
    expect(result.databaseResources).toBeUndefined();
    expect(result.expiredTime).toBe("2026-10-01T00:00:00.000Z");
    expect(result.environments).toEqual(["environments/prod"]);
  });
});
