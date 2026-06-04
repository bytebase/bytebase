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
  useAuthStore: vi.fn(() => ({
    updateCurrentUserNameForEmailChange: vi.fn(),
  })),
  getWorkspaceRolesByName: vi.fn(() => new Set<string>()),
  fetchWorkspaceIamPolicy: vi.fn(async () => undefined),
  hasFeature: vi.fn(() => true),
  pushNotification: vi.fn(),
  getUserByIdentifier: vi.fn(),
  getOrFetchUserByIdentifier: vi.fn(),
  updateUser: vi.fn(async () => updatedCurrentUser),
  updateEmail: vi.fn(),
  routerReplace: vi.fn(),
  routerPush: vi.fn(),
  migrateUserStorage: vi.fn(),
  setDocumentTitle: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: mocks.useCurrentUser,
}));

vi.mock("@/react/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: vi.fn(),
}));

vi.mock("@/react/stores/app", () => {
  const buildState = () => ({
    getUserByIdentifier: mocks.getUserByIdentifier,
    getOrFetchUserByIdentifier: mocks.getOrFetchUserByIdentifier,
    updateUser: mocks.updateUser,
    updateEmail: mocks.updateEmail,
    updateCurrentUserNameForEmailChange: () => {},
    roleList: [],
    workspacePolicy: undefined,
    getWorkspaceRolesByName: mocks.getWorkspaceRolesByName,
    fetchWorkspaceIamPolicy: mocks.fetchWorkspaceIamPolicy,
    hasFeature: () => mocks.hasFeature(),
    // Migrated off the Pinia actuator/setting store mocks.
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

vi.mock("@/store", () => ({
  hasFeature: mocks.hasFeature,
  pushNotification: mocks.pushNotification,
  useAuthStore: mocks.useAuthStore,
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    replace: mocks.routerReplace,
    push: mocks.routerPush,
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

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => <span data-testid="feature-badge" />,
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

vi.mock("@/react/components/ui/input", () => ({
  Input: ({
    ref,
    ...props
  }: React.InputHTMLAttributes<HTMLInputElement> & {
    ref?: React.Ref<HTMLInputElement>;
  }) => <input ref={ref} {...props} />,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ children }: { children: React.ReactNode }) => <>{children}</>,
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

vi.mock("@/react/components/ui/dropdown-menu", () => ({
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

vi.mock("@/react/components/ui/feature-modal", () => ({
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

vi.mock(
  "@/react/pages/settings/two-factor/RegenerateRecoveryCodesView",
  () => ({
    RegenerateRecoveryCodesView: () => <div data-testid="recovery-codes" />,
  })
);

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let ProfilePage: typeof import("./ProfilePage").ProfilePage;

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
  ({ ProfilePage } = await import("./ProfilePage"));
});

describe("ProfilePage", () => {
  test("renders the updated self profile from the update response", async () => {
    const { container, render, unmount } = renderIntoContainer(<ProfilePage />);

    await render();
    expect(container.textContent).toContain("Old Alice");

    const edit = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.edit"
    );
    await act(async () => {
      edit?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const titleInput = container.querySelector("input")!;
    await act(async () => {
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(titleInput, "New Alice");
      titleInput.dispatchEvent(new Event("input", { bubbles: true }));
    });

    const save = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.save"
    );
    await act(async () => {
      save?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      await Promise.resolve();
    });

    expect(mocks.updateUser).toHaveBeenCalled();
    expect(container.textContent).toContain("New Alice");

    unmount();
  });
});
