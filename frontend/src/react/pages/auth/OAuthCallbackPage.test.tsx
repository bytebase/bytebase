import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
  login: vi.fn<(payload: unknown) => Promise<void>>(async () => {}),
  useAuthStore: vi.fn(),
  routerPush: vi.fn(),
  currentRoute: {
    value: { query: {} as Record<string, string> },
  },
  retrieveOAuthState: vi.fn(),
  clearOAuthState: vi.fn(),
  resolveWorkspaceName: vi.fn(() => undefined),
}));
mocks.useAuthStore.mockImplementation(() => ({ login: mocks.login }));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useAuthStore: mocks.useAuthStore,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/router/auth", () => ({
  AUTH_SIGNIN_MODULE: "auth.signin",
}));

vi.mock("@/utils/sso", () => ({
  retrieveOAuthState: mocks.retrieveOAuthState,
  clearOAuthState: mocks.clearOAuthState,
}));

vi.mock("@/utils", () => {
  return {
    resolveWorkspaceName: mocks.resolveWorkspaceName,
  };
});

vi.mock("@bufbuild/protobuf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@bufbuild/protobuf")>();
  return {
    ...actual,
    create: (_schema: unknown, data: Record<string, unknown>) => data,
  };
});

vi.mock("@/types/proto-es/v1/auth_service_pb", async (importOriginal) => {
  const actual =
    await importOriginal<
      typeof import("@/types/proto-es/v1/auth_service_pb")
    >();
  return {
    ...actual,
    LoginRequestSchema: {},
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

let OAuthCallbackPage: typeof import("./OAuthCallbackPage").OAuthCallbackPage;

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
  });

const originalOpener = (window as unknown as { opener: unknown }).opener;

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.currentRoute.value.query = {};
  mocks.login.mockResolvedValue(undefined);
  mocks.retrieveOAuthState.mockReset();
  mocks.clearOAuthState.mockReset();
  ({ OAuthCallbackPage } = await import("./OAuthCallbackPage"));
});

afterEach(() => {
  (window as unknown as { opener: unknown }).opener = originalOpener;
});

describe("OAuthCallbackPage", () => {
  test("renders error + back-to-signin link when state token is missing", async () => {
    mocks.currentRoute.value.query = {};
    const { container, render, unmount } = renderIntoContainer(
      <OAuthCallbackPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain(
      "auth.oauth-callback.invalid-state"
    );
    const backBtn = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "auth.back-to-signin");
    expect(backBtn).toBeDefined();
    unmount();
  });

  test("renders error when stored state not found", async () => {
    mocks.currentRoute.value.query = { state: "xyz" };
    mocks.retrieveOAuthState.mockReturnValue(null);
    const { container, render, unmount } = renderIntoContainer(
      <OAuthCallbackPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain(
      "auth.oauth-callback.session-expired"
    );
    unmount();
  });

  test("renders security-validation error and clears state when token mismatches", async () => {
    mocks.currentRoute.value.query = { state: "xyz" };
    mocks.retrieveOAuthState.mockReturnValue({
      token: "OTHER",
      event: "bb.oauth.signin.gh",
      idpType: IdentityProviderType.OAUTH2,
      timestamp: Date.now(),
    });
    const { container, render, unmount } = renderIntoContainer(
      <OAuthCallbackPage />
    );
    render();
    await flushPromises();
    expect(container.textContent).toContain(
      "auth.oauth-callback.security-failed"
    );
    expect(mocks.clearOAuthState).toHaveBeenCalledWith("xyz");
    unmount();
  });

  test("redirect mode: valid signin event calls authStore.login with oauth2Context", async () => {
    mocks.currentRoute.value.query = { state: "xyz", code: "abc" };
    mocks.retrieveOAuthState.mockReturnValue({
      token: "xyz",
      event: "bb.oauth.signin.gh",
      idpType: IdentityProviderType.OAUTH2,
      popup: false,
      redirect: "/home",
      timestamp: Date.now(),
    });
    const { render, unmount } = renderIntoContainer(<OAuthCallbackPage />);
    render();
    await flushPromises();
    expect(mocks.login).toHaveBeenCalledTimes(1);
    const arg = mocks.login.mock.calls[0]?.[0] as {
      request: {
        idpName: string;
        idpContext: {
          context: { case: string; value: { code: string } };
        };
      };
      redirect: boolean;
      redirectUrl: string;
    };
    expect(arg.request.idpName).toBe("gh");
    expect(arg.request.idpContext.context.case).toBe("oauth2Context");
    expect(arg.request.idpContext.context.value.code).toBe("abc");
    expect(arg.redirect).toBe(true);
    expect(arg.redirectUrl).toBe("/home");
    unmount();
  });

  test("redirect mode: OIDC event uses oidcContext", async () => {
    mocks.currentRoute.value.query = { state: "xyz", code: "abc" };
    mocks.retrieveOAuthState.mockReturnValue({
      token: "xyz",
      event: "bb.oauth.signin.okta",
      idpType: IdentityProviderType.OIDC,
      popup: false,
      redirect: "/",
      timestamp: Date.now(),
    });
    const { render, unmount } = renderIntoContainer(<OAuthCallbackPage />);
    render();
    await flushPromises();
    const arg = mocks.login.mock.calls[0]?.[0] as {
      request: {
        idpName: string;
        idpContext: { context: { case: string } };
      };
    };
    expect(arg.request.idpName).toBe("okta");
    expect(arg.request.idpContext.context.case).toBe("oidcContext");
    unmount();
  });

  test("popup mode: dispatches CustomEvent on window.opener with payload", async () => {
    mocks.currentRoute.value.query = { state: "xyz", code: "abc" };
    mocks.retrieveOAuthState.mockReturnValue({
      token: "xyz",
      event: "bb.oauth.signin.gh",
      idpType: IdentityProviderType.OAUTH2,
      popup: true,
      redirect: "/home",
      timestamp: Date.now(),
    });
    const dispatchEvent = vi.fn();
    (window as unknown as { opener: unknown }).opener = {
      closed: false,
      dispatchEvent,
    };
    const { render, unmount } = renderIntoContainer(<OAuthCallbackPage />);
    render();
    await flushPromises();
    expect(dispatchEvent).toHaveBeenCalledTimes(1);
    const arg = dispatchEvent.mock.calls[0]?.[0] as CustomEvent<{
      code: string;
    }>;
    expect(arg).toBeInstanceOf(CustomEvent);
    expect(arg.type).toBe("bb.oauth.signin.gh");
    expect(arg.detail.code).toBe("abc");
    // authStore.login must NOT be called in popup mode
    expect(mocks.login).not.toHaveBeenCalled();
    unmount();
  });
});
