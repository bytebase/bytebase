import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  currentRoute: {
    value: { query: {} as Record<string, string | undefined> },
  },
  pushNotification: vi.fn(),
  openWindowForSSO: vi.fn(),
  actuatorStore: null as unknown,
  identityProviderList: [] as unknown[],
  listIdentityProviders: vi.fn(),
  authStore: null as unknown,
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    push: mocks.routerPush,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useAuthStore: () => mocks.authStore,
}));

vi.mock("@/react/stores/app", () => {
  const getState = () => ({
    ...(mocks.actuatorStore as Record<string, unknown>),
    identityProviderList: () => mocks.identityProviderList,
    listIdentityProviders: mocks.listIdentityProviders,
    login: (mocks.authStore as { login: unknown }).login,
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
  openWindowForSSO: mocks.openWindowForSSO,
  isValidEmail: (value: string) => /\S+@\S+\.\S+/.test(value),
}));

vi.mock("@/react/lib/workspace", () => ({
  resolveWorkspaceName: () => undefined,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
  // Completes the mock for the react-i18next migration: `@/react/i18n`
  // registers this plugin via `i18next.use(...)`.
  initReactI18next: { type: "3rdParty", init: () => {} },
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
    isSaaSMode: () => false,
    activeUserCount: () => 1,
    fetchServerInfo: vi.fn(async () => ({})),
  };
  mocks.identityProviderList = [
    {
      name: "idps/corp-ldap",
      title: "Corp LDAP",
      type: IdentityProviderType.LDAP,
    },
  ];
  mocks.listIdentityProviders.mockResolvedValue([
    {
      name: "idps/corp-ldap",
      title: "Corp LDAP",
      type: IdentityProviderType.LDAP,
    },
  ]);
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

  test("renders flat OAuth-first layout when email code is the only method", async () => {
    // The real cloud config: the SaaS override sets disallowSignup=true
    // (it gates the password Signup RPC only) — email-code login still
    // creates accounts for unknown emails, so the page is a signup surface.
    mocks.actuatorStore = {
      serverInfo: {
        restriction: {
          disallowPasswordSignin: true,
          allowEmailCodeSignin: true,
          disallowSignup: true,
        },
      },
      isSaaSMode: () => true,
      activeUserCount: () => 1,
      fetchServerInfo: vi.fn(async () => ({})),
    };
    const idps = [
      {
        name: "idps/github",
        title: "GitHub",
        type: IdentityProviderType.OAUTH2,
      },
      {
        name: "idps/google",
        title: "Google",
        type: IdentityProviderType.OAUTH2,
      },
    ];
    mocks.identityProviderList = idps;
    mocks.listIdentityProviders.mockResolvedValue(idps);

    const { container, render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    // Single method: no tab chrome at all.
    expect(container.querySelector('[role="tab"]')).toBeNull();
    expect(container.textContent).not.toContain("auth.sign-in.email-code-tab");

    // Combined sign-in/sign-up copy and SaaS terms line.
    expect(container.textContent).toContain("auth.sign-in.sign-in-or-create");
    expect(container.textContent).toContain("auth.sign-in.tos");

    // OAuth buttons carry brand icons and "Continue with" copy.
    const buttons = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    );
    const githubButton = buttons.find((button) =>
      button.textContent?.includes(
        'auth.sign-in.continue-with-idp:{"idp":"GitHub"}'
      )
    );
    expect(githubButton).toBeTruthy();
    expect(githubButton?.querySelector("svg")).toBeTruthy();
    // Icon must not flex-shrink when a long IdP title squeezes the button.
    expect(githubButton?.querySelector("svg")?.getAttribute("class")).toContain(
      "shrink-0"
    );
    const googleButton = buttons.find((button) =>
      button.textContent?.includes(
        'auth.sign-in.continue-with-idp:{"idp":"Google"}'
      )
    );
    expect(googleButton).toBeTruthy();
    expect(googleButton?.querySelector("svg")).toBeTruthy();

    // OAuth buttons render above the email form.
    const emailInput = container.querySelector<HTMLInputElement>(
      'input[type="email"]'
    );
    expect(emailInput).toBeTruthy();
    expect(
      githubButton!.compareDocumentPosition(emailInput!) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
    expect(container.textContent).toContain("auth.sign-in.continue-with-email");

    unmount();
  });

  test("renders localized tab labels when multiple methods exist", async () => {
    mocks.actuatorStore = {
      serverInfo: {
        restriction: {
          disallowPasswordSignin: false,
          allowEmailCodeSignin: true,
          disallowSignup: false,
        },
      },
      isSaaSMode: () => false,
      activeUserCount: () => 1,
      fetchServerInfo: vi.fn(async () => ({})),
    };
    mocks.identityProviderList = [];
    mocks.listIdentityProviders.mockResolvedValue([]);

    const { container, render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    expect(container.querySelector('[role="tab"]')).toBeTruthy();
    expect(container.textContent).toContain("auth.sign-in.standard-tab");
    expect(container.textContent).not.toContain("Standard");
    expect(container.textContent).toContain("auth.sign-in.email-code-tab");

    unmount();
  });

  test("hides terms line and signup copy on a re-auth surface", async () => {
    // SessionExpiredSurface passes allowSignup={false}: same SaaS config,
    // but the user already has an account — no signup copy, no terms line.
    mocks.actuatorStore = {
      serverInfo: {
        restriction: {
          disallowPasswordSignin: true,
          allowEmailCodeSignin: true,
          disallowSignup: true,
        },
      },
      isSaaSMode: () => true,
      activeUserCount: () => 1,
      fetchServerInfo: vi.fn(async () => ({})),
    };
    mocks.identityProviderList = [];
    mocks.listIdentityProviders.mockResolvedValue([]);

    const { container, render, unmount } = renderIntoContainer(
      <SigninPage allowSignup={false} />
    );
    render();
    await flushPromises();

    expect(container.textContent).not.toContain("auth.sign-in.tos");
    expect(container.textContent).toContain("auth.sign-in.sign-in-to-account");
    expect(container.textContent).not.toContain(
      "auth.sign-in.sign-in-or-create"
    );

    unmount();
  });

  test("hides terms line and signup copy outside SaaS mode", async () => {
    mocks.actuatorStore = {
      serverInfo: {
        restriction: {
          disallowPasswordSignin: true,
          allowEmailCodeSignin: true,
          disallowSignup: true,
        },
      },
      isSaaSMode: () => false,
      activeUserCount: () => 1,
      fetchServerInfo: vi.fn(async () => ({})),
    };
    mocks.identityProviderList = [];
    mocks.listIdentityProviders.mockResolvedValue([]);

    const { container, render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    expect(container.textContent).not.toContain("auth.sign-in.tos");
    expect(container.textContent).toContain("auth.sign-in.sign-in-to-account");
    expect(container.textContent).not.toContain(
      "auth.sign-in.sign-in-or-create"
    );

    unmount();
  });
});
