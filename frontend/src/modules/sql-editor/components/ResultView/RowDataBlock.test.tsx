import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import { RowDataBlock } from "./RowDataBlock";

vi.mock("@/modules/sql-editor/components/MaskingReasonPopover", () => ({
  MaskingReasonPopover: ({
    database,
    statement,
  }: {
    database: string;
    statement?: string;
  }) => <span>{`${database}:${statement}`}</span>,
}));

const columns = [
  { id: "one", name: "First", columnType: "TEXT" },
  { id: "two", name: "Second", columnType: "TEXT" },
];

describe("RowDataBlock", () => {
  test("keeps SQL Editor masking context beside the affected column", () => {
    const { container } = render(
      <RowDataBlock
        columns={columns}
        database="instances/example/databases/sample"
        statement="SELECT * FROM example"
        getMaskingReason={(index) =>
          index === 0 ? ({} as MaskingReason) : undefined
        }
        renderValue={(_, index) => (index === 0 ? "masked" : "visible")}
      />
    );

    expect(screen.getByTestId("row-data-block")).toBeInTheDocument();
    expect(container.querySelectorAll("dt")).toHaveLength(2);
    expect(screen.getByText("First").closest("dt")?.textContent).toContain(
      "instances/example/databases/sample:SELECT * FROM example"
    );
    expect(screen.getByText("Second").closest("dt")?.textContent).not.toContain(
      "instances/example/databases/sample"
    );
  });
});
