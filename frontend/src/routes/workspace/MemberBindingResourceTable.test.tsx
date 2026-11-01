import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { DatabaseResource } from "@/types";
import { MemberBindingResourceTable } from "./MemberBindingResourceTable";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("./MemberDatabaseResourceName", () => ({
  MemberDatabaseResourceName: ({
    resource,
  }: {
    resource?: DatabaseResource;
  }) => <span>{resource?.databaseFullName ?? "*"}</span>,
}));

const columnHeaders = () =>
  screen.getAllByRole("columnheader").map((th) => th.textContent);

describe("MemberBindingResourceTable", () => {
  test("lists database, schema and table, one row per resource, without an expiration column", () => {
    render(
      <MemberBindingResourceTable
        resources={[
          {
            databaseFullName: "instances/prod/databases/hr",
            schema: "public",
            table: "employee",
          },
          { databaseFullName: "instances/prod/databases/hr", schema: "audit" },
        ]}
      />
    );

    expect(columnHeaders()).toEqual([
      "common.database",
      "common.schema",
      "common.table",
    ]);
    const rows = screen.getAllByRole("row").slice(1);
    expect(rows.map((row) => within(row).getAllByRole("cell").map((c) => c.textContent))).toEqual([
      ["instances/prod/databases/hr", "public", "employee"],
      ["instances/prod/databases/hr", "audit", "*"],
    ]);
  });

  test("dragging a column edge widens that column in its own table only", () => {
    const { container } = render(
      <>
        <MemberBindingResourceTable resources={[]} />
        <MemberBindingResourceTable resources={[]} />
      </>
    );
    const [first, second] = Array.from(container.querySelectorAll("table"));
    const colWidths = (table: HTMLTableElement) =>
      Array.from(table.querySelectorAll("col")).map((col) => col.style.width);
    const before = colWidths(first);
    expect(colWidths(second)).toEqual(before);

    const schemaHeader = within(first).getByRole("columnheader", {
      name: "common.schema",
    });
    const handle = schemaHeader.lastElementChild as HTMLElement;
    fireEvent.mouseDown(handle, { clientX: 100 });
    fireEvent.mouseMove(document, { clientX: 180 });
    fireEvent.mouseUp(document);

    expect(colWidths(first)).toEqual([
      before[0],
      `${Number.parseInt(before[1]) + 80}px`,
      before[2],
    ]);
    expect(colWidths(second)).toEqual(before);
  });

  test("an unscoped binding is one row of *", () => {
    render(<MemberBindingResourceTable resources={[]} />);

    const rows = screen.getAllByRole("row").slice(1);
    expect(rows.map((row) => within(row).getAllByRole("cell").map((c) => c.textContent))).toEqual([
      ["*", "*", "*"],
    ]);
  });
});
