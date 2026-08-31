import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const currentTimestamp = () => ({
  seconds: BigInt(Math.floor(Date.now() / 1000)),
  nanos: 0,
});

const currentUser = {
  name: "users/alice@example.com",
  email: "alice@example.com",
  mfaEnabled: false,
  profile: { lastChangePasswordTime: currentTimestamp() },
};

const mintedEnrollment = {
  otpSecret: "new-secret",
  recoveryCodes: ["code-1", "code-2"],
  expireTime: {
    seconds: BigInt(Math.floor(Date.now() / 1000) + 300),
    nanos: 0,
  },
  pendingVersion: currentTimestamp(),
};

const mocks = vi.hoisted(() => ({
  useCurrentUser: vi.fn(() => currentUser),
  startMFAEnrollment: vi.fn(async () => mintedEnrollment),
  setCurrentUser: vi.fn(),
  pushNotification: vi.fn(),
  routerReplace: vi.fn(),
  currentRoute: {
    value: { name: "workspace.setting.profile" },
  },
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: mocks.useCurrentUser,
}));

vi.mock("@/api", () => ({
  userServiceClientConnect: {
    startMFAEnrollment: mocks.startMFAEnrollment,
    enableMFA: vi.fn(),
    confirmRecoveryCodes: vi.fn(),
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      setCurrentUser: mocks.setCurrentUser,
      serverInfo: { saas: false },
    }),
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    replace: mocks.routerReplace,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@bufbuild/protobuf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@bufbuild/protobuf")>();
  return {
    ...actual,
    create: (_schema: unknown, data: Record<string, unknown>) => data,
  };
});

vi.mock("@/types/proto-es/v1/user_service_pb", async (importOriginal) => {
  const actual =
    await importOriginal<
      typeof import("@/types/proto-es/v1/user_service_pb")
    >();
  return {
    ...actual,
    StartMFAEnrollmentRequestSchema: {},
    EnableMFARequestSchema: {},
    ConfirmRecoveryCodesRequestSchema: {},
    CredentialProofSchema: {},
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}));

vi.mock("qrcode.react", () => ({
  QRCodeSVG: ({ value }: { value: string }) => (
    <div data-testid="qr-code" data-value={value} />
  ),
}));

vi.mock("@/components/LearnMoreLink", () => ({
  LearnMoreLink: () => <a href="https://example.com">learn</a>,
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/otp-input", () => ({
  OtpInput: () => <div data-testid="otp-input" />,
}));

vi.mock("@/components/CredentialProofInput", async (importOriginal) => ({
  ...(await importOriginal<
    typeof import("@/components/CredentialProofInput")
  >()),
  CredentialProofInput: () => <div data-testid="credential-proof" />,
  useCredentialProofMode: () => "password" as const,
}));

vi.mock("./RecoveryCodesView", () => ({
  RecoveryCodesView: () => <div data-testid="recovery-codes" />,
}));

vi.mock("./TwoFactorSecretModal", () => ({
  TwoFactorSecretModal: ({ secret }: { secret: string }) => (
    <div data-testid="secret-modal" data-secret={secret} />
  ),
}));

let TwoFactorSetupPage: typeof import("./TwoFactorSetupPage").TwoFactorSetupPage;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: async () => {
      await act(async () => {
        root.render(element);
        await Promise.resolve();
      });
    },
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ TwoFactorSetupPage } = await import("./TwoFactorSetupPage"));
});

describe("TwoFactorSetupPage", () => {
  test("renders the secret from the enrollment response, not from the user", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <TwoFactorSetupPage />
    );

    await render();

    expect(mocks.startMFAEnrollment).toHaveBeenCalledWith(
      expect.objectContaining({ name: currentUser.name })
    );
    expect(
      container
        .querySelector('[data-testid="qr-code"]')
        ?.getAttribute("data-value")
    ).toContain("secret=new-secret");
    expect(
      container
        .querySelector('[data-testid="secret-modal"]')
        ?.getAttribute("data-secret")
    ).toBe("new-secret");
    // An account with a password proves it before the swap.
    expect(
      container.querySelector('[data-testid="credential-proof"]')
    ).not.toBeNull();

    unmount();
  });
});
