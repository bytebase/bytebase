import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { StepIndicator } from "./step-indicator";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

describe("StepIndicator", () => {
  test("fills only the current step with the accent; a completed step has a tinted check and a plain title", () => {
    const { container, unmount } = renderIntoContainer(
      <StepIndicator
        currentIndex={1}
        steps={[
          { title: "First" },
          { title: "Second" },
          { title: "Third" },
        ]}
      />
    );

    const labels = [...container.querySelectorAll("li span")];
    expect(labels.map((label) => label.textContent)).toEqual([
      "First",
      "Second",
      "Third",
    ]);
    expect(labels[0]).not.toHaveClass("text-accent");
    expect(labels[0]).toHaveClass("text-control");
    expect(labels[0]).not.toHaveClass("font-medium");
    expect(labels[1]).toHaveClass("text-accent", "font-medium");
    expect(labels[2]).toHaveClass("text-control-light");
    expect(labels[2]).not.toHaveClass("font-medium");
    const circles = labels.map((label) => label.previousElementSibling);
    expect(circles[0]).toHaveClass("bg-accent/10", "text-accent");
    expect(circles[0]).not.toHaveClass("bg-accent");
    expect(circles[1]).toHaveClass("bg-accent", "text-accent-text");
    expect(circles[2]).toHaveClass("bg-control-bg-hover", "text-control-light");
    expect(container.querySelector("svg")).toBeTruthy();
    // A completed step shows a check, which carries the number it replaced.
    expect(container.textContent).not.toContain("1");
    expect(container.querySelector("svg")?.getAttribute("aria-label")).toBe(
      "1"
    );
    expect(
      [...container.querySelectorAll("li")].map((item) =>
        item.getAttribute("aria-current")
      )
    ).toEqual([null, "step", null]);
    expect(container.textContent).toContain("2");
    expect(container.textContent).toContain("3");
    expect(container.querySelectorAll(".bg-control-border")).toHaveLength(2);

    unmount();
  });

  describe("with fitOneRow", () => {
    // jsdom lays nothing out, so the widths are given: the one row the steps
    // need, and the list they sit in.
    const withWidths = (row: number, list: number, run: () => void) => {
      const observer = globalThis.ResizeObserver;
      globalThis.ResizeObserver = class {
        constructor(private readonly callback: () => void) {}
        observe() {
          this.callback();
        }
        disconnect() {}
        unobserve() {}
      } as unknown as typeof ResizeObserver;
      const scroll = vi
        .spyOn(HTMLElement.prototype, "scrollWidth", "get")
        .mockReturnValue(row);
      const client = vi
        .spyOn(HTMLElement.prototype, "clientWidth", "get")
        .mockReturnValue(list);
      try {
        run();
      } finally {
        scroll.mockRestore();
        client.mockRestore();
        globalThis.ResizeObserver = observer;
      }
    };
    const titlesIn = (container: HTMLElement) =>
      [
        ...container.querySelectorAll(
          '[data-slot="step-indicator"] li span'
        ),
      ].map((title) => ({
        text: title.textContent,
        shown: !title.className.includes("sr-only"),
      }));

    test("keeps only the current step's title in sight when the steps do not fit one row", () => {
      withWidths(500, 300, () => {
        const { container, unmount } = renderIntoContainer(
          <StepIndicator
            fitOneRow
            currentIndex={1}
            steps={[
              { title: "First" },
              { title: "Second" },
              { title: "Third" },
            ]}
          />
        );

        expect(titlesIn(container)).toEqual([
          { text: "First", shown: false },
          { text: "Second", shown: true },
          { text: "Third", shown: false },
        ]);
        unmount();
      });
    });

    test("measures the same row on every step, so changing step never switches between full and compact", () => {
      const steps = [{ title: "First" }, { title: "Second" }, { title: "Third" }];
      const measured = (currentIndex: number) => {
        const { container, unmount } = renderIntoContainer(
          <StepIndicator fitOneRow currentIndex={currentIndex} steps={steps} />
        );
        // The out-of-sight row whose width decides the mode.
        const titles = [
          ...container.querySelectorAll('ol[aria-hidden="true"] li span'),
        ].map((title) => ({
          text: title.textContent,
          bold: title.classList.contains("font-medium"),
        }));
        unmount();
        return titles;
      };

      expect(measured(0)).toEqual(measured(2));
      expect(measured(1).every(({ bold }) => bold)).toBe(true);
    });

    test("keeps every title in sight when the steps fit one row", () => {
      withWidths(500, 600, () => {
        const { container, unmount } = renderIntoContainer(
          <StepIndicator
            fitOneRow
            currentIndex={1}
            steps={[
              { title: "First" },
              { title: "Second" },
              { title: "Third" },
            ]}
          />
        );

        expect(titlesIn(container).every(({ shown }) => shown)).toBe(true);
        unmount();
      });
    });
  });

  describe("vertical", () => {
    test("shows each description and marks finished steps as success", () => {
      const { container, unmount } = renderIntoContainer(
        <StepIndicator
          orientation="vertical"
          currentIndex={1}
          steps={[
            { title: "First", description: "Pick one" },
            { title: "Second", description: "Fill in" },
          ]}
        />
      );

      const items = container.querySelectorAll("li");
      expect(items[0].textContent).toContain("Pick one");
      expect(items[1].textContent).toContain("Fill in");
      expect(items[0].querySelector(".bg-success")).toBeTruthy();
      expect(items[1].querySelector('[aria-current="step"]')).toBeTruthy();
      unmount();
    });
  });

  test("rail row shows only the current title and marks finished steps", () => {
    const { container, unmount } = renderIntoContainer(
      <StepIndicator
        variant="rail"
        currentIndex={1}
        steps={[{ title: "First" }, { title: "Second" }, { title: "Third" }]}
      />
    );

    const titles = Array.from(container.querySelectorAll("li span")).map(
      (span) => ({
        text: span.textContent,
        shown: !span.classList.contains("sr-only"),
      })
    );
    expect(titles).toEqual([
      { text: "First", shown: false },
      { text: "Second", shown: true },
      { text: "Third", shown: false },
    ]);
    expect(
      container.querySelectorAll("li")[0].querySelector(".bg-success")
    ).toBeTruthy();
    unmount();
  });

  test("moves to a step when it is clicked", () => {
    const onStepClick = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <StepIndicator
        orientation="vertical"
        currentIndex={0}
        onStepClick={onStepClick}
        steps={[{ title: "First" }, { title: "Second" }]}
      />
    );

    act(() => {
      container.querySelectorAll("button")[1].click();
    });
    expect(onStepClick).toHaveBeenCalledWith(1);
    unmount();
  });
});
