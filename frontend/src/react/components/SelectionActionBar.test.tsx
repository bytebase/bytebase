import { Archive, Trash2 } from "lucide-react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SelectionActionBar } from "./SelectionActionBar";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));

function mockMatchMedia(matches: (query: string) => boolean) {
  window.matchMedia = ((query: string) => ({
    matches: matches(query),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  })) as unknown as typeof window.matchMedia;
}

describe("SelectionActionBar", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    // Default to the widest tier so existing tests see all actions inline.
    mockMatchMedia(() => true);
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

  test("at lg breakpoint, shows first 5 actions inline and rest in More menu", async () => {
    mockMatchMedia(() => true);
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            { key: "a1", label: "Action1", icon: Archive, onClick: () => {} },
            { key: "a2", label: "Action2", icon: Archive, onClick: () => {} },
            { key: "a3", label: "Action3", icon: Archive, onClick: () => {} },
            { key: "a4", label: "Action4", icon: Archive, onClick: () => {} },
            { key: "a5", label: "Action5", icon: Archive, onClick: () => {} },
            { key: "a6", label: "Action6", icon: Archive, onClick: () => {} },
            { key: "a7", label: "Action7", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    for (const label of [
      "Action1",
      "Action2",
      "Action3",
      "Action4",
      "Action5",
    ]) {
      expect(container.textContent ?? "").toContain(label);
    }
    expect(container.textContent ?? "").not.toContain("Action6");
    expect(container.textContent ?? "").not.toContain("Action7");
    const moreButton = container.querySelector("button[aria-label]");
    expect(moreButton).not.toBeNull();
  });

  test("at < sm breakpoint, shows only 1 action inline and rest in More menu", async () => {
    mockMatchMedia(() => false);
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={2}
          label="2 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            {
              key: "a1",
              label: "InlineOnly",
              icon: Archive,
              onClick: () => {},
            },
            { key: "a2", label: "Overflow1", icon: Archive, onClick: () => {} },
            { key: "a3", label: "Overflow2", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    expect(container.textContent ?? "").toContain("InlineOnly");
    expect(container.textContent ?? "").not.toContain("Overflow1");
    expect(container.textContent ?? "").not.toContain("Overflow2");
  });

  test("hidden actions don't count toward maxVisible", async () => {
    mockMatchMedia(() => false); // < sm → maxVisible = 1
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          actions={[
            {
              key: "h",
              label: "HiddenA",
              icon: Archive,
              onClick: () => {},
              hidden: true,
            },
            { key: "v", label: "VisibleA", icon: Archive, onClick: () => {} },
          ]}
        />
      );
    });
    expect(container.textContent ?? "").toContain("VisibleA");
    expect(container.textContent ?? "").not.toContain("HiddenA");
    const moreButton = container.querySelector("button[aria-label]");
    expect(moreButton).toBeNull();
  });

  test("maxVisibleActions override skips the More menu entirely", async () => {
    mockMatchMedia(() => false);
    await act(async () => {
      root.render(
        <SelectionActionBar
          count={1}
          label="1 selected"
          allSelected={false}
          onToggleSelectAll={() => {}}
          maxVisibleActions={99}
          actions={[
            {
              key: "a1",
              label: "ActionAlpha",
              icon: Archive,
              onClick: () => {},
            },
            {
              key: "a2",
              label: "ActionBeta",
              icon: Archive,
              onClick: () => {},
            },
            {
              key: "a3",
              label: "ActionGamma",
              icon: Archive,
              onClick: () => {},
            },
          ]}
        />
      );
    });
    expect(container.textContent ?? "").toContain("ActionAlpha");
    expect(container.textContent ?? "").toContain("ActionBeta");
    expect(container.textContent ?? "").toContain("ActionGamma");
    const moreButton = container.querySelector("button[aria-label]");
    expect(moreButton).toBeNull();
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
