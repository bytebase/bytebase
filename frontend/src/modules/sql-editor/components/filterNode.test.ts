import { describe, expect, test } from "vitest";

import type { SavedQueryFolderNode } from "@/modules/sql-editor/model/Sheet";

import { filterNode } from "./filterNode";

const makeNode = (
  partial: Partial<SavedQueryFolderNode>
): SavedQueryFolderNode => ({
  ...partial,
  key: "/my/node",
  label: "node",
  editable: false,
  children: [],
});

describe("filterNode", () => {
  test("keeps load-more nodes visible during keyword search", () => {
    const pred = filterNode("/my");

    expect(
      pred(
        "payroll",
        makeNode({
          key: "__savedQuery_load_more__:/my",
          label: "common.load-more",
          loadMore: true,
        })
      )
    ).toBe(true);
  });
});
