import { fireEvent, render, screen, within } from "@testing-library/react";
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

  test.each(["overlay", "agent"] as const)(
    "exposes full highlighted column names in the %s tooltip layer",
    async (tooltipLayer) => {
      const columnName =
        "JSON_SET(document, '$.actions', CAST('{\"example:user-one\":[\"read\",\"edit\"]}' AS JSON))";
      const json = '{"id":"sample","actions":{"example:user-one":["read","edit"]}}';
      const { container } = render(
        <QueryResultVerticalRow
          columns={[
            { id: "expression", name: columnName, value: json },
            { id: "id", name: "id", value: "42" },
          ]}
          tooltipLayer={tooltipLayer}
          renderColumnName={(column) =>
            column.id === "expression" ? (
              <>
                <mark>JSON_SET</mark>
                {column.name.slice(8)}
              </>
            ) : (
              column.name
            )
          }
          renderColumnAccessory={(column) =>
            column.id === "expression" ? (
              <button type="button">Masking reason</button>
            ) : null
          }
          renderValue={(column) => column.value}
        />
      );

      const term = container.querySelector("dt")!;
      expect(within(term).getByText("JSON_SET").tagName).toBe("MARK");
      expect(within(term).getByRole("button", { name: "Masking reason" })).toBeVisible();
      expect(Array.from(container.querySelectorAll("dd"), (value) => value.textContent)).toEqual([
        json,
        "42",
      ]);

      const label = term.querySelector<HTMLElement>("[tabindex='0']");
      expect(label).not.toBeNull();
      fireEvent.focus(label!);

      const tooltipRoot = document.getElementById(`bb-react-layer-${tooltipLayer}`)!;
      expect(await within(tooltipRoot).findByText(columnName)).toBeVisible();
      expect(term).toHaveTextContent(columnName);
    }
  );
});
