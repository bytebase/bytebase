import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { AgentMinimizedLauncher } from "./AgentMinimizedLauncher";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: () => "Agent" }),
}));

let resizeCallbacks: (() => void)[];
let disconnects: ReturnType<typeof vi.fn>[];
let rects: Map<Element, DOMRect>;
let cleanups: (() => void)[];

function renderLauncher() {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  const onOpen = vi.fn();
  act(() => root.render(<AgentMinimizedLauncher onOpen={onOpen} />));
  const button = container.querySelector<HTMLButtonElement>("button")!;
  const unmount = () => act(() => root.unmount());
  cleanups.push(unmount);
  return { button, onOpen, unmount };
}

function createFooter(height: number, left = 0, width = window.innerWidth) {
  const footer = document.createElement("footer");
  footer.dataset.slot = "sticky-action-footer";
  rects.set(
    footer,
    new DOMRect(left, window.innerHeight - height, width, height)
  );
  return footer;
}

function notifyResize() {
  act(() => {
    for (const callback of resizeCallbacks) callback();
  });
}

beforeEach(() => {
  resizeCallbacks = [];
  disconnects = [];
  rects = new Map();
  cleanups = [];
  vi.stubGlobal("innerWidth", 1600);
  vi.stubGlobal("innerHeight", 1000);
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe = vi.fn();
      disconnect = vi.fn();
      constructor(callback: () => void) {
        resizeCallbacks.push(callback);
        disconnects.push(this.disconnect);
      }
    }
  );
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
    function (this: Element) {
      if (
        this instanceof HTMLButtonElement &&
        this.hasAttribute("data-agent-window")
      ) {
        const bottom = Number.parseFloat(this.style.bottom) || 16;
        return new DOMRect(
          window.innerWidth - 56,
          window.innerHeight - bottom - 40,
          40,
          40
        );
      }
      return rects.get(this) ?? new DOMRect();
    }
  );
});

afterEach(() => {
  for (const cleanup of cleanups) cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  document.body.innerHTML = "";
});

describe("AgentMinimizedLauncher", () => {
  test("remains a labeled icon button and opens only when clicked", () => {
    const { button, onOpen } = renderLauncher();
    expect(button.type).toBe("button");
    expect(button.getAttribute("aria-label")).toBe("Agent");
    expect(button.title).toBe("Agent");
    expect(button.querySelector(".lucide-bot")).toBeInstanceOf(SVGElement);
    expect(button.style.bottom).toBe("16px");
    expect(onOpen).not.toHaveBeenCalled();
    act(() => button.click());
    expect(onOpen).toHaveBeenCalledOnce();
  });

  test.each([
    { width: 1600, height: 1000, footerHeight: 72 },
    { width: 390, height: 844, footerHeight: 120 },
  ])(
    "clears an already mounted footer at $width px without oscillating",
    ({ width, height, footerHeight }) => {
      vi.stubGlobal("innerWidth", width);
      vi.stubGlobal("innerHeight", height);
      const footer = createFooter(footerHeight);
      document.body.appendChild(footer);
      const { button } = renderLauncher();
      for (let i = 0; i < 4; i++) {
        expect(button.style.bottom).toBe(`${footerHeight + 16}px`);
        expect(button.getBoundingClientRect().bottom).toBe(
          height - footerHeight - 16
        );
        notifyResize();
      }
    }
  );

  test("tracks footer mount, wrapping, visibility, and removal", async () => {
    const { button } = renderLauncher();
    const footer = createFooter(72);
    await act(async () => {
      document.body.appendChild(footer);
    });
    expect(button.style.bottom).toBe("88px");

    rects.set(footer, new DOMRect(0, 864, 1600, 136));
    notifyResize();
    expect(button.style.bottom).toBe("152px");

    footer.style.display = "none";
    notifyResize();
    expect(button.style.bottom).toBe("16px");
    footer.style.display = "";
    notifyResize();
    expect(button.style.bottom).toBe("152px");

    await act(async () => footer.remove());
    expect(button.style.bottom).toBe("16px");
  });

  test("ignores off-axis and offscreen footers and tracks scroll and viewport changes", () => {
    const footer = createFooter(72, 0, 500);
    document.body.appendChild(footer);
    const { button } = renderLauncher();
    expect(button.style.bottom).toBe("16px");

    rects.set(footer, new DOMRect(0, 928, 1600, 72));
    act(() => document.dispatchEvent(new Event("scroll")));
    expect(button.style.bottom).toBe("88px");

    vi.stubGlobal("innerWidth", 390);
    vi.stubGlobal("innerHeight", 844);
    rects.set(footer, new DOMRect(0, 724, 390, 120));
    act(() => window.dispatchEvent(new Event("resize")));
    expect(button.style.bottom).toBe("136px");

    rects.set(footer, new DOMRect(0, 900, 390, 120));
    act(() => document.dispatchEvent(new Event("scroll")));
    expect(button.style.bottom).toBe("16px");
  });

  test("keeps the icon on screen with an oversized footer and releases observers on unmount", () => {
    const footer = createFooter(1000);
    document.body.appendChild(footer);
    const { button, unmount } = renderLauncher();
    expect(button.getBoundingClientRect().top).toBe(16);
    unmount();
    cleanups = [];
    expect(
      disconnects.every((disconnect) => disconnect.mock.calls.length > 0)
    ).toBe(true);
    act(() => window.dispatchEvent(new Event("resize")));
  });
});
