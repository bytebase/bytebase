import { afterEach, describe, expect, test, vi } from "vitest";
import type { Router } from "vue-router";
import {
  lazyExecuteDomAction,
  lazyExtractDomRefSuggestions,
  lazyExtractDomTree,
} from "./index";

afterEach(() => {
  document.body.innerHTML = "";
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("lazyExtractDomRefSuggestions", () => {
  test("returns structured suggestions from the shared DOM registry", async () => {
    document.body.innerHTML = `<button aria-label="Run query">Run</button>`;

    await expect(lazyExtractDomRefSuggestions()).resolves.toEqual([
      {
        ref: "e1",
        tag: "button",
        role: undefined,
        label: "Run query",
        value: undefined,
      },
    ]);

    await expect(
      lazyExecuteDomAction({ type: "read", ref: "e1" })
    ).resolves.toEqual({ success: true, message: "Run" });
  });
});

describe("lazyExecuteDomAction", () => {
  test("looks up actions by ref", async () => {
    document.body.innerHTML = `<input value="hello" />`;
    await lazyExtractDomTree();

    const result = await lazyExecuteDomAction({ type: "read", ref: "e1" });

    expect(result).toEqual({ success: true, message: "hello" });
  });

  test("clicks pointer-cursor containers via refs without changing ref lookup semantics", async () => {
    vi.stubGlobal(
      "MouseEvent",
      class extends Event {
        constructor(type: string, init?: EventInit) {
          super(type, init);
        }
      }
    );

    const onClick = vi.fn();
    const row = document.createElement("div");
    row.style.cursor = "pointer";
    row.textContent = "Prod Primary";
    row.addEventListener("click", onClick);
    document.body.append(row);

    await expect(lazyExtractDomRefSuggestions()).resolves.toEqual([
      {
        ref: "e1",
        tag: "div",
        role: undefined,
        label: "Prod Primary",
        value: undefined,
      },
    ]);

    await expect(
      lazyExecuteDomAction({ type: "click", ref: "e1" })
    ).resolves.toEqual({ success: true, message: "Clicked div" });
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  test("returns only the top-level ref for inherited pointer-cursor wrappers", async () => {
    document.body.innerHTML = `
      <div style="cursor: pointer">
        <span>Prod Primary</span>
        <div>
          <span>Healthy</span>
        </div>
      </div>
    `;

    await expect(lazyExtractDomRefSuggestions()).resolves.toEqual([
      {
        ref: "e1",
        tag: "div",
        role: undefined,
        label: "Prod Primary Healthy",
        value: undefined,
      },
    ]);
  });

  test("returns refresh guidance for malformed, missing, and stale refs", async () => {
    document.body.innerHTML = `<button>Save</button>`;
    await lazyExtractDomTree();

    await expect(
      lazyExecuteDomAction({ type: "click", ref: 1 } as never)
    ).resolves.toEqual({
      success: false,
      message:
        'Invalid element ref: 1. Use refs like [e1] from the DOM tree. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });

    await expect(
      lazyExecuteDomAction({ type: "click", ref: "1" })
    ).resolves.toEqual({
      success: false,
      message:
        'Malformed element ref [1]. Use refs like [e1] from the DOM tree. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });

    await expect(
      lazyExecuteDomAction({ type: "click", ref: "e9" })
    ).resolves.toEqual({
      success: false,
      message:
        'Element [e9] was not found. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });

    document.querySelector("button")?.remove();
    await expect(
      lazyExecuteDomAction({ type: "click", ref: "e1" })
    ).resolves.toEqual({
      success: false,
      message:
        'Element [e1] is stale. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });
  });

  test("uses vue router for same-origin link clicks", async () => {
    window.history.pushState({}, "", "/current");
    document.body.innerHTML =
      '<a href="/settings?tab=general#profile">Settings</a>';
    await lazyExtractDomTree();

    const router = {
      push: vi.fn().mockResolvedValue(undefined),
    } as unknown as Router;

    const result = await lazyExecuteDomAction(
      { type: "click", ref: "e1" },
      router
    );

    expect(router.push).toHaveBeenCalledWith("/settings?tab=general#profile");
    expect(result).toEqual({
      success: true,
      message: "Navigated to /settings?tab=general#profile",
    });
  });
});
