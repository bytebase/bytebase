import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
  useAuthStore: vi.fn(),
  isLoggedIn: { value: true },
  routerReplace: vi.fn(),
  routerBack: vi.fn(),
  currentRoute: {
    value: {
      query: {} as Record<string, string>,
      fullPath: "/oauth2/consent?x=1",
    },
  },
  fetchImpl: vi.fn(),
}));
mocks.useAuthStore.mockImplementation(() => ({
  get isLoggedIn() {
    return mocks.isLoggedIn.value;
  },
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useAuthStore: mocks.useAuthStore,
}));

vi.mock("@/router", () => ({
  router: {
    replace: mocks.routerReplace,
    back: mocks.routerBack,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/router/auth", () => ({
  AUTH_SIGNIN_MODULE: "auth.signin",
}));

vi.mock("@/react/components/BytebaseLogo", () => ({
  BytebaseLogo: () => null,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, string>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}));

let OAuth2ConsentPage: typeof import("./OAuth2ConsentPage").OAuth2ConsentPage;

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

const flushPromises = () =>
  act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.isLoggedIn.value = true;
  mocks.currentRoute.value.query = {};
  mocks.currentRoute.value.fullPath = "/oauth2/consent";
  globalThis.fetch = mocks.fetchImpl as typeof fetch;
  mocks.fetchImpl.mockReset();
  ({ OAuth2ConsentPage } = await import("./OAuth2ConsentPage"));
});

describe("OAuth2ConsentPage", () => {
  test("redirects to signin when user is not logged in", async () => {
    mocks.isLoggedIn.value = false;
    mocks.currentRoute.value.fullPath = "/oauth2/consent?client_id=abc";
    const { render, unmount } = renderIntoContainer(<OAuth2ConsentPage />);
    render();
    await flushPromises();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "auth.signin",
      query: { redirect: "/oauth2/consent?client_id=abc" },
    });
    unmount();
  });

  test("renders error when required params are missing", async () => {
    mocks.currentRoute.value.query = { client_id: "abc" };
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain(
      "Missing required OAuth2 parameters"
    );
    unmount();
  });

  test("fetches client info and renders consent form", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(mocks.fetchImpl).toHaveBeenCalledWith("/api/oauth2/clients/c1");
    expect(container.textContent).toContain("Acme");
    expect(container.querySelector('form[method="POST"]')).not.toBeNull();
    const hiddenClientId = container.querySelector<HTMLInputElement>(
      'input[name="client_id"]'
    );
    expect(hiddenClientId?.value).toBe("c1");
    unmount();
  });

  test("renders error when client lookup fails", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: false,
      json: async () => ({ error_description: "client unknown" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain("client unknown");
    unmount();
  });

  test("deny creates a programmatic form and submits it", async () => {
    mocks.currentRoute.value.query = {
      client_id: "c1",
      redirect_uri: "https://app/callback",
      state: "s",
      code_challenge: "ch",
      code_challenge_method: "S256",
    };
    mocks.fetchImpl.mockResolvedValue({
      ok: true,
      json: async () => ({ client_name: "Acme" }),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuth2ConsentPage />
    );
    render();
    await flushPromises();
    const denyBtn = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "common.deny");
    expect(denyBtn).toBeDefined();
    const submitSpy = vi
      .spyOn(HTMLFormElement.prototype, "submit")
      .mockImplementation(() => {});
    act(() => {
      denyBtn?.click();
    });
    expect(submitSpy).toHaveBeenCalledTimes(1);
    submitSpy.mockRestore();
    unmount();
  });
});
