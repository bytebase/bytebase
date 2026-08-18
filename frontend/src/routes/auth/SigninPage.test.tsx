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
  routerReplace: vi.fn(),
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

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    push: mocks.routerPush,
    replace: mocks.routerReplace,
    resolve: (to: unknown) => ({ href: String(to), fullPath: String(to) }),
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
  useAuthStore: () => mocks.authStore,
}));

vi.mock("@/stores/app", () => {
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

vi.mock("@/lib/workspace", () => ({
  resolveWorkspaceName: () => undefined,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
  // Completes the mock for the react-i18next migration: `@/lib/i18n`
  // registers this plugin via `i18next.use(...)`.
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/components/BytebaseLogo", () => ({
  BytebaseLogo: () => null,
}));

vi.mock("@/components/auth/AuthFooter", () => ({
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
    authenticationInfo: {
      workspace: "workspaces/default",
      restriction: {
        disallowPasswordSignin: true,
        allowEmailCodeSignin: false,
        disallowSignup: false,
      },
    },
    fetchAuthenticationInfo: vi.fn(async () => ({})),
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
  test("redirects to signup when initial admin setup is required", async () => {
    mocks.actuatorStore = {
      authenticationInfo: {
        workspace: "",
        restriction: {
          disallowPasswordSignin: false,
          allowEmailCodeSignin: false,
          disallowSignup: false,
        },
      },
      fetchAuthenticationInfo: vi.fn(async () => ({})),
    };
    mocks.identityProviderList = [];
    mocks.listIdentityProviders.mockResolvedValue([]);

    const { render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "auth.signup",
    });

    unmount();
  });

  test("does not redirect to initial setup when signup is disallowed", async () => {
    mocks.actuatorStore = {
      authenticationInfo: {
        workspace: "",
        restriction: {
          disallowPasswordSignin: false,
          allowEmailCodeSignin: false,
          disallowSignup: true,
        },
      },
      fetchAuthenticationInfo: vi.fn(async () => ({})),
    };
    mocks.identityProviderList = [];
    mocks.listIdentityProviders.mockResolvedValue([]);

    const { render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    expect(mocks.routerReplace).not.toHaveBeenCalledWith({
      name: "auth.signup",
    });

    unmount();
  });

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
    // Email-code can create an account, so its public surface always carries
    // the terms even when the restriction conservatively disallows signup.
    mocks.actuatorStore = {
      authenticationInfo: {
        workspace: "",
        restriction: {
          disallowPasswordSignin: true,
          allowEmailCodeSignin: true,
          disallowSignup: true,
        },
      },
      fetchAuthenticationInfo: vi.fn(async () => ({})),
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

    expect(mocks.routerReplace).not.toHaveBeenCalledWith({
      name: "auth.signup",
    });

    // Single method: no tab chrome at all.
    expect(container.querySelector('[role="tab"]')).toBeNull();
    expect(container.textContent).not.toContain("auth.sign-in.email-code-tab");

    // The restriction owns the signup copy; terms do not require SaaS mode.
    expect(container.textContent).toContain("auth.sign-in.sign-in-to-account");
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
      authenticationInfo: {
        workspace: "workspaces/default",
        restriction: {
          disallowPasswordSignin: false,
          allowEmailCodeSignin: true,
          disallowSignup: false,
        },
      },
      fetchAuthenticationInfo: vi.fn(async () => ({})),
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
    // SessionExpiredSurface passes allowSignup={false}: same email-code config,
    // but the user already has an account — no signup copy, no terms line.
    mocks.actuatorStore = {
      authenticationInfo: {
        workspace: "",
        restriction: {
          disallowPasswordSignin: true,
          allowEmailCodeSignin: true,
          disallowSignup: true,
        },
      },
      fetchAuthenticationInfo: vi.fn(async () => ({})),
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

  test("shows terms for email code without SaaS mode", async () => {
    mocks.actuatorStore = {
      authenticationInfo: {
        workspace: "workspaces/default",
        restriction: {
          disallowPasswordSignin: true,
          allowEmailCodeSignin: true,
          disallowSignup: true,
        },
      },
      fetchAuthenticationInfo: vi.fn(async () => ({})),
    };
    mocks.identityProviderList = [];
    mocks.listIdentityProviders.mockResolvedValue([]);

    const { container, render, unmount } = renderIntoContainer(<SigninPage />);
    render();
    await flushPromises();

    expect(container.textContent).toContain("auth.sign-in.tos");
    expect(container.textContent).toContain("auth.sign-in.sign-in-to-account");
    expect(container.textContent).not.toContain(
      "auth.sign-in.sign-in-or-create"
    );

    unmount();
  });
});
