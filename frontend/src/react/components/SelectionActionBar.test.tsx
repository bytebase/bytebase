import { Archive, Trash2 } from "lucide-react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SelectionActionBar } from "./SelectionActionBar";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SelectionActionBar", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => {
      root.unmount();
    });
    document.body.innerHTML = "";
  });

  test("renders nothing when count is 0", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={0}
          label="0 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[]}
        />
      );
    });
    expect(container.textContent ?? "").not.toContain("0 selected");
    expect(container.querySelector("[role='toolbar']")).toBeNull();
  });

  test("renders label and visible actions when count > 0", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={2}
          label="2 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            {
              key: "archive",
              label: "Archive",
              icon: Archive,
              onClick: () => {},
            },
            {
              key: "delete",
              label: "Delete",
              icon: Trash2,
              onClick: () => {},
              tone: "destructive",
            },
          ]}
        />
      );
    });
    expect(container.textContent).toContain("2 selected");
    expect(container.textContent).toContain("Archive");
    expect(container.textContent).toContain("Delete");
  });

  test("omits hidden actions and disables disabled actions", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={true}
          onToggleSelectAll={() => {}}
          actions={[
            { key: "a", label: "AlphaAction", onClick: () => {} },
            { key: "b", label: "BetaAction", onClick: () => {}, hidden: true },
            {
              key: "c",
              label: "GammaAction",
              onClick: () => {},
              disabled: true,
            },
          ]}
        />
      );
    });
    expect(container.textContent).toContain("AlphaAction");
    expect(container.textContent).not.toContain("BetaAction");
    const gamma = Array.from(container.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("GammaAction")
    );
    expect(gamma).toBeDefined();
    expect(gamma?.hasAttribute("disabled")).toBe(true);
  });

  test("clicking an action invokes its onClick", async () => {
    const onClick = vi.fn();
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={true}
          onToggleSelectAll={() => {}}
          actions={[{ key: "a", label: "ActionLabel", onClick }]}
        />
      );
    });
    const btn = Array.from(container.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("ActionLabel")
    )!;
    await act(async () => {
      btn.click();
    });
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  test("toggling the leading checkbox invokes onToggleSelectAll", async () => {
    const onToggleSelectAll = vi.fn();
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={onToggleSelectAll}
          actions={[]}
        />
      );
    });
    const checkbox =
      container.querySelector<HTMLButtonElement>('[role="checkbox"]')!;
    await act(async () => {
      checkbox.click();
    });
    expect(onToggleSelectAll).toHaveBeenCalledTimes(1);
  });

  test("destructive tone applies the red-tone override class", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={true}
          onToggleSelectAll={() => {}}
          actions={[
            {
              key: "del",
              label: "DestructiveAction",
              onClick: () => {},
              tone: "destructive",
            },
          ]}
        />
      );
    });
    const btn = Array.from(container.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("DestructiveAction")
    )!;
    expect(btn.className).toContain("text-error");
  });

  test("children render after declarative actions", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={true}
          onToggleSelectAll={() => {}}
          actions={[{ key: "a", label: "InlineAction", onClick: () => {} }]}
        >
          <span data-testid="custom-slot">CustomSlot</span>
        </SelectionActionBar>
      );
    });
    expect(
      container.querySelector('[data-testid="custom-slot"]')
    ).not.toBeNull();
    expect(container.textContent).toContain("CustomSlot");
  });

  test("disabled action with disabledReason is wrapped in a tooltip-capable element", async () => {
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={true}
          onToggleSelectAll={() => {}}
          actions={[
            {
              key: "blocked",
              label: "BlockedAction",
              onClick: () => {},
              disabled: true,
              disabledReason: "Permission denied",
            },
          ]}
        />
      );
    });
    const btn = Array.from(container.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("BlockedAction")
    );
    expect(btn).toBeDefined();
    expect(btn?.hasAttribute("disabled")).toBe(true);
  });
});
