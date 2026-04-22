import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
  routerPush: vi.fn(),
  currentRoute: {
    value: { query: {} as Record<string, string | undefined> },
  },
  pushNotification: vi.fn(),
  openWindowForSSO: vi.fn(),
  actuatorStore: null as unknown,
  identityProviderStore: null as unknown,
  authStore: null as unknown,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/router/auth", () => ({
  AUTH_SIGNUP_MODULE: "auth.signup",
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useActuatorV1Store: () => mocks.actuatorStore,
  useAuthStore: () => mocks.authStore,
  useIdentityProviderStore: () => mocks.identityProviderStore,
}));

vi.mock("@/utils", () => ({
  openWindowForSSO: mocks.openWindowForSSO,
  resolveWorkspaceName: () => undefined,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}));

vi.mock("@/react/components/BytebaseLogo", () => ({
  BytebaseLogo: () => null,
}));

vi.mock("@/react/components/auth/AuthFooter", () => ({
  AuthFooter: () => null,
}));

let SigninPage: typeof import("./SigninPage").SigninPage;

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

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.currentRoute.value.query = {};
  mocks.actuatorStore = {
    serverInfo: {
      restriction: {
        disallowPasswordSignin: true,
        allowEmailCodeSignin: false,
        disallowSignup: false,
      },
    },
    isDemo: false,
    isSaaSMode: false,
    activeUserCount: 1,
    fetchServerInfo: vi.fn(async () => ({})),
  };
  mocks.identityProviderStore = {
    identityProviderList: [
      {
        name: "idps/corp-ldap",
        title: "Corp LDAP",
        type: IdentityProviderType.LDAP,
      },
    ],
    fetchIdentityProviderList: vi.fn(async () => [
      {
        name: "idps/corp-ldap",
        title: "Corp LDAP",
        type: IdentityProviderType.LDAP,
      },
    ]),
  };
  mocks.authStore = {
    login: vi.fn(async () => {}),
  };
  ({ SigninPage } = await import("./SigninPage"));
});

describe("SigninPage", () => {
  test("renders a username text field for LDAP tabs", async () => {
    const { container, render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    const usernameInput = container.querySelector<HTMLInputElement>(
      'input#username[type="text"]'
    );
    expect(usernameInput).toBeTruthy();
    expect(usernameInput?.placeholder).toBe("jim");
    expect(usernameInput?.getAttribute("autocomplete")).toBe("username");

    const passwordInput = container.querySelector<HTMLInputElement>(
      'input#password[type="password"]'
    );
    expect(passwordInput).toBeTruthy();
    expect(passwordInput?.getAttribute("autocomplete")).toBe(
      "current-password"
    );

    const emailInput = container.querySelector('input#email[type="email"]');
    expect(emailInput).toBeNull();

    unmount();
  });
});
