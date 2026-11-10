// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  type Constant,
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { getMaskingDraftCELShape } from "./maskingDraftCEL";

const ident = (name: string, id = 0n) =>
  create(ExprSchema, { id, exprKind: { case: "identExpr", value: { name } } });
const call = (fn: string, args: Expr[], target?: Expr) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args, target } },
  });
const select = (operand: Expr, field: string, testOnly = false) =>
  create(ExprSchema, {
    exprKind: { case: "selectExpr", value: { operand, field, testOnly } },
  });

describe("masking draft CEL shape", () => {
  it("ignores expression IDs and equivalent direct factor notation", () => {
    const expected = ["factor", "resource.column_name"];
    expect(
      getMaskingDraftCELShape(ident("resource.column_name", 1n), "rule")
    ).toEqual(expected);
    expect(
      getMaskingDraftCELShape(
        select(ident("resource", 2n), "column_name"),
        "rule"
      )
    ).toEqual(expected);
  });

  it.each<[string, Constant["constantKind"], string]>([
    ["stringValue", { case: "stringValue", value: "1" }, "1"],
    ["int64Value", { case: "int64Value", value: 1n }, "1"],
    ["uint64Value", { case: "uint64Value", value: 1n }, "1"],
    ["doubleValue", { case: "doubleValue", value: 1 }, "1"],
    ["boolValue", { case: "boolValue", value: true }, "true"],
    [
      "bytesValue",
      { case: "bytesValue", value: new Uint8Array([1, 2]) },
      "1,2",
    ],
    ["nullValue", { case: "nullValue", value: 0 }, "0"],
  ])("retains constant type and value for %s", (type, constantKind, value) => {
    const expr = create(ExprSchema, {
      exprKind: { case: "constExpr", value: { constantKind } },
    });
    expect(getMaskingDraftCELShape(expr, "rule")).toEqual([
      "constant",
      type,
      value,
    ]);
  });

  it.each(["_&&_", "_||_"])(
    "flattens only binary associativity for %s while retaining operand order",
    (operator) => {
      const first = ident("first");
      const second = ident("second");
      const third = ident("third");
      const expected = [
        operator,
        [
          ["factor", "first"],
          ["factor", "second"],
          ["factor", "third"],
        ],
      ];
      expect(
        getMaskingDraftCELShape(
          call(operator, [call(operator, [first, second]), third]),
          "rule"
        )
      ).toEqual(expected);
      expect(
        getMaskingDraftCELShape(
          call(operator, [first, call(operator, [second, third])]),
          "exemption"
        )
      ).toEqual(expected);
    }
  );

  it("preserves nested operators, call receivers, and argument order", () => {
    expect(
      getMaskingDraftCELShape(
        call("_&&_", [
          ident("first"),
          call("_||_", [ident("second"), ident("third")], ident("receiver")),
        ]),
        "rule"
      )
    ).toEqual([
      "_&&_",
      [
        ["factor", "first"],
        [
          "call",
          "_||_",
          ["factor", "receiver"],
          [
            ["factor", "second"],
            ["factor", "third"],
          ],
        ],
      ],
    ]);
  });

  it("does not flatten OR operands into an enclosing AND", () => {
    expect(
      getMaskingDraftCELShape(
        call("_&&_", [
          ident("first"),
          call("_||_", [ident("second"), ident("third")]),
        ]),
        "rule"
      )
    ).toEqual([
      "_&&_",
      [
        ["factor", "first"],
        [
          "_||_",
          [
            ["factor", "second"],
            ["factor", "third"],
          ],
        ],
      ],
    ]);
  });

  it("retains list element order and optional indices", () => {
    const expr = create(ExprSchema, {
      exprKind: {
        case: "listExpr",
        value: {
          elements: [ident("first"), ident("second")],
          optionalIndices: [1],
        },
      },
    });
    expect(getMaskingDraftCELShape(expr, "rule")).toEqual([
      "list",
      [
        ["factor", "first"],
        ["factor", "second"],
      ],
      [1],
    ]);
  });

  it.each<[string, Expr]>([
    ["empty expression", create(ExprSchema)],
    ["nested selection", select(select(ident("resource"), "nested"), "field")],
    ["test-only selection", select(ident("resource"), "column_name", true)],
    [
      "struct",
      create(ExprSchema, { exprKind: { case: "structExpr", value: {} } }),
    ],
    [
      "comprehension",
      create(ExprSchema, {
        exprKind: { case: "comprehensionExpr", value: {} },
      }),
    ],
    ["unary AND", call("_&&_", [ident("first")])],
    [
      "ternary OR",
      call("_||_", [ident("first"), ident("second"), ident("third")]),
    ],
  ])("rejects %s with the owning selector context", (_, expr) => {
    for (const selector of ["rule", "exemption"] as const)
      expect(() => getMaskingDraftCELShape(expr, selector)).toThrow(
        `The condition cannot be represented by the ${selector} selector.`
      );
  });
});
