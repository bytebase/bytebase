import { afterEach, describe, expect, test, vi } from "vitest";
import type { Router } from "vue-router";
import { lazyExecuteDomAction, lazyExtractDomTree } from "./index";

afterEach(() => {
  document.body.innerHTML = "";
  vi.restoreAllMocks();
});

describe("lazyExecuteDomAction", () => {
  test("looks up actions by ref", async () => {
    document.body.innerHTML = `<input value="hello" />`;
    await lazyExtractDomTree();

    const result = await lazyExecuteDomAction({ type: "read", index: "e1" });

    expect(result).toEqual({ success: true, message: "hello" });
  });

  test("returns refresh guidance for malformed, missing, and stale refs", async () => {
    document.body.innerHTML = `<button>Save</button>`;
    await lazyExtractDomTree();

    await expect(
      lazyExecuteDomAction({ type: "click", index: 1 })
    ).resolves.toEqual({
      success: false,
      message:
        'Invalid element ref: 1. Use refs like [e1] from the DOM tree. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });

    await expect(
      lazyExecuteDomAction({ type: "click", index: "1" })
    ).resolves.toEqual({
      success: false,
      message:
        'Malformed element ref [1]. Use refs like [e1] from the DOM tree. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });

    await expect(
      lazyExecuteDomAction({ type: "click", index: "e9" })
    ).resolves.toEqual({
      success: false,
      message:
        'Element [e9] was not found. Run get_page_state(mode="dom") to refresh the DOM tree.',
    });

    document.querySelector("button")?.remove();
    await expect(
      lazyExecuteDomAction({ type: "click", index: "e1" })
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
      { type: "click", index: "e1" },
      router
    );

    expect(router.push).toHaveBeenCalledWith("/settings?tab=general#profile");
    expect(result).toEqual({
      success: true,
      message: "Navigated to /settings?tab=general#profile",
    });
  });
});
