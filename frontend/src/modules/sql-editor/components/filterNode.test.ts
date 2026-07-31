import { describe, expect, test } from "vitest";

import type { WorksheetFolderNode } from "@/modules/sql-editor/model/Sheet";

import { filterNode } from "./filterNode";

const makeNode = (
  partial: Partial<WorksheetFolderNode>
): WorksheetFolderNode => ({
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
          key: "__worksheet_load_more__:/my",
          label: "common.load-more",
          loadMore: true,
        })
      )
    ).toBe(true);
  });
});
