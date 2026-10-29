import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { getCellDisplayText, PlainCellValue } from "./PlainCellValue";

describe("PlainCellValue", () => {
  test("distinguishes null, unset, empty, and ordinary values", () => {
    expect(getCellDisplayText(null)).toBe("NULL");
    expect(getCellDisplayText(undefined)).toBe("UNSET");
    expect(getCellDisplayText("")).toBe("");

    const { container } = render(
      <>
        <PlainCellValue value={null} />
        <PlainCellValue value={undefined} />
        <PlainCellValue value="" />
        <PlainCellValue value="visible" />
      </>
    );

    expect(screen.getByText("NULL")).toHaveClass("italic");
    expect(screen.getByText("UNSET")).toHaveClass("italic");
    expect(container.querySelector("br")).not.toBeNull();
    expect(screen.getByText("visible")).toBeInTheDocument();
  });

  test("uses supplied children without hiding null or unset semantics", () => {
    render(
      <>
        <PlainCellValue value={null}>custom null</PlainCellValue>
        <PlainCellValue value="plain">formatted plain</PlainCellValue>
      </>
    );

    expect(screen.getByText("custom null")).toHaveClass("italic");
    expect(screen.getByText("formatted plain")).toBeInTheDocument();
  });
});
