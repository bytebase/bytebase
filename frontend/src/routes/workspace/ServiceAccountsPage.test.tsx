import type { ButtonHTMLAttributes, ReactElement, ReactNode } from "react";
import { act, cloneElement, createElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const { passThrough } = vi.hoisted(() => ({
  passThrough: ({ children }: { children?: ReactNode }) => children,
}));

const pagedState = vi.hoisted(() => ({ hasMore: false, refresh: vi.fn() }));

const {
  mockDeleteServiceAccount,
  mockPushNotification,
  mockRemoveCache,
  mockUpdateCache,
  mockUpdateServiceAccount,
  mockWriteTextToClipboard,
} = vi.hoisted(() => ({
  mockDeleteServiceAccount: vi.fn(),
  mockPushNotification: vi.fn(),
  mockRemoveCache: vi.fn(),
  mockUpdateCache: vi.fn(),
  mockUpdateServiceAccount: vi.fn(),
  mockWriteTextToClipboard: vi.fn(),
}));


vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string, values?: Record<string, string>) =>
      values?.name ? `${key}:${values.name}` : key,
  }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
}));

vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: ({ tsMs }: { tsMs: number }) => createElement("span", {}, tsMs),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageInfo: () => null,
  ProjectPageLayout: passThrough,
  ProjectPageToolbar: passThrough,
}));

vi.mock("@/components/WorkspacePageLayout", () => ({
  WorkspacePageInfo: ({ description }: { description: string }) =>
    createElement("p", { "data-testid": "page-info" }, description),
  WorkspacePageLayout: passThrough,
  WorkspacePageToolbar: passThrough,
}));

vi.mock("@/components/RoleSelect", () => ({ RoleSelect: () => null }));

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ open, children }: { open: boolean; children: ReactNode }) =>
    open ? createElement("div", { role: "alertdialog" }, children) : null,
  AlertDialogContent: passThrough,
  AlertDialogDescription: passThrough,
  AlertDialogFooter: passThrough,
  AlertDialogTitle: passThrough,
}));

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    appearance: _appearance,
    size: _size,
    variant: _variant,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    appearance?: string;
    size?: string;
    variant?: string;
  }) => createElement("button", { type: "button", ...props }, children),
}));

vi.mock("@/components/ui/copy-button", () => ({ CopyButton: () => null }));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: passThrough,
  DropdownMenuContent: ({ children }: { children: ReactNode }) =>
    createElement("div", { role: "menu" }, children),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactNode;
    onClick: () => void;
  }) => createElement("button", { role: "menuitem", onClick }, children),
  DropdownMenuSeparator: () => createElement("hr"),
  DropdownMenuTrigger: ({
    render,
    children,
  }: {
    render: ReactElement;
    children: ReactNode;
  }) => cloneElement(render, {}, children),
}));

vi.mock("@/components/ui/ellipsis-text", () => ({
  EllipsisText: ({ text }: { text: string }) =>
    createElement("span", {}, text),
}));

vi.mock("@/components/ui/form", () => ({
  FormField: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/components/ui/input", () => ({
  Input: () => createElement("input"),
}));

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: () => createElement("input", { type: "search" }),
}));

vi.mock("@/components/ui/select", () => ({
  Select: passThrough,
  SelectContent: () => null,
  SelectItem: () => null,
  SelectTrigger: () => null,
  SelectValue: () => null,
}));

vi.mock("@/components/ui/sheet", () => ({
  Sheet: () => null,
  SheetBody: () => null,
  SheetContent: () => null,
  SheetFooter: () => null,
  SheetHeader: () => null,
  SheetTitle: () => null,
}));

vi.mock("@/components/ui/table", () => ({
  Table: ({ children }: { children: ReactNode }) =>
    createElement("table", {}, children),
  TableBody: ({ children }: { children: ReactNode }) =>
    createElement("tbody", {}, children),
  TableCell: ({ children }: { children: ReactNode }) =>
    createElement("td", {}, children),
  TableEmptyView: ({ children }: { children: ReactNode }) =>
    createElement("tr", {}, createElement("td", {}, children)),
  TableHead: ({ children }: { children: ReactNode }) =>
    createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) =>
    createElement("thead", {}, children),
  TableRow: ({ children }: { children: ReactNode }) =>
    createElement("tr", {}, children),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/hooks/usePagedData", () => ({
  PagedTableFooter: () => null,
  usePagedData: () => {
    const [dataList, setDataList] = useState([
      {
        name: "serviceAccounts/bot@service.bytebase.com",
        title: "bot",
        email: "bot@service.bytebase.com",
        state: 1,
        serviceKey: "",
      },
    ]);
    return {
      dataList,
      hasMore: pagedState.hasMore,
      isFetchingMore: false,
      isLoading: false,
      refresh: pagedState.refresh,
      loadMore: vi.fn(),
      onPageSizeChange: vi.fn(),
      pageSize: 20,
      pageSizeOptions: [20],
      removeCache: mockRemoveCache,
      updateCache: (accounts: typeof dataList) => {
        mockUpdateCache(accounts);
        setDataList(accounts);
      },
    };
  },
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mockWriteTextToClipboard,
}));

vi.mock("@/stores", () => ({ pushNotification: mockPushNotification }));

vi.mock("@/stores/app", () => {
  const state = {
    createServiceAccount: vi.fn(),
    deleteServiceAccount: mockDeleteServiceAccount,
    fetchWorkspaceIamPolicy: vi.fn(),
    getProjectIamPolicy: vi.fn(),
    listServiceAccounts: vi.fn(),
    loadProjectIamPolicy: vi.fn(),
    patchWorkspaceIamPolicy: vi.fn(),
    projectPoliciesByName: {},
    projectsByName: {},
    undeleteServiceAccount: vi.fn(),
    updateProjectIamPolicy: vi.fn(),
    updateServiceAccount: mockUpdateServiceAccount,
    workspacePolicy: {
      bindings: [
        {
          role: "roles/workspaceDBA",
          members: ["serviceAccount:bot@service.bytebase.com"],
        },
        {
          role: "roles/workspaceMember",
          members: ["serviceAccount:bot@service.bytebase.com"],
        },
      ],
    },
    workspaceResourceName: () => "workspaces/-",
  };
  return {
    useAppStore: Object.assign(
      <T,>(selector: (store: typeof state) => T) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/types", () => ({
  getServiceAccountNameInBinding: (email: string) => `serviceAccount:${email}`,
  getServiceAccountSuffix: () => "service.bytebase.com",
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
}));

vi.mock("@/utils/role", () => ({
  displayRoleTitle: (role: string) => role.replace("roles/", ""),
}));

vi.mock("@/utils/v1/iam", () => ({ isBindingPolicyExpired: () => false }));

import { ServiceAccountsPage } from "./ServiceAccountsPage";

describe("ServiceAccountsPage", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  const menuItem = (label: string) =>
    Array.from(
      container.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')
    ).find((item) => item.textContent === label);

  const dialogButton = (label: string) =>
    Array.from(
      document.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button')
    ).find((button) => button.textContent === label);

  const tableText = () => container.querySelector("tbody")?.textContent ?? "";

  beforeEach(() => {
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    mockUpdateServiceAccount.mockResolvedValue({
      name: "serviceAccounts/bot@service.bytebase.com",
      title: "bot",
      email: "bot@service.bytebase.com",
      state: 1,
      serviceKey: "new-service-key",
    });
    mockWriteTextToClipboard.mockResolvedValue(true);
    mockDeleteServiceAccount.mockResolvedValue(undefined);
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("shows the role, the page description and one row menu", async () => {
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    expect(
      container.querySelector('[data-testid="page-info"]')?.textContent
    ).toBe("account-list.service-accounts-description");
    expect(tableText()).toContain("workspaceDBAworkspaceMember");
    expect(tableText()).not.toContain("settings.members.service-account");
    expect(
      Array.from(container.querySelectorAll('[role="menuitem"]')).map(
        (item) => item.textContent
      )
    ).toEqual([
      "settings.members.reset-service-key",
      "common.edit",
      "common.deactivate",
    ]);
  });

  it("confirms and acknowledges a reset service key", async () => {
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    await act(async () => menuItem("settings.members.reset-service-key")?.click());
    expect(document.querySelector('[role="alertdialog"]')?.textContent).toContain(
      "settings.members.reset-service-key-alert"
    );

    vi.useFakeTimers();
    await act(async () => dialogButton("common.reset")?.click());

    expect(mockUpdateServiceAccount).toHaveBeenCalledOnce();
    expect(mockWriteTextToClipboard).toHaveBeenCalledWith("new-service-key");
    expect(document.querySelector('[role="alertdialog"]')).toBeNull();
    expect(mockUpdateCache).toHaveBeenCalledWith([
      expect.objectContaining({ serviceKey: "" }),
    ]);
    expect(mockPushNotification).toHaveBeenCalledWith(
      expect.objectContaining({ style: "SUCCESS", title: "common.copied" })
    );
    expect(tableText()).toContain(
      "settings.members.service-key-reset-and-copied"
    );

    await act(async () => vi.advanceTimersByTime(2000));
    expect(tableText()).not.toContain(
      "settings.members.service-key-reset-and-copied"
    );
  });

  it("keeps the new key available to copy when automatic copying fails", async () => {
    mockWriteTextToClipboard.mockResolvedValue(false);
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    await act(async () => menuItem("settings.members.reset-service-key")?.click());
    await act(async () => dialogButton("common.reset")?.click());

    expect(mockUpdateCache).toHaveBeenCalledWith([
      expect.objectContaining({ serviceKey: "new-service-key" }),
    ]);
    expect(mockPushNotification).toHaveBeenCalledWith(
      expect.objectContaining({ style: "CRITICAL", title: "common.copy-failed" })
    );
    expect(tableText()).toContain("settings.members.copy-service-key");
  });

  it("closes the reset confirmation without resetting on cancel", async () => {
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    await act(async () => menuItem("settings.members.reset-service-key")?.click());
    await act(async () => dialogButton("common.cancel")?.click());

    expect(document.querySelector('[role="alertdialog"]')).toBeNull();
    expect(mockUpdateServiceAccount).not.toHaveBeenCalled();
  });

  it("names the account when deactivating and drops it from the active list", async () => {
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    await act(async () => menuItem("common.deactivate")?.click());
    expect(document.querySelector('[role="alertdialog"]')?.textContent).toContain(
      "account-list.deactivate-title:bot"
    );

    await act(async () => dialogButton("common.deactivate")?.click());

    expect(mockDeleteServiceAccount).toHaveBeenCalledWith(
      "serviceAccounts/bot@service.bytebase.com"
    );
    expect(mockRemoveCache).toHaveBeenCalledWith(
      expect.objectContaining({ email: "bot@service.bytebase.com", state: 2 })
    );
    expect(document.querySelector('[role="alertdialog"]')).toBeNull();
  });

  it("refetches when deactivating the last loaded row leaves more pages", async () => {
    pagedState.hasMore = true;
    try {
      await act(async () => root.render(createElement(ServiceAccountsPage)));
      await act(async () => menuItem("common.deactivate")?.click());
      await act(async () => dialogButton("common.deactivate")?.click());
      expect(mockRemoveCache).toHaveBeenCalledOnce();
      expect(pagedState.refresh).toHaveBeenCalledOnce();
    } finally {
      pagedState.hasMore = false;
    }
  });
});
