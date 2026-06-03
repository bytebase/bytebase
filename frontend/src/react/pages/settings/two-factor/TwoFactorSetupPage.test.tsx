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

const legacyCurrentUser = {
  name: "users/alice@example.com",
  email: "alice@example.com",
  tempOtpSecret: "old-secret",
  tempOtpSecretCreatedTime: currentTimestamp(),
  tempRecoveryCodes: [],
};

const regeneratedCurrentUser = {
  ...legacyCurrentUser,
  tempOtpSecret: "new-secret",
  tempOtpSecretCreatedTime: currentTimestamp(),
};

const mocks = vi.hoisted(() => ({
  useCurrentUser: vi.fn(() => legacyCurrentUser),
  updateUser: vi.fn(async () => regeneratedCurrentUser),
  pushNotification: vi.fn(),
  routerReplace: vi.fn(),
  currentRoute: {
    value: { name: "workspace.setting.profile" },
  },
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: mocks.useCurrentUser,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      updateUser: mocks.updateUser,
    }),
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
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
    UpdateUserRequestSchema: {},
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

vi.mock("@/react/components/LearnMoreLink", () => ({
  LearnMoreLink: () => <a href="https://example.com">learn</a>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/otp-input", () => ({
  OtpInput: () => <div data-testid="otp-input" />,
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
  test("renders regenerated MFA secret from the update response", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <TwoFactorSetupPage />
    );

    await render();

    expect(mocks.updateUser).toHaveBeenCalledWith(
      expect.objectContaining({
        regenerateTempMfaSecret: true,
      })
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

    unmount();
  });
});
