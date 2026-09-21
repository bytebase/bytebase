import type { ButtonHTMLAttributes, ReactNode } from "react";
import { act, createElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const {
  mockPushNotification,
  mockUpdateCache,
  mockUpdateServiceAccount,
  mockWriteTextToClipboard,
} = vi.hoisted(() => ({
  mockPushNotification: vi.fn(),
  mockUpdateCache: vi.fn(),
  mockUpdateServiceAccount: vi.fn(),
  mockWriteTextToClipboard: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageLayout: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  ProjectPageToolbar: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/components/RoleSelect", () => ({
  RoleSelect: () => null,
}));

vi.mock("@/components/UserCell", () => ({
  UserCell: ({ title, subtitle, badges }: { title: string; subtitle: string; badges: ReactNode }) =>
    createElement("div", {}, title, subtitle, badges),
}));

vi.mock("@/components/WorkspacePageLayout", () => ({
  WorkspacePageLayout: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  WorkspacePageToolbar: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/components/ui/popover", async () => {
  const { createContext, createElement, useContext } = await import("react");
  const { createPortal } = await import("react-dom");
  const { getLayerRoot } = await import("@/components/ui/layer");
  const Context = createContext({
    open: false,
    onOpenChange: (_open: boolean) => {},
  });
  return {
    Popover: ({
      open,
      onOpenChange,
      children,
    }: {
      open: boolean;
      onOpenChange: (open: boolean) => void;
      children: ReactNode;
    }) =>
      createElement(Context.Provider, { value: { open, onOpenChange } }, children),
    PopoverTrigger: ({
      children,
      onClick,
    }: {
      children: ReactNode;
      onClick?: React.MouseEventHandler<HTMLButtonElement>;
    }) => {
      const { open, onOpenChange } = useContext(Context);
      return createElement(
        "button",
        {
          onClick: (event: React.MouseEvent<HTMLButtonElement>) => {
            onClick?.(event);
            onOpenChange(!open);
          },
        },
        children
      );
    },
    PopoverContent: ({ children }: { children: ReactNode }) => {
      const { open } = useContext(Context);
      return open
        ? createPortal(
            createElement("div", { "data-testid": "reset-popover" }, children),
            getLayerRoot("overlay")
          )
        : null;
    },
  };
});

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
  }) => createElement("button", props, children),
}));

vi.mock("@/components/ui/checkbox", () => ({
  Checkbox: () => createElement("input", { type: "checkbox" }),
}));

vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: () => null,
}));

vi.mock("@/components/ui/form", () => ({
  FormField: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/components/ui/input", () => ({
  Input: () => createElement("input"),
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
  Table: ({ children }: { children: ReactNode }) => createElement("table", {}, children),
  TableBody: ({ children }: { children: ReactNode }) => createElement("tbody", {}, children),
  TableCell: ({ children, ...props }: { children: ReactNode }) =>
    createElement("td", props, children),
  TableHead: ({ children }: { children: ReactNode }) => createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) => createElement("thead", {}, children),
  TableRow: ({ children, ...props }: { children: ReactNode }) =>
    createElement("tr", props, children),
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
      hasMore: false,
      isFetchingMore: false,
      isLoading: false,
      loadMore: vi.fn(),
      onPageSizeChange: vi.fn(),
      pageSize: 20,
      pageSizeOptions: [20],
      removeCache: vi.fn(),
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

vi.mock("@/lib/utils", () => ({ cn: (...classes: unknown[]) => classes.filter(Boolean).join(" ") }));

vi.mock("@/stores", () => ({ pushNotification: mockPushNotification }));

vi.mock("@/stores/app", () => {
  const state = {
    createServiceAccount: vi.fn(),
    deleteServiceAccount: vi.fn(),
    getProjectIamPolicy: vi.fn(),
    listServiceAccounts: vi.fn(),
    patchWorkspaceIamPolicy: vi.fn(),
    projectsByName: {},
    undeleteServiceAccount: vi.fn(),
    updateProjectIamPolicy: vi.fn(),
    updateServiceAccount: mockUpdateServiceAccount,
    workspaceResourceName: () => "workspaces/-",
  };
  return {
    useAppStore: Object.assign(
      <T,>(selector: (store: typeof state) => T) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("@/stores/modules/v1/common", () => ({ projectNamePrefix: "projects/" }));

vi.mock("@/types", () => ({
  getServiceAccountNameInBinding: (email: string) => `serviceAccount:${email}`,
  getServiceAccountSuffix: () => "service.bytebase.com",
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
}));

import { ServiceAccountsPage } from "./ServiceAccountsPage";

describe("ServiceAccountsPage", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

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
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("confirms and acknowledges a reset service key in the action", async () => {
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    const resetButton = Array.from(
      container.querySelectorAll<HTMLButtonElement>("tbody button")
    ).find(
      (button) =>
        button.textContent === "settings.members.reset-service-key"
    );
    expect(resetButton).toBeDefined();

    await act(async () => resetButton?.click());
    const popover = document.querySelector('[data-testid="reset-popover"]');
    expect(popover).not.toBeNull();
    expect(document.querySelector('[role="alertdialog"]')).toBeNull();
    expect(popover?.textContent).toContain(
      "settings.members.reset-service-key-alert"
    );
    expect(container.querySelector("tbody")?.textContent).not.toContain(
      "settings.members.reset-service-key-alert"
    );

    const confirmButton = Array.from(
      popover?.querySelectorAll<HTMLButtonElement>("button") ?? []
    ).find((button) => button.textContent === "common.reset");
    expect(confirmButton).toBeDefined();

    vi.useFakeTimers();
    await act(async () => confirmButton?.click());

    expect(mockUpdateServiceAccount).toHaveBeenCalledOnce();
    expect(mockWriteTextToClipboard).toHaveBeenCalledWith("new-service-key");
    expect(document.querySelector('[data-testid="reset-popover"]')).toBeNull();
    expect(mockUpdateCache).toHaveBeenCalledWith([
      expect.objectContaining({ serviceKey: "" }),
    ]);
    expect(mockPushNotification).toHaveBeenCalledWith(
      expect.objectContaining({ style: "SUCCESS", title: "common.copied" })
    );
    expect(container.querySelector("tbody")?.textContent).toContain(
      "settings.members.service-key-reset-and-copied"
    );

    await act(async () => vi.advanceTimersByTime(2000));
    expect(container.querySelector("tbody")?.textContent).toContain(
      "settings.members.reset-service-key"
    );
    expect(container.querySelector("tbody")?.textContent).not.toContain(
      "settings.members.service-key-reset-and-copied"
    );
  });

  it("keeps the new key available to copy when automatic copying fails", async () => {
    mockWriteTextToClipboard.mockResolvedValue(false);
    await act(async () => root.render(createElement(ServiceAccountsPage)));

    const resetButton = Array.from(
      container.querySelectorAll<HTMLButtonElement>("tbody button")
    ).find(
      (button) =>
        button.textContent === "settings.members.reset-service-key"
    );
    expect(resetButton).toBeDefined();
    await act(async () => resetButton?.click());
    const popover = document.querySelector('[data-testid="reset-popover"]');
    const confirmButton = Array.from(
      popover?.querySelectorAll<HTMLButtonElement>("button") ?? []
    ).find((button) => button.textContent === "common.reset");
    await act(async () => confirmButton?.click());

    expect(mockUpdateCache).toHaveBeenCalledWith([
      expect.objectContaining({ serviceKey: "new-service-key" }),
    ]);
    expect(mockPushNotification).toHaveBeenCalledWith(
      expect.objectContaining({ style: "CRITICAL", title: "common.copy-failed" })
    );
    expect(container.querySelector("tbody")?.textContent).toContain(
      "settings.members.copy-service-key"
    );
  });

  it("closes the confirmation popover without resetting on cancel", async () => {
    await act(async () => root.render(createElement(ServiceAccountsPage)));
    const resetButton = Array.from(
      container.querySelectorAll<HTMLButtonElement>("tbody button")
    ).find(
      (button) =>
        button.textContent === "settings.members.reset-service-key"
    );
    await act(async () => resetButton?.click());

    const popover = document.querySelector('[data-testid="reset-popover"]');
    const cancelButton = Array.from(
      popover?.querySelectorAll<HTMLButtonElement>("button") ?? []
    ).find((button) => button.textContent === "common.cancel");
    await act(async () => cancelButton?.click());

    expect(document.querySelector('[data-testid="reset-popover"]')).toBeNull();
    expect(mockUpdateServiceAccount).not.toHaveBeenCalled();
    expect(container.querySelector("tbody")?.textContent).toContain(
      "settings.members.reset-service-key"
    );
  });
});
