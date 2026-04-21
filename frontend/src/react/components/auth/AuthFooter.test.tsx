import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  emitStorageChangedEvent: vi.fn(),
  setDocumentTitle: vi.fn(),
  routeTitle: vi.fn((_route: unknown) => "Signin"),
  locale: { value: "en-US" },
  currentRoute: {
    value: {
      meta: {} as { title?: (route: unknown) => string },
    },
  },
}));

const originalLocalStorage = globalThis.localStorage;
const storage = new Map<string, string>();
const localStorageMock = {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => {
    storage.set(key, value);
  },
  removeItem: (key: string) => {
    storage.delete(key);
  },
  clear: () => {
    storage.clear();
  },
  key: (index: number) => Array.from(storage.keys())[index] ?? null,
  get length() {
    return storage.size;
  },
};

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/plugins/i18n", () => ({
  default: {
    global: {
      locale: mocks.locale,
    },
  },
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/utils", () => ({
  emitStorageChangedEvent: mocks.emitStorageChangedEvent,
  setDocumentTitle: mocks.setDocumentTitle,
}));

vi.mock("@/utils/storage-keys", () => ({
  STORAGE_KEY_LANGUAGE: "bb.language",
}));

let AuthFooter: typeof import("./AuthFooter").AuthFooter;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.locale.value = "en-US";
  mocks.currentRoute.value.meta = {};
  storage.clear();
  Object.defineProperty(globalThis, "localStorage", {
    value: localStorageMock,
    configurable: true,
  });
  ({ AuthFooter } = await import("./AuthFooter"));
});

afterEach(() => {
  Object.defineProperty(globalThis, "localStorage", {
    value: originalLocalStorage,
    configurable: true,
  });
});

describe("AuthFooter", () => {
  test("renders five language links in order", () => {
    mocks.useVueState.mockReturnValue("en-US");
    const { container, render, unmount } = renderIntoContainer(<AuthFooter />);
    render();
    const anchors = Array.from(container.querySelectorAll("a"));
    expect(anchors.map((a) => a.textContent)).toEqual([
      "English",
      "简体中文",
      "Español",
      "日本語",
      "Tiếng việt",
    ]);
    unmount();
  });

  test("highlights the current locale with text-main", () => {
    mocks.useVueState.mockReturnValue("zh-CN");
    const { container, render, unmount } = renderIntoContainer(<AuthFooter />);
    render();
    const anchors = Array.from(container.querySelectorAll("a"));
    const zhAnchor = anchors.find((a) => a.textContent === "简体中文");
    const enAnchor = anchors.find((a) => a.textContent === "English");
    expect(zhAnchor?.className).toContain("text-main");
    expect(enAnchor?.className).not.toContain("text-main");
    unmount();
  });

  test("invokes Vue i18n + storage + title setter on click", () => {
    mocks.useVueState.mockReturnValue("en-US");
    mocks.currentRoute.value.meta = { title: mocks.routeTitle };
    const { container, render, unmount } = renderIntoContainer(<AuthFooter />);
    render();
    const esAnchor = Array.from(container.querySelectorAll("a")).find(
      (a) => a.textContent === "Español"
    );
    expect(esAnchor).toBeDefined();
    act(() => {
      esAnchor?.click();
    });
    expect(mocks.locale.value).toBe("es-ES");
    expect(localStorage.getItem("bb.language")).toBe('"es-ES"');
    expect(mocks.emitStorageChangedEvent).toHaveBeenCalledTimes(1);
    expect(mocks.routeTitle).toHaveBeenCalledTimes(1);
    expect(mocks.setDocumentTitle).toHaveBeenCalledWith("Signin");
    unmount();
  });
});
