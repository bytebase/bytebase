import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const restriction: {
    passwordResetEnabled: boolean;
    disallowPasswordSignin: boolean;
  } = {
    passwordResetEnabled: true,
    disallowPasswordSignin: false,
  };
  return {
    restriction,
    useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) =>
      getter()
    ),
    useActuatorV1Store: vi.fn(() => ({
      serverInfo: { restriction },
    })),
    pushNotification: vi.fn(),
    routerPush: vi.fn(),
    routerReplace: vi.fn(),
    currentRoute: { value: { query: {} as Record<string, string> } },
    requestPasswordReset: vi.fn(),
    resolveWorkspaceName: vi.fn(() => undefined),
  };
});

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useActuatorV1Store: mocks.useActuatorV1Store,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
    replace: mocks.routerReplace,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/router/auth", () => ({
  AUTH_PASSWORD_RESET_MODULE: "auth.password.reset",
  AUTH_SIGNIN_MODULE: "auth.signin",
}));

vi.mock("@/connect", () => ({
  authServiceClientConnect: {
    requestPasswordReset: mocks.requestPasswordReset,
  },
}));

vi.mock("@/utils", async () => {
  const actual = await vi.importActual<typeof import("@/utils")>("@/utils");
  return {
    ...actual,
    resolveWorkspaceName: mocks.resolveWorkspaceName,
  };
});

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

let PasswordForgotPage: typeof import("./PasswordForgotPage").PasswordForgotPage;

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
  mocks.restriction.passwordResetEnabled = true;
  mocks.restriction.disallowPasswordSignin = false;
  mocks.currentRoute.value.query = {};
  ({ PasswordForgotPage } = await import("./PasswordForgotPage"));
});

describe("PasswordForgotPage", () => {
  test("renders self-host warning when passwordResetEnabled is false", () => {
    mocks.restriction.passwordResetEnabled = false;
    const { container, render, unmount } = renderIntoContainer(
      <PasswordForgotPage />
    );
    render();
    expect(container.textContent).toContain("auth.password-forget.selfhost");
    expect(container.querySelector("#forgot-email")).toBeNull();
    unmount();
  });

  test("renders email input and disabled send button when passwordResetEnabled is true", () => {
    const { container, render, unmount } = renderIntoContainer(
      <PasswordForgotPage />
    );
    render();
    const input = container.querySelector<HTMLInputElement>("#forgot-email");
    expect(input).not.toBeNull();
    const button = container.querySelector<HTMLButtonElement>("button");
    expect(button?.textContent).toBe("auth.password-forget.send-reset-code");
    expect(button?.disabled).toBe(true);
    unmount();
  });

  test("enables submit button for a valid email", () => {
    const { container, render, unmount } = renderIntoContainer(
      <PasswordForgotPage />
    );
    render();
    const input = container.querySelector<HTMLInputElement>("#forgot-email");
    expect(input).not.toBeNull();
    setInputValue(input!, "foo@bar.com");
    const button = container.querySelector<HTMLButtonElement>("button");
    expect(button?.disabled).toBe(false);
    unmount();
  });

  test("submit calls requestPasswordReset then navigates to password-reset", async () => {
    mocks.requestPasswordReset.mockResolvedValue({});
    const { container, render, unmount } = renderIntoContainer(
      <PasswordForgotPage />
    );
    render();
    const input = container.querySelector<HTMLInputElement>("#forgot-email")!;
    setInputValue(input, "foo@bar.com");
    const button = container.querySelector<HTMLButtonElement>("button")!;
    act(() => {
      button.click();
    });
    await flushPromises();
    expect(mocks.requestPasswordReset).toHaveBeenCalledWith({
      email: "foo@bar.com",
      workspace: undefined,
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "auth.password.reset",
      query: { email: "foo@bar.com" },
    });
    unmount();
  });

  test("shows notification when requestPasswordReset rejects", async () => {
    mocks.requestPasswordReset.mockRejectedValue(new Error("boom"));
    const { container, render, unmount } = renderIntoContainer(
      <PasswordForgotPage />
    );
    render();
    const input = container.querySelector<HTMLInputElement>("#forgot-email")!;
    setInputValue(input, "foo@bar.com");
    const button = container.querySelector<HTMLButtonElement>("button")!;
    act(() => {
      button.click();
    });
    await flushPromises();
    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "CRITICAL",
        title: "auth.password-forget.failed-to-send-code",
      })
    );
    expect(mocks.routerPush).not.toHaveBeenCalled();
    unmount();
  });

  test("redirects to signin when disallowPasswordSignin is true", () => {
    mocks.restriction.disallowPasswordSignin = true;
    const { render, unmount } = renderIntoContainer(<PasswordForgotPage />);
    render();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "auth.signin",
      query: {},
    });
    unmount();
  });
});
