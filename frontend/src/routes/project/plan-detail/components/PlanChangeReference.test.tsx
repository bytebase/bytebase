import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PlanChangeReference as PlanChangeReferenceModel } from "../utils/changeReference";
import {
  PlanChangeReference,
  PlanChangeReferenceTooltip,
} from "./PlanChangeReference";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let resizeObserverCallback: ResizeObserverCallback | undefined;

class ResizeObserverStub implements ResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    resizeObserverCallback = callback;
  }
  disconnect() {}
  observe() {}
  takeRecords(): ResizeObserverEntry[] {
    return [];
  }
  unobserve() {}
}

describe("PlanChangeReference", () => {
  let container: HTMLDivElement;
  let root: Root;
  let clientWidthSpy: {
    mockRestore: () => void;
    mockReturnValue: (value: number) => unknown;
  };
  let scrollWidthSpy: {
    mockRestore: () => void;
    mockReturnValue: (value: number) => unknown;
  };

  beforeEach(() => {
    globalThis.ResizeObserver = ResizeObserverStub;
    resizeObserverCallback = undefined;
    clientWidthSpy = vi
      .spyOn(HTMLElement.prototype, "clientWidth", "get")
      .mockReturnValue(280);
    scrollWidthSpy = vi
      .spyOn(HTMLElement.prototype, "scrollWidth", "get")
      .mockReturnValue(120);
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    clientWidthSpy.mockRestore();
    scrollWidthSpy.mockRestore();
    document.body.removeChild(container);
  });

  it("switches a long database list to its count label", async () => {
    clientWidthSpy.mockReturnValue(120);
    scrollWidthSpy.mockReturnValue(280);
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 1: orders, customers, audit",
      countLabel: "3 databases · Production",
      fullLabel: "orders, customers, audit",
      icon: "database",
      index: 1,
      label: "orders, customers, audit",
      showIndex: false,
      showIndexWithCountLabel: true,
      specId: "spec-1",
    };

    await act(async () => {
      root.render(<PlanChangeReference reference={reference} />);
      await Promise.resolve();
    });

    const element = container.querySelector("[data-plan-change-reference]");
    expect(element?.getAttribute("data-plan-change-reference-label")).toBe(
      "3 databases · Production"
    );
    expect(element).toHaveAttribute(
      "data-plan-change-reference-overflow",
      "true"
    );
    expect(
      element?.children[0]?.textContent
    ).toBe("1");
  });

  it("restores the full label when its available width increases", async () => {
    clientWidthSpy.mockReturnValue(120);
    scrollWidthSpy.mockReturnValue(280);
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 1: orders, customers, audit",
      countLabel: "3 databases",
      fullLabel: "orders, customers, audit",
      icon: "database",
      index: 1,
      label: "orders, customers, audit",
      showIndex: false,
      showIndexWithCountLabel: false,
      specId: "spec-1",
    };

    await act(async () => {
      root.render(<PlanChangeReference reference={reference} />);
      await Promise.resolve();
    });
    expect(
      container
        .querySelector("[data-plan-change-reference]")
        ?.getAttribute("data-plan-change-reference-label")
    ).toBe("3 databases");

    clientWidthSpy.mockReturnValue(280);
    scrollWidthSpy.mockReturnValue(200);
    await act(async () => {
      resizeObserverCallback?.([], {} as ResizeObserver);
      await Promise.resolve();
    });

    const element = container.querySelector("[data-plan-change-reference]");
    expect(element?.getAttribute("data-plan-change-reference-label")).toBe(
      "orders, customers, audit"
    );
    expect(element).not.toHaveAttribute("data-plan-change-reference-overflow");
  });

  it("remeasures the full label when an overflowing reference changes", async () => {
    clientWidthSpy.mockReturnValue(120);
    scrollWidthSpy.mockReturnValue(280);
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 1: orders, customers, audit",
      countLabel: "3 databases",
      fullLabel: "orders, customers, audit",
      icon: "database",
      index: 1,
      label: "orders, customers, audit",
      showIndex: false,
      showIndexWithCountLabel: true,
      specId: "spec-1",
    };

    await act(async () => {
      root.render(<PlanChangeReference reference={reference} />);
      await Promise.resolve();
    });
    expect(
      container
        .querySelector("[data-plan-change-reference]")
        ?.getAttribute("data-plan-change-reference-label")
    ).toBe("3 databases");

    clientWidthSpy.mockReturnValue(180);
    scrollWidthSpy.mockReturnValue(100);
    await act(async () => {
      root.render(
        <PlanChangeReference
          reference={{
            ...reference,
            accessibleLabel: "Change 1: orders, audit",
            countLabel: "2 databases",
            fullLabel: "orders, audit",
            label: "orders, audit",
          }}
        />
      );
      await Promise.resolve();
    });

    const element = container.querySelector("[data-plan-change-reference]");
    expect(element?.getAttribute("data-plan-change-reference-label")).toBe(
      "orders, audit"
    );
    expect(element).not.toHaveAttribute("data-plan-change-reference-overflow");
  });

  it("omits the index and tooltip when the reference fully fits", async () => {
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 1: orders",
      fullLabel: "orders",
      icon: "database",
      index: 1,
      label: "orders",
      showIndex: false,
      showIndexWithCountLabel: false,
      specId: "spec-1",
    };

    await act(async () => {
      root.render(<PlanChangeReference reference={reference} />);
      await Promise.resolve();
    });

    const element = container.querySelector("[data-plan-change-reference]");
    expect(element).toHaveClass("max-w-80");
    expect(element?.children[0]?.querySelector("svg")).not.toBeNull();
    expect(element?.children[1]).toHaveClass("max-w-72");
    expect(element?.children[1]?.textContent).toBe("orders");
    expect(element?.textContent).toBe("orders");
    expect(element?.getAttribute("aria-label")).toBe("Change 1: orders");
    expect(element).not.toHaveAttribute("data-plan-change-reference-overflow");
    const measurement = container.querySelector(
      "[data-plan-change-reference-measure]"
    );
    expect(measurement).toHaveClass("invisible", "overflow-hidden");
    expect(measurement).not.toHaveClass("absolute");
  });

  it("renders a subdued index when references collide", async () => {
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 2: orders",
      fullLabel: "orders",
      icon: "database",
      index: 2,
      label: "orders",
      showIndex: true,
      showIndexWithCountLabel: false,
      specId: "spec-2",
    };

    await act(async () => {
      root.render(<PlanChangeReference reference={reference} />);
      await Promise.resolve();
    });

    const element = container.querySelector("[data-plan-change-reference]");
    expect(element?.children[0]?.textContent).toBe("2");
    expect(element?.children[0]?.className).toContain("text-sm");
    expect(element?.children[0]?.className).toContain(
      "text-control-placeholder"
    );
    expect(element?.children[0]?.className).toContain("tabular-nums");
    expect(element?.children[1]?.querySelector("svg")).not.toBeNull();
    expect(element?.children[2]?.textContent).toBe("orders");
  });

  it("uses the compact width limit in the plan tab", async () => {
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 1: customer_orders_production_archive",
      fullLabel: "customer_orders_production_archive",
      icon: "database",
      index: 1,
      label: "customer_orders_production_archive",
      showIndex: false,
      showIndexWithCountLabel: false,
      specId: "spec-1",
    };

    await act(async () => {
      root.render(
        <PlanChangeReference density="tab" reference={reference} />
      );
      await Promise.resolve();
    });

    const element = container.querySelector("[data-plan-change-reference]");
    expect(element).toHaveClass("max-w-64");
    expect(element?.children[1]).toHaveClass("max-w-56");
    expect(element?.textContent).toBe("customer_orders_production_archive");
  });

  it("shows the complete label and summary in the structured tooltip", async () => {
    const reference: PlanChangeReferenceModel = {
      accessibleLabel: "Change 4: orders, customers, audit",
      countLabel: "3 databases · Production",
      fullLabel: "orders, customers, audit",
      icon: "database",
      index: 4,
      label: "orders, customers, audit",
      showIndex: false,
      showIndexWithCountLabel: false,
      specId: "spec-4",
    };

    await act(async () => {
      root.render(<PlanChangeReferenceTooltip reference={reference} />);
      await Promise.resolve();
    });

    expect(container.textContent).toContain("Change 4");
    expect(container.textContent).toContain("orders, customers, audit");
    expect(container.textContent).toContain("3 databases · Production");
    expect(container.querySelector('[dir="auto"]')?.textContent).toBe(
      reference.fullLabel
    );
  });
});
