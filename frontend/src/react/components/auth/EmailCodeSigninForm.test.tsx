import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  sendEmailLoginCode: vi.fn<
    (email: string, workspace?: string) => Promise<void>
  >(async () => {}),
  resolveWorkspaceName: vi.fn(() => undefined),
  pushNotification: vi.fn(),
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useAuthStore: () => ({
    sendEmailLoginCode: mocks.sendEmailLoginCode,
  }),
}));

vi.mock("@/utils", () => ({
  isValidEmail: (value: string) => /\S+@\S+\.\S+/.test(value),
  resolveWorkspaceName: mocks.resolveWorkspaceName,
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

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}));

let EmailCodeSigninForm: typeof import("./EmailCodeSigninForm").EmailCodeSigninForm;

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
  ({ EmailCodeSigninForm } = await import("./EmailCodeSigninForm"));
});

describe("EmailCodeSigninForm", () => {
  test("auto-submits after the sixth OTP digit", async () => {
    const onSignin = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <EmailCodeSigninForm loading={false} onSignin={onSignin} />
    );
    render();

    const emailInput = container.querySelector<HTMLInputElement>(
      'input[type="email"]'
    );
    expect(emailInput).toBeTruthy();
    setInputValue(emailInput!, "user@example.com");

    const sendCodeButton = Array.from(
      container.querySelectorAll<HTMLButtonElement>("button")
    ).find((button) => button.textContent === "auth.sign-in.send-code");
    expect(sendCodeButton).toBeTruthy();
    act(() => {
      sendCodeButton?.click();
    });
    await flushPromises();

    expect(mocks.sendEmailLoginCode).toHaveBeenCalledWith(
      "user@example.com",
      undefined
    );

    const otpInputs = Array.from(
      container.querySelectorAll<HTMLInputElement>('input[inputmode="numeric"]')
    );
    expect(otpInputs).toHaveLength(6);

    otpInputs.forEach((input, index) => {
      setInputValue(input, String(index + 1));
    });

    expect(onSignin).toHaveBeenCalledTimes(1);
    expect(onSignin).toHaveBeenCalledWith({
      email: "user@example.com",
      emailCode: "123456",
      workspace: undefined,
    });

    unmount();
  });
});
