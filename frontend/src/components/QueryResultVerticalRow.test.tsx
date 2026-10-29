import { render, screen, within } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { QueryResultVerticalRow } from "./QueryResultVerticalRow";

const columns = [
  { id: "id", name: "Identifier", value: "42" },
  { id: "name", name: "Name", value: "Sample" },
];

describe("QueryResultVerticalRow", () => {
  test("renders accessible name/value pairs and optional row actions", () => {
    const { container } = render(
      <QueryResultVerticalRow
        columns={columns}
        renderValue={(column) => column.value}
        actions={<button type="button">Copy row</button>}
        data-testid="result-row"
      />
    );

    const row = screen.getByTestId("result-row");
    expect(within(row).getByRole("button", { name: "Copy row" })).toBeVisible();
    expect(container.querySelectorAll("dl")).toHaveLength(1);
    expect(container.querySelectorAll("dt")).toHaveLength(2);
    expect(container.querySelectorAll("dd")).toHaveLength(2);
    expect(container.querySelector("dt")?.textContent).toContain("Identifier");
    expect(container.querySelector("dd")?.textContent).toBe("42");
  });

  test("accepts column labels, accessories, and compact style without SQL Editor context", () => {
    render(
      <QueryResultVerticalRow
        columns={columns}
        className="bg-transparent px-0"
        renderColumnName={(column) => column.name.toUpperCase()}
        renderColumnAccessory={(column) =>
          column.id === "id" ? <span>Masked</span> : null
        }
        renderValue={(column) => column.value}
        data-testid="result-row"
      />
    );

    expect(screen.getByTestId("result-row")).toHaveClass("bg-transparent");
    expect(screen.getByText("IDENTIFIER")).toBeInTheDocument();
    expect(screen.getByText("Masked")).toBeInTheDocument();
  });
});
