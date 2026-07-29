import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  activeUserCount: 0,
  currentRoute: {
    value: { query: {} as Record<string, string | undefined> },
  },
  loadServerInfo: vi.fn(),
  replace: vi.fn(),
  signup: vi.fn(),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    currentRoute: mocks.currentRoute,
    replace: mocks.replace,
  },
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({
    activeUserCount: () => mocks.activeUserCount,
    loadServerInfo: mocks.loadServerInfo,
    serverInfo: {
      restriction: {
        disallowSignup: false,
      },
    },
    signup: mocks.signup,
  });
  return {
    useAppStore: Object.assign(
      (selector?: (state: ReturnType<typeof getState>) => unknown) =>
        selector ? selector(getState()) : getState(),
      { getState }
    ),
  };
});

vi.mock("@/utils", () => ({
  isValidEmail: (value: string) => /\S+@\S+\.\S+/.test(value),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
}));

vi.mock("@/components/BytebaseLogo", () => ({
  BytebaseLogo: ({ className }: { className?: string }) => (
    <div className={className} data-testid="bytebase-logo" />
  ),
}));

vi.mock("@/components/auth/AuthFooter", () => ({
  AuthFooter: () => null,
}));

let SignupPage: typeof import("./SignupPage").SignupPage;

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
  mocks.activeUserCount = 0;
  ({ SignupPage } = await import("./SignupPage"));
});

describe("SignupPage", () => {
  test("renders the admin setup title with the normal left-aligned heading layout", () => {
    const { container, render, unmount } = renderIntoContainer(<SignupPage />);
    render();

    const heading = container.querySelector("h2");
    expect(heading?.textContent).toBe("auth.sign-up.admin-title");
    expect(heading?.className).toContain("text-main");
    expect(heading?.querySelector("p")).toBeNull();
    expect(heading?.querySelector(".text-center")).toBeNull();

    unmount();
  });
});
