import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const legacyCurrentUser = {
  name: "users/alice@example.com",
  email: "alice@example.com",
  title: "Old Alice",
  phone: "",
  password: "",
  state: State.ACTIVE,
  mfaEnabled: false,
  tempRecoveryCodes: [],
};

const updatedCurrentUser = {
  ...legacyCurrentUser,
  title: "New Alice",
};

const mocks = vi.hoisted(() => ({
  useCurrentUser: vi.fn(() => legacyCurrentUser),
  getWorkspaceRolesByName: vi.fn(() => new Set<string>()),
  hasFeature: vi.fn(() => true),
  pushNotification: vi.fn(),
  updateUser: vi.fn(async (_request: unknown) => updatedCurrentUser),
  updateEmail: vi.fn(),
  routerPush: vi.fn(),
  migrateUserStorage: vi.fn(),
  setDocumentTitle: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: mocks.useCurrentUser,
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: vi.fn(),
}));

vi.mock("@/stores/app", () => {
  const buildState = () => ({
    updateUser: mocks.updateUser,
    updateEmail: mocks.updateEmail,
    updateCurrentUserNameForEmailChange: () => {},
    roleList: [],
    workspacePolicy: undefined,
    getWorkspaceRolesByName: mocks.getWorkspaceRolesByName,
    hasFeature: () => mocks.hasFeature(),
    isSaaSMode: () => false,
    getWorkspaceProfile: () => ({
      passwordRestriction: undefined,
      requireMfa: false,
    }),
  });
  const useAppStore = (selector: (state: unknown) => unknown) =>
    selector(buildState());
  useAppStore.getState = () => buildState();
  return { useAppStore };
});

vi.mock("@/stores", () => ({
  hasFeature: mocks.hasFeature,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    push: mocks.routerPush,
    replace: vi.fn(),
    resolve: () => ({ href: "#" }),
  },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  setDocumentTitle: mocks.setDocumentTitle,
  sortRoles: (roles: string[]) => roles,
}));

vi.mock("@/utils/storage-migrate", () => ({
  migrateUserStorage: mocks.migrateUserStorage,
}));

vi.mock("@bufbuild/protobuf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@bufbuild/protobuf")>();
  return {
    ...actual,
    create: (_schema: unknown, data: Record<string, unknown>) => data,
  };
});

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => <span data-testid="feature-badge" />,
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

vi.mock("@/components/ui/input", () => ({
  Input: ({
    ref,
    ...props
  }: React.InputHTMLAttributes<HTMLInputElement> & {
    ref?: React.Ref<HTMLInputElement>;
  }) => <input ref={ref} {...props} />,
}));

vi.mock("@/components/ui/dialog", () => ({
  // Honor `open` — a mock that renders closed dialogs would hide the very
  // thing these tests check, namely that password fields are not sitting in
  // the page waiting to be swept up by the profile Save.
  Dialog: ({ open, children }: { open?: boolean; children: React.ReactNode }) =>
    open ? <>{children}</> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogDescription: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    ...props
  }: React.HTMLAttributes<HTMLDivElement>) => <div {...props}>{children}</div>,
  DropdownMenuTrigger: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/feature-modal", () => ({
  FeatureModal: () => <div data-testid="feature-modal" />,
}));

vi.mock("./EmailInput", () => ({
  EmailInput: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (value: string) => void;
  }) => <input value={value} onChange={(e) => onChange(e.target.value)} />,
}));

vi.mock("./UserPasswordSection", () => ({
  getPasswordErrors: () => ({ hasHint: false, hasMismatch: false }),
  UserPasswordSection: () => <div data-testid="password-section" />,
}));

vi.mock("@/routes/workspace/two-factor/RegenerateRecoveryCodesView", () => ({
  RegenerateRecoveryCodesView: () => <div data-testid="recovery-codes" />,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let AccountSettingsPage: typeof import("./AccountSettingsPage").AccountSettingsPage;

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

const findButton = (container: HTMLElement, text: string) =>
  [...container.querySelectorAll("button")].find(
    (button) => button.textContent === text
  );

const setInputValue = async (input: HTMLInputElement, value: string) => {
  await act(async () => {
    Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    )?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ AccountSettingsPage } = await import("./AccountSettingsPage"));
});

describe("AccountSettingsPage", () => {
  test("saves the display name and renders the update response", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <AccountSettingsPage />
    );

    await render();
    const titleInput = container.querySelector("input")!;
    expect(titleInput.value).toBe("Old Alice");

    // Nothing changed yet, so the profile section save is inert.
    expect(findButton(container, "common.save")?.disabled).toBe(true);

    await setInputValue(titleInput, "New Alice");
    expect(findButton(container, "common.save")?.disabled).toBe(false);

    await act(async () => {
      findButton(container, "common.save")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
      await Promise.resolve();
    });

    expect(mocks.updateUser).toHaveBeenCalledTimes(1);
    expect(mocks.updateUser.mock.calls[0][0]).toMatchObject({
      updateMask: { paths: ["title"] },
    });
    expect(container.querySelector("input")!.value).toBe("New Alice");

    unmount();
  });

  test("does not let you change your own sign-in email", async () => {
    // Moving an account to another address changes the identity someone signs
    // in with. That is an administrative act, done from the Users directory,
    // not something to self-serve — even as an admin on your own account.
    mocks.hasWorkspacePermissionV2.mockReturnValue(true);

    const { container, render, unmount } = renderIntoContainer(
      <AccountSettingsPage />
    );
    await render();

    expect(findButton(container, "settings.account.change-email")).toBeUndefined();
    expect(container.textContent).toContain("settings.account.email-managed");

    unmount();
  });

  test("password is behind its own dialog, not a field in the profile form", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <AccountSettingsPage />
    );

    await render();

    // Changing a password signs the user out everywhere, so it must not be
    // reachable as a field the profile Save happens to sweep up. Until the
    // dialog is opened there are no password inputs on the page at all.
    expect(
      container.querySelector('[data-testid="password-section"]')
    ).toBeNull();

    // The page offers it as a deliberate, separately-confirmed action.
    const trigger = findButton(container, "settings.account.update-password");
    expect(trigger).not.toBeNull();
    expect(trigger?.disabled).toBe(false);

    unmount();
  });
});
