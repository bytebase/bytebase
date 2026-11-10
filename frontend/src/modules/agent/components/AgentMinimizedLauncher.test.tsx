import { act, type ComponentProps } from "react";
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

function renderLauncher(
  props: Partial<ComponentProps<typeof AgentMinimizedLauncher>> = {}
) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  const onOpen = vi.fn();
  act(() => root.render(<AgentMinimizedLauncher {...props} onOpen={onOpen} />));
  const button = container.querySelector<HTMLButtonElement>("button")!;
  button.setPointerCapture = vi.fn();
  button.releasePointerCapture = vi.fn();
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

function pointer(
  button: HTMLButtonElement,
  type: string,
  init: PointerEventInit = {}
) {
  act(() => {
    button.dispatchEvent(
      new PointerEvent(type, {
        bubbles: true,
        button: 0,
        pointerId: 1,
        isPrimary: true,
        clientX: 1564,
        clientY: 964,
        ...init,
      })
    );
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
        const right = Number.parseFloat(this.style.right) || 16;
        return new DOMRect(
          window.innerWidth - right - 40,
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
  test("restores a manual position without saving temporary footer displacement", async () => {
    const footer = createFooter(72, 0, 500);
    document.body.appendChild(footer);
    const onPositionChange = vi.fn();
    const { button } = renderLauncher({
      initialPosition: { right: 1316, bottom: 16 },
      onPositionChange,
    });
    expect(button.style.right).toBe("1316px");
    expect(button.style.bottom).toBe("88px");
    notifyResize();
    expect(onPositionChange).not.toHaveBeenCalled();

    await act(async () => footer.remove());
    expect(button.style.bottom).toBe("16px");
    pointer(button, "pointerdown");
    pointer(button, "pointermove", { clientX: 1464, clientY: 864 });
    pointer(button, "pointerup");
    expect(onPositionChange).toHaveBeenLastCalledWith({
      right: 1416,
      bottom: 116,
    });
  });

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

  test.each(["mouse", "touch", "pen"])(
    "drags with a %s pointer without opening on release",
    (pointerType) => {
      const { button, onOpen } = renderLauncher();
      pointer(button, "pointerdown", { pointerType });
      pointer(button, "pointermove", {
        pointerType,
        clientX: 1364,
        clientY: 864,
      });
      pointer(button, "pointerup", { pointerType });
      act(() => button.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 })));

      expect(button.style.right).toBe("216px");
      expect(button.style.bottom).toBe("116px");
      expect(button.setPointerCapture).toHaveBeenCalledWith(1);
      expect(button.releasePointerCapture).toHaveBeenCalledWith(1);
      expect(onOpen).not.toHaveBeenCalled();
      notifyResize();
      expect(button.style.right).toBe("216px");
      expect(button.style.bottom).toBe("116px");

      pointer(button, "pointerdown");
      pointer(button, "pointerup");
      act(() => button.click());
      expect(onOpen).toHaveBeenCalledOnce();
    }
  );

  test("allows click jitter and keyboard activation after dragging", () => {
    const { button, onOpen } = renderLauncher();
    pointer(button, "pointerdown");
    pointer(button, "pointermove", { clientX: 1566, clientY: 966 });
    pointer(button, "pointerup");
    act(() => button.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 })));
    expect(onOpen).toHaveBeenCalledOnce();
    expect(button.style.bottom).toBe("16px");

    pointer(button, "pointerdown");
    pointer(button, "pointermove", { clientX: 1364, clientY: 864 });
    pointer(button, "pointerup");
    act(() => button.click());
    expect(onOpen).toHaveBeenCalledTimes(2);
  });

  test("clamps dragging and viewport changes to the visible bounds", () => {
    const { button } = renderLauncher();
    pointer(button, "pointerdown");
    pointer(button, "pointermove", { clientX: -1000, clientY: -1000 });
    expect(button.getBoundingClientRect().left).toBe(16);
    expect(button.getBoundingClientRect().top).toBe(16);
    pointer(button, "pointerup");

    vi.stubGlobal("innerWidth", 390);
    vi.stubGlobal("innerHeight", 844);
    act(() => window.dispatchEvent(new Event("resize")));
    expect(button.getBoundingClientRect().left).toBe(16);
    expect(button.getBoundingClientRect().top).toBe(16);

    pointer(button, "pointerdown");
    pointer(button, "pointermove", { clientX: 4000, clientY: 4000 });
    pointer(button, "pointerup");
    expect(button.style.right).toBe("16px");
    expect(button.style.bottom).toBe("16px");
  });

  test("checks footer collisions at the dragged horizontal position", () => {
    const footer = createFooter(72, 0, 500);
    document.body.appendChild(footer);
    const { button } = renderLauncher();
    pointer(button, "pointerdown");
    pointer(button, "pointermove", { clientX: 264 });
    pointer(button, "pointerup");
    expect(button.style.right).toBe("1316px");
    expect(button.style.bottom).toBe("88px");
    notifyResize();
    expect(button.style.bottom).toBe("88px");
  });

  test("ignores secondary pointers and stops when pointer capture is lost", () => {
    const { button } = renderLauncher();
    pointer(button, "pointerdown", { button: 2 });
    pointer(button, "pointermove", { clientX: 1364, clientY: 864 });
    expect(button.style.bottom).toBe("16px");

    pointer(button, "pointerdown");
    pointer(button, "pointermove", { pointerId: 2, clientX: 1364, clientY: 864 });
    expect(button.style.bottom).toBe("16px");
    pointer(button, "lostpointercapture");
    pointer(button, "pointermove", { clientX: 1364, clientY: 864 });
    expect(button.style.bottom).toBe("16px");

    pointer(button, "pointerdown");
    pointer(button, "pointercancel");
    pointer(button, "pointermove", { clientX: 1364, clientY: 864 });
    expect(button.style.bottom).toBe("16px");
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
