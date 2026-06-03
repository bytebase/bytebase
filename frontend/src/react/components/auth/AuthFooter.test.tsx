import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  language: { value: "en-US" },
  changeLanguage: vi.fn(),
  emitStorageChangedEvent: vi.fn(),
  setDocumentTitle: vi.fn(),
  currentRoute: {
    value: {
      title: undefined as string | undefined,
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

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    i18n: {
      get language() {
        return mocks.language.value;
      },
      changeLanguage: mocks.changeLanguage,
    },
  }),
}));

vi.mock("@/react/i18n", () => ({
  default: { changeLanguage: mocks.changeLanguage },
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
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
  mocks.language.value = "en-US";
  mocks.currentRoute.value.title = undefined;
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
    mocks.language.value = "en-US";
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
    mocks.language.value = "zh-CN";
    const { container, render, unmount } = renderIntoContainer(<AuthFooter />);
    render();
    const anchors = Array.from(container.querySelectorAll("a"));
    const zhAnchor = anchors.find((a) => a.textContent === "简体中文");
    const enAnchor = anchors.find((a) => a.textContent === "English");
    expect(zhAnchor?.className).toContain("text-main");
    expect(enAnchor?.className).not.toContain("text-main");
    unmount();
  });

  test("changes language + storage + title setter on click", () => {
    mocks.language.value = "en-US";
    mocks.currentRoute.value.title = "Signin";
    const { container, render, unmount } = renderIntoContainer(<AuthFooter />);
    render();
    const esAnchor = Array.from(container.querySelectorAll("a")).find(
      (a) => a.textContent === "Español"
    );
    expect(esAnchor).toBeDefined();
    act(() => {
      esAnchor?.click();
    });
    expect(mocks.changeLanguage).toHaveBeenCalledWith("es-ES");
    expect(localStorage.getItem("bb.language")).toBe('"es-ES"');
    expect(mocks.emitStorageChangedEvent).toHaveBeenCalledTimes(1);
    expect(mocks.setDocumentTitle).toHaveBeenCalledWith("Signin");
    unmount();
  });
});
