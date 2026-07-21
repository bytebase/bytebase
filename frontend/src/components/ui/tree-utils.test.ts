import { describe, expect, test } from "vitest";
import { countVisibleRows } from "./tree-utils";

type TestNode = {
  key: string;
  label: string;
  children?: TestNode[];
};

describe("countVisibleRows", () => {
  test("counts direct children of matching nodes when requested", () => {
    const tree: TestNode = {
      key: "schema",
      label: "billing",
      children: [
        {
          key: "table",
          label: "invoice",
          children: [{ key: "column", label: "amount" }],
        },
        { key: "view", label: "payment_view" },
      ],
    };

    expect(
      countVisibleRows(
        tree,
        new Set(),
        "billing",
        (node, term) => node.label.includes(term),
        true
      )
    ).toBe(3);
  });
});
