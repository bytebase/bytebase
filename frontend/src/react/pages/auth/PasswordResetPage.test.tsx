import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
  useActuatorV1Store: vi.fn(() => ({
    serverInfo: {
      restriction: {
        passwordRestriction: undefined,
        disallowPasswordSignin: false,
      },
    },
  })),
  useAuthStore: vi.fn(() => ({
    requireResetPassword: false,
    setRequireResetPassword: vi.fn(),
    login: vi.fn(async () => {}),
  })),
  useCurrentUserV1: vi.fn(() => ({ value: { name: "users/1", email: "u@e" } })),
  useUserStore: vi.fn(() => ({ updateUser: vi.fn() })),
  pushNotification: vi.fn(),
  routerReplace: vi.fn(),
  routerPush: vi.fn(),
  currentRoute: {
    value: { query: {} as Record<string, string> },
  },
  resetPassword: vi.fn(),
  requestPasswordReset: vi.fn(),
  resolveWorkspaceName: vi.fn(() => undefined),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useActuatorV1Store: mocks.useActuatorV1Store,
  useAuthStore: mocks.useAuthStore,
  useCurrentUserV1: mocks.useCurrentUserV1,
  useUserStore: mocks.useUserStore,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/router", () => ({
  router: {
    replace: mocks.routerReplace,
    push: mocks.routerPush,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/router/auth", () => ({
  AUTH_SIGNIN_MODULE: "auth.signin",
}));

vi.mock("@/connect", () => ({
  authServiceClientConnect: {
    resetPassword: mocks.resetPassword,
    requestPasswordReset: mocks.requestPasswordReset,
  },
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
    ResetPasswordRequestSchema: {},
  };
});

vi.mock("@/types/proto-es/v1/user_service_pb", async (importOriginal) => {
  const actual =
    await importOriginal<
      typeof import("@/types/proto-es/v1/user_service_pb")
    >();
  return {
    ...actual,
    UpdateUserRequestSchema: {},
  };
});

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}));

let PasswordResetPage: typeof import("./PasswordResetPage").PasswordResetPage;

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

const setInputValue = (input: HTMLInputElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    );
    descriptor?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.currentRoute.value.query = {};
  mocks.useActuatorV1Store.mockReturnValue({
    serverInfo: {
      restriction: {
        passwordRestriction: undefined,
        disallowPasswordSignin: false,
      },
    },
  });
  mocks.useAuthStore.mockReturnValue({
    requireResetPassword: true,
    setRequireResetPassword: vi.fn(),
    login: vi.fn(async () => {}),
  });
  ({ PasswordResetPage } = await import("./PasswordResetPage"));
});

describe("PasswordResetPage", () => {
  test("forced-reset mode: redirects when requireResetPassword is false", () => {
    mocks.useAuthStore.mockReturnValue({
      requireResetPassword: false,
      setRequireResetPassword: vi.fn(),
      login: vi.fn(),
    });
    const { render, unmount } = renderIntoContainer(<PasswordResetPage />);
    render();
    expect(mocks.routerReplace).toHaveBeenCalled();
    unmount();
  });

  test("code mode: redirects to signin when password signin is disallowed", () => {
    mocks.currentRoute.value.query = { email: "u@e.com" };
    mocks.useActuatorV1Store.mockReturnValue({
      serverInfo: {
        restriction: {
          passwordRestriction: undefined,
          disallowPasswordSignin: true,
        },
      },
    });
    const { render, unmount } = renderIntoContainer(<PasswordResetPage />);
    render();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "auth.signin",
      query: { email: "u@e.com" },
    });
    unmount();
  });

  test("code mode: renders email + verification code + password fields", () => {
    mocks.currentRoute.value.query = { email: "u@e.com" };
    const { container, render, unmount } = renderIntoContainer(
      <PasswordResetPage />
    );
    render();
    const emailInput = container.querySelector<HTMLInputElement>(
      'input[type="email"]'
    );
    expect(emailInput?.value).toBe("u@e.com");
    expect(emailInput?.disabled).toBe(true);
    const otpInputs = container.querySelectorAll('input[inputmode="numeric"]');
    expect(otpInputs.length).toBe(6);
    const passwordInputs = container.querySelectorAll<HTMLInputElement>(
      'input[type="password"]'
    );
    expect(passwordInputs.length).toBe(2);
    unmount();
  });

  test("confirm button is disabled until valid password + matching confirm", () => {
    mocks.currentRoute.value.query = {};
    const { container, render, unmount } = renderIntoContainer(
      <PasswordResetPage />
    );
    render();
    const confirmBtn = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "common.confirm");
    expect(confirmBtn?.disabled).toBe(true);
    const passwordInputs = container.querySelectorAll<HTMLInputElement>(
      'input[type="password"]'
    );
    setInputValue(passwordInputs[0], "Passw0rd!");
    setInputValue(passwordInputs[1], "Passw0rd!");
    expect(confirmBtn?.disabled).toBe(false);
    unmount();
  });

  test("code mode: confirm calls resetPassword and logs in on success", async () => {
    mocks.currentRoute.value.query = { email: "u@e.com" };
    mocks.resetPassword.mockResolvedValue({});
    const login = vi.fn(async () => {});
    mocks.useAuthStore.mockReturnValue({
      requireResetPassword: true,
      setRequireResetPassword: vi.fn(),
      login,
    });
    const { container, render, unmount } = renderIntoContainer(
      <PasswordResetPage />
    );
    render();
    // Fill OTP
    const otpInputs = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[inputmode="numeric"]')
    );
    "111111".split("").forEach((d, i) => setInputValue(otpInputs[i], d));
    // Fill passwords
    const passwordInputs = container.querySelectorAll<HTMLInputElement>(
      'input[type="password"]'
    );
    setInputValue(passwordInputs[0], "Passw0rd!");
    setInputValue(passwordInputs[1], "Passw0rd!");
    const confirmBtn = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((b) => b.textContent === "common.confirm")!;
    act(() => {
      confirmBtn.click();
    });
    await flushPromises();
    expect(mocks.resetPassword).toHaveBeenCalledWith(
      expect.objectContaining({
        email: "u@e.com",
        code: "111111",
        newPassword: "Passw0rd!",
      })
    );
    expect(login).toHaveBeenCalled();
    unmount();
  });
});
