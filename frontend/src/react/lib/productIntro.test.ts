import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

const appStoreMocks = vi.hoisted(() => {
  const introState = new Map<string, boolean>();
  return {
    introState,
    getIntroStateByKey: vi.fn((key: string) => introState.get(key) ?? false),
    saveIntroStateByKey: vi.fn(
      ({ key, newState }: { key: string; newState: boolean }) => {
        introState.set(key, newState);
      }
    ),
  };
});

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      getIntroStateByKey: appStoreMocks.getIntroStateByKey,
      saveIntroStateByKey: appStoreMocks.saveIntroStateByKey,
    }),
  },
}));

import { showProductIntroOnce } from "./productIntro";

const setRect = (element: HTMLElement, rect: Partial<DOMRect>) => {
  element.getBoundingClientRect = vi.fn(
    () =>
      ({
        x: rect.left ?? 0,
        y: rect.top ?? 0,
        top: rect.top ?? 0,
        left: rect.left ?? 0,
        right: rect.right ?? 0,
        bottom: rect.bottom ?? 0,
        width: rect.width ?? 0,
        height: rect.height ?? 0,
      }) as DOMRect
  );
};

const createButton = (rect: Partial<DOMRect>) => {
  const button = document.createElement("button");
  button.setAttribute("data-product-intro-target", "connect-database");
  setRect(button, rect);
  document.body.appendChild(button);
  return button;
};

const introOptions = {
  id: "connect-database",
  title: "Connect your first database",
  description: "Connect a database.",
  closeLabel: "Close",
};

const libDir = dirname(fileURLToPath(import.meta.url));

describe("showProductIntroOnce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    appStoreMocks.introState.clear();
    appStoreMocks.getIntroStateByKey.mockClear();
    appStoreMocks.saveIntroStateByKey.mockClear();
    window.localStorage.clear();
    document.body.innerHTML = "";
  });

  afterEach(() => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    window.localStorage.clear();
    document.body.innerHTML = "";
    vi.useRealTimers();
  });

  test("renders a custom intro around the visible target", async () => {
    const button = createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await expect(showProductIntroOnce(introOptions)).resolves.toBe(true);

    expect(document.querySelectorAll(".bb-product-intro")).toHaveLength(1);
    expect(button.classList.contains("bb-product-intro-active")).toBe(true);
    expect(document.querySelector(".bb-product-intro-title")?.textContent).toBe(
      "Connect your first database"
    );
    expect(
      document.querySelector(".bb-product-intro-description")?.textContent
    ).toBe("Connect a database.");
    expect(document.querySelector(".bb-product-intro-stage")).toHaveProperty(
      "style.top",
      "234px"
    );
    expect(document.querySelector(".bb-product-intro-popover")).toHaveProperty(
      "style.top",
      "294px"
    );
  });

  test("uses the visible target when responsive layouts render multiple matches", async () => {
    const hiddenButton = createButton({
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      width: 0,
      height: 0,
    });
    const visibleButton = createButton({
      top: 300,
      left: 600,
      right: 760,
      bottom: 340,
      width: 160,
      height: 40,
    });

    await showProductIntroOnce(introOptions);

    expect(hiddenButton.classList.contains("bb-product-intro-active")).toBe(
      false
    );
    expect(visibleButton.classList.contains("bb-product-intro-active")).toBe(
      true
    );
  });

  test("re-resolves and repositions the target after responsive replacement", async () => {
    const oldButton = createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntroOnce(introOptions);

    setRect(oldButton, {
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
      width: 0,
      height: 0,
    });
    oldButton.remove();
    const newButton = createButton({
      top: 300,
      left: 600,
      right: 760,
      bottom: 340,
      width: 160,
      height: 40,
    });

    window.dispatchEvent(new Event("resize"));
    vi.advanceTimersByTime(100);

    expect(document.querySelectorAll(".bb-product-intro")).toHaveLength(1);
    expect(oldButton.classList.contains("bb-product-intro-active")).toBe(false);
    expect(newButton.classList.contains("bb-product-intro-active")).toBe(true);
    expect(document.querySelector(".bb-product-intro-stage")).toHaveProperty(
      "style.top",
      "294px"
    );
    expect(document.querySelector(".bb-product-intro-popover")).toHaveProperty(
      "style.top",
      "354px"
    );
  });

  test("does not show an already dismissed intro", async () => {
    appStoreMocks.introState.set("connect-database", true);
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await expect(showProductIntroOnce(introOptions)).resolves.toBe(false);

    expect(appStoreMocks.getIntroStateByKey).toHaveBeenCalledWith(
      "connect-database"
    );
    expect(document.querySelector(".bb-product-intro")).toBeNull();
  });

  test("marks the intro dismissed when the close button is clicked", async () => {
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntroOnce(introOptions);

    expect(appStoreMocks.saveIntroStateByKey).not.toHaveBeenCalled();
    (
      document.querySelector(".bb-product-intro-close") as HTMLButtonElement
    ).click();

    expect(appStoreMocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "connect-database",
      newState: true,
    });
    expect(document.querySelector(".bb-product-intro")).toBeNull();
  });

  test("destroys the intro when the highlighted target is clicked", async () => {
    const onClick = vi.fn();
    const button = createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });
    button.addEventListener("click", onClick);

    await showProductIntroOnce(introOptions);

    button.click();

    expect(onClick).toHaveBeenCalledTimes(1);
    expect(appStoreMocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "connect-database",
      newState: true,
    });
    expect(document.querySelector(".bb-product-intro")).toBeNull();
  });

  test("keeps the page cover light enough to preserve context", () => {
    const css = readFileSync(join(libDir, "productIntro.css"), "utf8");

    expect(css).toContain("0 0 0 9999px rgb(var(--color-overlay) / 38%)");
    expect(css).not.toContain("0 0 0 9999px rgb(var(--color-overlay) / 62%)");
  });
});
