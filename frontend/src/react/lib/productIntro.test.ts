import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

vi.mock("@/react/router", () => ({
  router: {
    replace: ({ fullPath }: { fullPath: string }) => {
      window.history.replaceState(window.history.state, "", fullPath);
    },
  },
  useCurrentRoute: () => ({
    query: Object.fromEntries(new URLSearchParams(window.location.search)),
  }),
}));

import { showProductIntro } from "./productIntro";

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
};

const libDir = dirname(fileURLToPath(import.meta.url));

describe("showProductIntro", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    window.history.replaceState({}, "", "/");
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

    await expect(showProductIntro(introOptions)).resolves.toBe(true);

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

    await showProductIntro(introOptions);

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

    await showProductIntro(introOptions);

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

  test("always renders the intro when the target exists", async () => {
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await expect(showProductIntro(introOptions)).resolves.toBe(true);

    expect(document.querySelector(".bb-product-intro")).not.toBeNull();
  });

  test("destroys the intro when the close button is clicked", async () => {
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntro(introOptions);

    (
      document.querySelector(".bb-product-intro-close") as HTMLButtonElement
    ).click();

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

    await showProductIntro(introOptions);

    button.click();

    expect(onClick).toHaveBeenCalledTimes(1);
    expect(document.querySelector(".bb-product-intro")).toBeNull();
  });

  test("destroys the intro when the mask is clicked", async () => {
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntro(introOptions);

    (document.querySelector(".bb-product-intro") as HTMLDivElement).click();

    expect(document.querySelector(".bb-product-intro")).toBeNull();
  });

  test("replaces an existing intro instead of stacking duplicate masks", async () => {
    window.history.replaceState({}, "", "/?intro=connect-database");
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntro(introOptions);
    await showProductIntro(introOptions);

    expect(document.querySelectorAll(".bb-product-intro")).toHaveLength(1);
    expect(window.location.search).toBe("?intro=connect-database");

    (document.querySelector(".bb-product-intro") as HTMLDivElement).click();

    expect(document.querySelector(".bb-product-intro")).toBeNull();
    expect(window.location.search).toBe("");
  });

  test("destroys the intro when the target DOM is removed", async () => {
    window.history.replaceState({}, "", "/?intro=connect-database");
    const button = createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntro(introOptions);

    button.remove();
    await Promise.resolve();

    expect(document.querySelector(".bb-product-intro")).toBeNull();
    expect(button.classList.contains("bb-product-intro-active")).toBe(false);
    expect(window.location.search).toBe("?intro=connect-database");
  });

  test("removes the intro query parameter when the intro is dismissed", async () => {
    window.history.replaceState(
      {},
      "",
      "/settings/general?intro=connect-database&tab=workspace#ai-assistant"
    );
    createButton({
      top: 240,
      left: 400,
      right: 560,
      bottom: 280,
      width: 160,
      height: 40,
    });

    await showProductIntro(introOptions);

    (
      document.querySelector(".bb-product-intro-close") as HTMLButtonElement
    ).click();

    expect(window.location.pathname).toBe("/settings/general");
    expect(window.location.search).toBe("?tab=workspace");
    expect(window.location.hash).toBe("#ai-assistant");
  });

  test("keeps the page cover light enough to preserve context", () => {
    const css = readFileSync(join(libDir, "productIntro.css"), "utf8");

    expect(css).toContain("0 0 0 9999px rgb(var(--color-overlay) / 38%)");
    expect(css).not.toContain("0 0 0 9999px rgb(var(--color-overlay) / 62%)");
  });
});
