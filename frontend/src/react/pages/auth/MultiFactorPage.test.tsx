import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>((getter) => getter()),
  login: vi.fn<(payload: unknown) => Promise<void>>(async () => {}),
  useAuthStore: vi.fn(),
  currentRoute: {
    value: { query: {} as Record<string, string> },
  },
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
    currentRoute: mocks.currentRoute,
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

let MultiFactorPage: typeof import("./MultiFactorPage").MultiFactorPage;

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
  mocks.login.mockResolvedValue(undefined);
  ({ MultiFactorPage } = await import("./MultiFactorPage"));
});

describe("MultiFactorPage", () => {
  test("renders OTP mode by default with Authentication code label", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MultiFactorPage />
    );
    render();
    expect(container.textContent).toContain("multi-factor.auth-code");
    // 6 OTP inputs present
    const otpInputs = container.querySelectorAll('input[inputmode="numeric"]');
    expect(otpInputs.length).toBe(6);
    unmount();
  });

  test("switches to RECOVERY_CODE when the user clicks the other method link", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MultiFactorPage />
    );
    render();
    const switchBtn = Array.from(container.querySelectorAll("button")).find(
      (b) =>
        b.textContent?.includes(
          "multi-factor.other-methods.use-recovery-code.self"
        )
    );
    expect(switchBtn).toBeDefined();
    act(() => {
      switchBtn?.click();
    });
    expect(container.textContent).toContain("multi-factor.recovery-code");
    const otpInputs = container.querySelectorAll('input[inputmode="numeric"]');
    expect(otpInputs.length).toBe(0);
    unmount();
  });

  test("switches back to OTP from RECOVERY_CODE", () => {
    const { container, render, unmount } = renderIntoContainer(
      <MultiFactorPage />
    );
    render();
    const toRecovery = Array.from(container.querySelectorAll("button")).find(
      (b) =>
        b.textContent?.includes(
          "multi-factor.other-methods.use-recovery-code.self"
        )
    );
    act(() => {
      toRecovery?.click();
    });
    const toOtp = Array.from(container.querySelectorAll("button")).find((b) =>
      b.textContent?.includes("multi-factor.other-methods.use-auth-app.self")
    );
    expect(toOtp).toBeDefined();
    act(() => {
      toOtp?.click();
    });
    const otpInputs = container.querySelectorAll('input[inputmode="numeric"]');
    expect(otpInputs.length).toBe(6);
    unmount();
  });

  test("challenge submits otpCode when all 6 digits are entered and Verify is pressed", async () => {
    mocks.currentRoute.value.query = { mfaTempToken: "TOKEN" };
    const { container, render, unmount } = renderIntoContainer(
      <MultiFactorPage />
    );
    render();
    const otpInputs = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[inputmode="numeric"]')
    );
    expect(otpInputs.length).toBe(6);
    "123456".split("").forEach((digit, i) => {
      setInputValue(otpInputs[i], digit);
    });
    await flushPromises();
    expect(mocks.login).toHaveBeenCalledTimes(1);
    expect(mocks.login).toHaveBeenCalledWith({
      request: expect.objectContaining({
        mfaTempToken: "TOKEN",
        otpCode: "123456",
      }),
      redirect: true,
    });
    unmount();
  });

  test("challenge submits recoveryCode in RECOVERY_CODE mode", async () => {
    mocks.currentRoute.value.query = { mfaTempToken: "TOKEN" };
    const { container, render, unmount } = renderIntoContainer(
      <MultiFactorPage />
    );
    render();
    const toRecovery = Array.from(container.querySelectorAll("button")).find(
      (b) =>
        b.textContent?.includes(
          "multi-factor.other-methods.use-recovery-code.self"
        )
    );
    act(() => {
      toRecovery?.click();
    });
    const recoveryInput = container.querySelector<HTMLInputElement>(
      'input[placeholder="XXXXXXXXXX"]'
    );
    expect(recoveryInput).not.toBeNull();
    setInputValue(recoveryInput!, "AAA-BBB");
    const form = container.querySelector("form");
    expect(form).not.toBeNull();
    act(() => {
      form!.dispatchEvent(
        new Event("submit", { bubbles: true, cancelable: true })
      );
    });
    await flushPromises();
    expect(mocks.login).toHaveBeenCalledTimes(1);
    const callArg = mocks.login.mock.calls[0]?.[0] as {
      request: {
        recoveryCode?: string;
        otpCode?: string;
        mfaTempToken: string;
      };
    };
    expect(callArg.request.mfaTempToken).toBe("TOKEN");
    expect(callArg.request.recoveryCode).toBe("AAA-BBB");
    expect(callArg.request.otpCode).toBeUndefined();
    unmount();
  });

  test("reads mfaTempToken from route query", async () => {
    mocks.currentRoute.value.query = { mfaTempToken: "FROM-QUERY" };
    const { container, render, unmount } = renderIntoContainer(
      <MultiFactorPage />
    );
    render();
    const otpInputs = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[inputmode="numeric"]')
    );
    "999999".split("").forEach((digit, i) => {
      setInputValue(otpInputs[i], digit);
    });
    await flushPromises();
    expect(mocks.login).toHaveBeenCalledWith({
      request: expect.objectContaining({ mfaTempToken: "FROM-QUERY" }),
      redirect: true,
    });
    unmount();
  });
});
