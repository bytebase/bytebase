import { createRef } from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  QueryResultVerticalBody,
  type QueryResultVerticalBodyHandle,
} from "./QueryResultVerticalBody";

const { scrollToIndex } = vi.hoisted(() => ({ scrollToIndex: vi.fn() }));

vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: (options: {
    count: number;
    estimateSize: (index: number) => number;
    gap?: number;
    paddingEnd?: number;
  }) => ({
    getVirtualItems: () =>
      Array.from({ length: options.count }, (_, index) => ({
        index,
        key: index,
        start: index * (options.estimateSize(index) + (options.gap ?? 0)),
      })),
    getTotalSize: () =>
      options.count * options.estimateSize(0) +
      Math.max(0, options.count - 1) * (options.gap ?? 0) +
      (options.paddingEnd ?? 0),
    measureElement: vi.fn(),
    scrollToIndex,
  }),
}));

describe("QueryResultVerticalBody", () => {
  test("renders editor-style row headings around caller-supplied values", () => {
    const { container } = render(
      <QueryResultVerticalBody
        rowCount={2}
        columnCount={3}
        renderRow={(rowIndex) => <div>row value {rowIndex + 1}</div>}
      />
    );

    const body = container.querySelector("[data-query-result-vertical-body]");
    expect(body).toHaveClass("overflow-auto", "border-block-border");
    expect(
      container.querySelectorAll("[data-query-result-vertical-row]")
    ).toHaveLength(2);
    expect(screen.getByText(/1\. row/)).toBeInTheDocument();
    expect(screen.getByText(/2\. row/)).toBeInTheDocument();
    expect(screen.getByText("row value 1")).toBeInTheDocument();
    expect(screen.getByText("row value 2")).toBeInTheDocument();
  });

  test("exposes scrolling to a result row", () => {
    const ref = createRef<QueryResultVerticalBodyHandle>();
    render(
      <QueryResultVerticalBody
        ref={ref}
        rowCount={3}
        columnCount={1}
        renderRow={(rowIndex) => <div>value {rowIndex + 1}</div>}
      />
    );

    ref.current?.scrollTo(2);
    expect(scrollToIndex).toHaveBeenCalledWith(2, {
      align: "start",
      behavior: "smooth",
    });
  });
});
