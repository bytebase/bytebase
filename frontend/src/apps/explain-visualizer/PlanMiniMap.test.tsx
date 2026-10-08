import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { PlanLayout } from "./plan-layout";
import { PlanMiniMap } from "./PlanMiniMap";
import type { PlanNode } from "./plan-model";

const node = (id: string): PlanNode => ({
  id,
  nodeType: "Seq Scan",
  properties: [],
  warnings: [],
  children: [],
});

// An 800x400 plan with a card at the origin and another to its lower right.
const layout: PlanLayout = {
  nodes: [
    { node: node("0"), x: 0, y: 0 },
    { node: node("0.0"), x: 584, y: 304 },
  ],
  edges: [],
  width: 800,
  height: 400,
};

const SIZE = { width: 400, height: 300 };

const renderMap = (
  overrides: Partial<Parameters<typeof PlanMiniMap>[0]> = {}
) => {
  const onPanTo = vi.fn();
  const props = {
    layout,
    size: SIZE,
    view: { scale: 1, x: 0, y: 0 },
    selectedId: "0",
    onPanTo,
    ...overrides,
  };
  return { onPanTo, ...render(<PlanMiniMap {...props} />) };
};

const surface = () => screen.getByTestId("plan-mini-map");
const frame = () => screen.getByTestId("plan-mini-map-viewport");

describe("PlanMiniMap", () => {
  test("draws the whole plan scaled into its corner", () => {
    renderMap();

    // 160/800 binds, so the plan is drawn at a fifth of its size.
    expect(surface().style.width).toBe("160px");
    expect(surface().style.height).toBe("80px");
    expect(screen.getAllByTestId("plan-mini-map-node")).toHaveLength(2);
  });

  test("places each node where it sits in the plan", () => {
    renderMap();
    const dots = screen.getAllByTestId("plan-mini-map-node");

    const px = (value: string) => Number.parseFloat(value);
    expect(px(dots[1].style.left)).toBeCloseTo(584 / 5, 6);
    expect(px(dots[1].style.top)).toBeCloseTo(304 / 5, 6);
    // A 216x96 card at a fifth of its size.
    expect(px(dots[1].style.width)).toBeCloseTo(43.2, 6);
  });

  test("marks where the selected node sits", () => {
    renderMap({ selectedId: "0.0" });
    const dots = screen.getAllByTestId("plan-mini-map-node");

    expect(dots[0]).not.toHaveClass("bg-accent");
    expect(dots[1]).toHaveClass("bg-accent");
  });

  test("frames the part of the plan currently on screen", () => {
    renderMap({ view: { scale: 1, x: -200, y: -50 } });

    expect(frame().style.left).toBe("40px");
    expect(frame().style.top).toBe("10px");
    expect(frame().style.width).toBe("80px");
    expect(frame().style.height).toBe("60px");
  });

  test("stays out of the accessibility tree", () => {
    renderMap();
    expect(surface()).toHaveAttribute("aria-hidden", "true");
    expect(screen.queryAllByRole("button")).toHaveLength(0);
  });

  test("asks for the point clicked to be brought to the middle", () => {
    const { onPanTo } = renderMap();

    fireEvent.pointerDown(surface(), { button: 0, clientX: 80, clientY: 40 });

    // A fifth-size map, so 80,40 on it is 400,200 in the plan.
    expect(onPanTo).toHaveBeenCalledWith({ x: 400, y: 200 });
  });

  test("keeps following a drag after the press", () => {
    const { onPanTo } = renderMap();

    fireEvent.pointerDown(surface(), { button: 0, clientX: 80, clientY: 40 });
    fireEvent.pointerMove(window, { clientX: 120, clientY: 60 });
    expect(onPanTo).toHaveBeenLastCalledWith({ x: 600, y: 300 });

    fireEvent.pointerUp(window);
    fireEvent.pointerMove(window, { clientX: 20, clientY: 20 });
    expect(onPanTo).toHaveBeenCalledTimes(2);
  });

  test("leaves the diagram's own pan alone while it is being dragged", () => {
    // The diagram pans on a press anywhere on its canvas, so a press aimed at
    // the mini-map must not reach it.
    const onCanvasPointerDown = vi.fn();
    const onPanTo = vi.fn();
    render(
      <div onPointerDown={onCanvasPointerDown}>
        <PlanMiniMap
          layout={layout}
          size={SIZE}
          view={{ scale: 1, x: 0, y: 0 }}
          selectedId="0"
          onPanTo={onPanTo}
        />
      </div>
    );

    fireEvent.pointerDown(surface(), { button: 0, clientX: 10, clientY: 10 });

    expect(onPanTo).toHaveBeenCalled();
    expect(onCanvasPointerDown).not.toHaveBeenCalled();
  });

  test("ignores a press that is not the primary button", () => {
    const { onPanTo } = renderMap();

    fireEvent.pointerDown(surface(), { button: 2, clientX: 80, clientY: 40 });
    expect(onPanTo).not.toHaveBeenCalled();
  });

  test("renders nothing for a plan with no extent to overview", () => {
    renderMap({ layout: { ...layout, width: 0, height: 0 } });
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();
  });
});
