import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const adminUser = {
  name: "users/admin@example.com",
  email: "admin@example.com",
  title: "Admin",
  phone: "",
  state: State.ACTIVE,
  mfaEnabled: false,
  groups: [],
  tempRecoveryCodes: [],
};

const targetUser = {
  name: "users/bob@example.com",
  email: "bob@example.com",
  title: "Bob",
  phone: "+15550001111",
  state: State.ACTIVE,
  mfaEnabled: true,
  groups: [],
  profile: { source: "" },
  tempRecoveryCodes: [],
};

const mocks = vi.hoisted(() => ({
  useCurrentUser: vi.fn(),
  getUserByIdentifier: vi.fn(),
  getOrFetchUserByIdentifier: vi.fn(),
  updateUser: vi.fn(),
  updateEmail: vi.fn(),
  archiveUser: vi.fn(),
  restoreUser: vi.fn(),
  pushNotification: vi.fn(),
  routerReplace: vi.fn(),
  routerPush: vi.fn(),
  hasWorkspacePermissionV2: vi.fn((_permission: string) => true),
  setDocumentTitle: vi.fn(),
  isSaaSMode: vi.fn(() => false),
}));

const deniedPermissions = new Set<string>();

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: mocks.useCurrentUser,
}));

vi.mock("@/stores/app", () => {
  const buildState = () => ({
    getUserByIdentifier: mocks.getUserByIdentifier,
    getOrFetchUserByIdentifier: mocks.getOrFetchUserByIdentifier,
    updateUser: mocks.updateUser,
    updateEmail: mocks.updateEmail,
    archiveUser: mocks.archiveUser,
    restoreUser: mocks.restoreUser,
    roleList: [],
    workspacePolicy: undefined,
    getWorkspaceRolesByName: () => new Set<string>(),
    getGroupByIdentifier: () => undefined,
    batchGetOrFetchGroups: async () => [],
    isSaaSMode: () => mocks.isSaaSMode(),
    getWorkspaceProfile: () => ({}),
    workspaceUserMapToRoles: () => new Map<string, Set<string>>(),
  });
  const useAppStore = (selector: (state: unknown) => unknown) =>
    selector(buildState());
  useAppStore.getState = () => buildState();
  return { useAppStore };
});

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    replace: mocks.routerReplace,
    push: mocks.routerPush,
    resolve: () => ({ href: "#" }),
  },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  setDocumentTitle: mocks.setDocumentTitle,
  sortRoles: (roles: string[]) => roles,
}));

vi.mock("@/utils/storage-migrate", () => ({
  migrateUserStorage: vi.fn(),
}));

vi.mock("@bufbuild/protobuf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@bufbuild/protobuf")>();
  return {
    ...actual,
    create: (_schema: unknown, data: Record<string, unknown>) => data,
  };
});

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

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div>{children}</div> : null,
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

vi.mock("./UserFormSheet", () => ({
  UserFormSheet: ({ open }: { open: boolean }) =>
    open ? <div data-testid="user-form-sheet" /> : null,
}));

vi.mock("@/routes/workspace/profile/EmailInput", () => ({
  EmailInput: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (value: string) => void;
  }) => <input value={value} onChange={(e) => onChange(e.target.value)} />,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let UserDetailPage: typeof import("./UserDetailPage").UserDetailPage;

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

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useCurrentUser.mockReturnValue(adminUser);
  mocks.getUserByIdentifier.mockReturnValue(targetUser);
  mocks.getOrFetchUserByIdentifier.mockResolvedValue(targetUser);
  mocks.isSaaSMode.mockReturnValue(false);
  deniedPermissions.clear();
  mocks.hasWorkspacePermissionV2.mockImplementation(
    (permission: string) => !deniedPermissions.has(permission)
  );
  ({ UserDetailPage } = await import("./UserDetailPage"));
});

describe("UserDetailPage", () => {
  test("presents the account as administered actions, not a self-service form", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <UserDetailPage principalEmail="bob@example.com" />
    );

    await render();

    expect(container.textContent).toContain("Bob");
    expect(container.textContent).toContain("bob@example.com");
    // Admin operations are named and separate.
    expect(findButton(container, "common.edit")).toBeDefined();
    expect(
      findButton(container, "settings.members.admin.change-email")
    ).toBeDefined();
    expect(
      findButton(container, "settings.members.admin.reset-mfa")
    ).toBeDefined();
    expect(
      findButton(container, "settings.members.admin.deactivate")
    ).toBeDefined();
    // An admin can never enroll a second factor for somebody else.
    expect(findButton(container, "common.enable")).toBeUndefined();

    unmount();
  });

  test("resetting 2FA clears the enrollment through mfa_enabled", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <UserDetailPage principalEmail="bob@example.com" />
    );

    await render();

    await act(async () => {
      findButton(container, "settings.members.admin.reset-mfa")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    // The confirm dialog repeats the action label; the last one is the confirm.
    const confirmButtons = [...container.querySelectorAll("button")].filter(
      (button) => button.textContent === "settings.members.admin.reset-mfa"
    );
    await act(async () => {
      confirmButtons.at(-1)?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
      await Promise.resolve();
    });

    expect(mocks.updateUser).toHaveBeenCalledTimes(1);
    expect(mocks.updateUser.mock.calls[0][0]).toMatchObject({
      user: { name: "users/bob@example.com", mfaEnabled: false },
      updateMask: { paths: ["mfa_enabled"] },
    });

    unmount();
  });

  test("in SaaS mode the page is read-only for other accounts", async () => {
    mocks.isSaaSMode.mockReturnValue(true);
    const { container, render, unmount } = renderIntoContainer(
      <UserDetailPage principalEmail="bob@example.com" />
    );

    await render();

    expect(container.textContent).toContain(
      "settings.members.admin.saas-read-only"
    );
    expect(findButton(container, "common.edit")).toBeUndefined();
    expect(
      findButton(container, "settings.members.admin.change-email")
    ).toBeUndefined();
    expect(
      findButton(container, "settings.members.admin.reset-mfa")
    ).toBeUndefined();

    unmount();
  });

  test("a member without bb.users.update only gets the directory card", async () => {
    deniedPermissions.add("bb.users.update");
    deniedPermissions.add("bb.users.updateEmail");
    deniedPermissions.add("bb.users.delete");
    deniedPermissions.add("bb.users.undelete");

    const { container, render, unmount } = renderIntoContainer(
      <UserDetailPage principalEmail="bob@example.com" />
    );

    await render();

    expect(container.textContent).toContain("bob@example.com");
    // No security posture, no sign-in history, no admin verbs.
    expect(container.textContent).not.toContain("settings.account.security");
    expect(container.textContent).not.toContain(
      "settings.members.admin.last-sign-in"
    );
    expect(container.textContent).not.toContain(
      "settings.members.admin.account-status"
    );
    expect(findButton(container, "common.edit")).toBeUndefined();

    unmount();
  });

  test("viewing your own record points at account settings instead", async () => {
    mocks.getUserByIdentifier.mockReturnValue(adminUser);
    mocks.getOrFetchUserByIdentifier.mockResolvedValue(adminUser);

    const { container, render, unmount } = renderIntoContainer(
      <UserDetailPage principalEmail="admin@example.com" />
    );

    await render();

    expect(container.textContent).toContain(
      "settings.members.admin.viewing-self"
    );
    expect(container.textContent).toContain("settings.account.self");
    // You cannot deactivate yourself from the admin surface.
    expect(
      findButton(container, "settings.members.admin.deactivate")
    ).toBeUndefined();

    unmount();
  });
});
