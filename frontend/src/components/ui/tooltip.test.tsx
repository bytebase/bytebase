import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { BlockTooltip, Tooltip } from "./tooltip";

const { recordTooltipProvider } = vi.hoisted(() => ({
  recordTooltipProvider: vi.fn(),
}));

vi.mock("@base-ui/react/tooltip", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@base-ui/react/tooltip")>();
  const { createElement } = await import("react");

  function Provider(
    props: Parameters<typeof actual.Tooltip.Provider>[0]
  ) {
    recordTooltipProvider(props);
    return createElement(actual.Tooltip.Provider, props);
  }

  return {
    Tooltip: {
      ...actual.Tooltip,
      Provider,
    },
  };
});

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mount = (ui: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => root.render(ui));
  return { container, root };
};

describe("Tooltip", () => {
  afterEach(() => {
    recordTooltipProvider.mockClear();
    vi.useRealTimers();
    document.body.innerHTML = "";
  });

  test.each([
    ["with a body", <span key="body">a deadline</span>],
    ["with nothing to say", undefined],
  ])("keeps the caller's trigger element %s", (_name, content) => {
    // The element carries the caller's layout, so a row must not gain or lose
    // it with the tooltip's content.
    const { container, root } = mount(
        <Tooltip content={content} render={<span className="shrink-0" />}>
          12:00
        </Tooltip>
    );

    const trigger = container.querySelector(".shrink-0");
    expect(trigger).not.toBeNull();
    expect(trigger?.textContent).toBe("12:00");
    act(() => root.unmount());
  });

  test("blocks trigger a block, and add nothing when silent", () => {
    // The block wrapper is this component's own default rather than layout a
    // caller asked for, so a field with no tooltip is left as it was.
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() =>
      root.render(<BlockTooltip content={<span>why</span>}>a field</BlockTooltip>)
    );
    expect(container.querySelector(".flex-1")).not.toBeNull();

    act(() => root.render(<BlockTooltip content={undefined}>a field</BlockTooltip>));
    expect(container.querySelector(".flex-1")).toBeNull();
    expect(container.textContent).toBe("a field");
    act(() => root.unmount());
  });

  test.each([
    ["an empty string", ""],
    ["nothing", undefined],
  ])("opens nothing when its body is %s", async (_name, content) => {
    // Callers pass strings that can be empty -- a disabled reason, an option's
    // hint -- and an empty one is a tooltip with nothing to open.
    vi.useFakeTimers();
    const { container, root } = mount(
        <Tooltip content={content}>
          <button type="button">Trigger</button>
        </Tooltip>
    );

    await act(async () => {
      container
        .querySelector("button")
        ?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
    });

    // Not the text -- an empty body renders an empty popup, which reads the
    // same. What must not exist is the popup.
    expect(
      document.getElementById("bb-react-layer-overlay")?.childElementCount ?? 0
    ).toBe(0);
    act(() => root.unmount());
  });

  test("mounts tooltip content into the overlay layer root", async () => {
    vi.useFakeTimers();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <Tooltip content="Tip content" popupClassName="max-w-96">
          <button type="button">Trigger</button>
        </Tooltip>
      );
    });

    const trigger = container.querySelector("button");
    expect(trigger).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      trigger?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
    });

    const overlayRoot = document.getElementById("bb-react-layer-overlay");
    expect(overlayRoot).toBeInstanceOf(HTMLDivElement);
    expect(overlayRoot?.textContent).toContain("Tip content");
    expect(overlayRoot?.querySelector(".max-w-96")).toBeInstanceOf(
      HTMLDivElement
    );

    act(() => {
      root.unmount();
    });
  });

  test("does not mount providers for individual tooltips", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <BaseTooltip.Provider>
          <Tooltip content="First tooltip">
            <button type="button">First trigger</button>
          </Tooltip>
          <Tooltip content="Second tooltip">
            <button type="button">Second trigger</button>
          </Tooltip>
        </BaseTooltip.Provider>
      );
    });

    expect(recordTooltipProvider).toHaveBeenCalledTimes(1);

    act(() => {
      root.unmount();
    });
  });

  test("keeps a block tooltip closed until its controlled state opens", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onOpenChange = vi.fn();

    act(() => {
      root.render(
        <BlockTooltip
          content="Truncated query"
          open={false}
          onOpenChange={onOpenChange}
          render={<span className="block truncate" />}
        >
          <button type="button">Query</button>
        </BlockTooltip>
      );
    });

    const trigger = container.querySelector("button");
    expect(trigger).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      trigger?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    });

    expect(onOpenChange).toHaveBeenCalledWith(true, expect.anything());
    expect(document.body.textContent).not.toContain("Truncated query");

    act(() => {
      root.render(
        <BlockTooltip
          content="Truncated query"
          open
          onOpenChange={onOpenChange}
          render={<span className="block truncate" />}
        >
          <button type="button">Query</button>
        </BlockTooltip>
      );
    });

    expect(document.body.textContent).toContain("Truncated query");

    act(() => {
      root.unmount();
    });
  });
});
