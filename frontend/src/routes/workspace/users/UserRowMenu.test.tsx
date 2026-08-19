import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const activeUser = {
  name: "users/alice@example.com",
  email: "alice@example.com",
  title: "Alice",
  phone: "",
  password: "",
  state: State.ACTIVE,
  mfaEnabled: true,
  groups: [],
};

const mocks = vi.hoisted(() => ({
  hasWorkspacePermissionV2: vi.fn(() => true),
  isSaaSMode: vi.fn(() => false),
  pushNotification: vi.fn(),
  updateUser: vi.fn(),
  updateEmail: vi.fn(),
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/stores/app", () => {
  const buildState = () => ({
    updateUser: mocks.updateUser,
    updateEmail: mocks.updateEmail,
    isSaaSMode: () => mocks.isSaaSMode(),
    getWorkspaceProfile: () => ({
      requireMfa: false,
      passwordRestriction: undefined,
    }),
  });
  const useAppStore = (selector: (state: unknown) => unknown) =>
    selector(buildState());
  useAppStore.getState = () => buildState();
  return { useAppStore };
});

// Render dropdown content inline so the menu items are inspectable without
// driving Base UI's portal + open state.
vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  DropdownMenuTrigger: () => null,
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="menu">{children}</div>
  ),
  DropdownMenuItem: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="menu-item">{children}</div>
  ),
  DropdownMenuSeparator: () => <hr />,
}));

vi.mock("@/components/ui/dialog", () => ({
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

vi.mock("../profile/EmailInput", () => ({
  EmailInput: () => <input />,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let UserRowMenu: typeof import("./UserRowMenu").UserRowMenu;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: async () => {
      await act(async () => {
        root.render(element);
      });
    },
    unmount: () => act(() => root.unmount()),
  };
};

const menuItems = (container: HTMLElement) =>
  Array.from(container.querySelectorAll('[data-testid="menu-item"]')).map(
    (el) => el.textContent
  );

const renderMenu = async (user: typeof activeUser) => {
  const handle = renderIntoContainer(
    <UserRowMenu
      user={user as never}
      isSelf={false}
      onUserUpdated={() => {}}
      onEdit={() => {}}
      onDeactivate={async () => {}}
      onReactivate={async () => {}}
    />
  );
  await handle.render();
  return handle;
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.isSaaSMode.mockReturnValue(false);
  ({ UserRowMenu } = await import("./UserRowMenu"));
});

describe("UserRowMenu", () => {
  test("leads with the two recovery actions", async () => {
    const { container, unmount } = await renderMenu(activeUser);

    const items = menuItems(container);
    expect(items[0]).toBe("settings.members.admin.reset-password");
    expect(items[1]).toBe("settings.members.admin.reset-mfa");

    unmount();
  });

  test("offers no MFA reset for someone who never enrolled", async () => {
    const { container, unmount } = await renderMenu({
      ...activeUser,
      mfaEnabled: false,
    });

    expect(menuItems(container)).not.toContain(
      "settings.members.admin.reset-mfa"
    );

    unmount();
  });

  test("offers reactivate and nothing else for a deactivated account", async () => {
    // UpdateUser and UpdateEmail both reject a deleted user, so any other
    // action here would only produce a "user has been deleted" error. The one
    // thing to do with such an account is bring it back.
    const { container, unmount } = await renderMenu({
      ...activeUser,
      state: State.DELETED,
    });

    expect(menuItems(container)).toEqual([
      "settings.members.admin.reactivate",
    ]);

    unmount();
  });

  test("in cloud, explains instead of offering actions that the API rejects", async () => {
    // UpdateUser, UpdateEmail, DeleteUser and UndeleteUser all refuse to touch
    // another user in SaaS mode, so every one of these would only error.
    mocks.isSaaSMode.mockReturnValue(true);

    const { container, unmount } = await renderMenu(activeUser);

    expect(menuItems(container)).toEqual([]);
    expect(container.textContent).toContain(
      "settings.members.admin.saas-read-only"
    );

    unmount();
  });

  test("shows nothing at all without permission", async () => {
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);

    const { container, unmount } = await renderMenu(activeUser);

    expect(container.querySelector('[data-testid="menu"]')).toBeNull();

    unmount();
  });
});
