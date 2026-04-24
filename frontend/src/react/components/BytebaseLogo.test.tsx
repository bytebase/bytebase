import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  record: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(() => ({ fullPath: "/landing" })),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "settings.general.workspace.logo": "Logo",
      })[key] ?? key,
  }),
}));

vi.mock("@/store", () => ({
  useWorkspaceV1Store: vi.fn(),
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.push,
    resolve: mocks.resolve,
  },
}));

vi.mock("@/router/useRecentVisit", () => ({
  useRecentVisit: () => ({
    record: mocks.record,
  }),
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

let BytebaseLogo: typeof import("./BytebaseLogo").BytebaseLogo;

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
  ({ BytebaseLogo } = await import("./BytebaseLogo"));
});

describe("BytebaseLogo", () => {
  test("renders custom workspace logo when present", () => {
    mocks.useVueState.mockReturnValue("https://example.com/logo.png");
    const { container, render, unmount } = renderIntoContainer(
      <BytebaseLogo />
    );
    render();
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("https://example.com/logo.png");
    expect(img?.getAttribute("alt")).toBe("Logo");
    unmount();
  });

  test("renders fallback Bytebase SVG when workspace has no custom logo", () => {
    mocks.useVueState.mockReturnValue("");
    const { container, render, unmount } = renderIntoContainer(
      <BytebaseLogo />
    );
    render();
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("/assets/logo-full.svg");
    expect(img?.getAttribute("alt")).toBe("Bytebase");
    unmount();
  });

  test("records and navigates when a redirect is provided", () => {
    mocks.useVueState.mockReturnValue("");
    const { container, render, unmount } = renderIntoContainer(
      <BytebaseLogo redirect="workspace.landing" />
    );
    render();

    const button = container.querySelector("button");
    expect(button).not.toBeNull();
    act(() => {
      button?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mocks.resolve).toHaveBeenCalledWith({ name: "workspace.landing" });
    expect(mocks.record).toHaveBeenCalledWith("/landing");
    expect(mocks.push).toHaveBeenCalled();
    unmount();
  });
});
