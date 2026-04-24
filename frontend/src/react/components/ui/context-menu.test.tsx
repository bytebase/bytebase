import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "./context-menu";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Base UI's ContextMenu uses ResizeObserver internally.
globalThis.ResizeObserver ??= class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

describe("ContextMenu", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("right-clicking the trigger opens the content", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <ContextMenu>
          <ContextMenuTrigger>
            <div data-testid="trigger">Right-click me</div>
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuItem>Action</ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      );
    });

    const trigger = container.querySelector("[data-testid='trigger']");
    expect(trigger).toBeInstanceOf(HTMLDivElement);

    await act(async () => {
      const event = new MouseEvent("contextmenu", {
        bubbles: true,
        cancelable: true,
        clientX: 50,
        clientY: 50,
      });
      trigger?.dispatchEvent(event);
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Action");
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("clicking an item fires its onClick", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onClick = vi.fn();

    await act(async () => {
      root.render(
        <ContextMenu>
          <ContextMenuTrigger>
            <div data-testid="trigger">Right-click me</div>
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuItem onClick={onClick}>Clickable</ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      );
    });

    const trigger = container.querySelector("[data-testid='trigger']");

    // Open the menu first
    await act(async () => {
      const event = new MouseEvent("contextmenu", {
        bubbles: true,
        cancelable: true,
        clientX: 50,
        clientY: 50,
      });
      trigger?.dispatchEvent(event);
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Clickable");
    });

    // Click the item
    await act(async () => {
      const item = Array.from(
        document.querySelectorAll("[role='menuitem']")
      ).find((el) => el.textContent === "Clickable");
      item?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(onClick).toHaveBeenCalledTimes(1);

    await act(async () => {
      root.unmount();
    });
  });

  test("ESC key closes the menu", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <ContextMenu>
          <ContextMenuTrigger>
            <div data-testid="trigger">Right-click me</div>
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuItem>Action</ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      );
    });

    const trigger = container.querySelector("[data-testid='trigger']");

    // Open the menu
    await act(async () => {
      const event = new MouseEvent("contextmenu", {
        bubbles: true,
        cancelable: true,
        clientX: 50,
        clientY: 50,
      });
      trigger?.dispatchEvent(event);
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Action");
    });

    // Press ESC
    await act(async () => {
      document.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
      );
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).not.toContain("Action");
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("disabled items do not fire onClick when clicked", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const onClick = vi.fn();

    await act(async () => {
      root.render(
        <ContextMenu>
          <ContextMenuTrigger>
            <div data-testid="trigger">Right-click me</div>
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuItem disabled onClick={onClick}>
              Disabled action
            </ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      );
    });

    const trigger = container.querySelector("[data-testid='trigger']");

    // Open the menu
    await act(async () => {
      const event = new MouseEvent("contextmenu", {
        bubbles: true,
        cancelable: true,
        clientX: 50,
        clientY: 50,
      });
      trigger?.dispatchEvent(event);
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Disabled action");
    });

    // Attempt to click the disabled item
    await act(async () => {
      const item = Array.from(
        document.querySelectorAll("[role='menuitem']")
      ).find((el) => el.textContent === "Disabled action");
      item?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(onClick).not.toHaveBeenCalled();

    await act(async () => {
      root.unmount();
    });
  });

  test("renders separator and label without errors", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <ContextMenu>
          <ContextMenuTrigger>
            <div data-testid="trigger">Right-click me</div>
          </ContextMenuTrigger>
          <ContextMenuContent>
            <ContextMenuLabel>Section</ContextMenuLabel>
            <ContextMenuItem>Item one</ContextMenuItem>
            <ContextMenuSeparator />
            <ContextMenuItem>Item two</ContextMenuItem>
          </ContextMenuContent>
        </ContextMenu>
      );
    });

    const trigger = container.querySelector("[data-testid='trigger']");

    await act(async () => {
      const event = new MouseEvent("contextmenu", {
        bubbles: true,
        cancelable: true,
        clientX: 50,
        clientY: 50,
      });
      trigger?.dispatchEvent(event);
    });

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain("Section");
      expect(document.body.textContent).toContain("Item one");
      expect(document.body.textContent).toContain("Item two");
    });

    await act(async () => {
      root.unmount();
    });
  });
});
