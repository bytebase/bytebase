import { describe, expect, test } from "vitest";
import { TableBody, TableRow } from "./table";

describe("table primitives", () => {
  test("TableBody stripes rows by default", () => {
    const element = TableBody({ children: null });

    expect(element.props.className).toContain(
      "[&_tr:nth-child(even)]:bg-control-bg/50"
    );
  });

  test("TableBody can disable striping", () => {
    const element = TableBody({ children: null, striped: false });

    expect(element.props.className).not.toContain(
      "[&_tr:nth-child(even)]:bg-control-bg/50"
    );
  });

  test("TableRow can opt out of striping", () => {
    const element = TableRow({ children: null, striped: false });

    expect(element.props["data-striped"]).toBe("false");
    expect(element.props.className).toContain("!bg-transparent");
  });
});
