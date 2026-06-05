import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { nativeChange } from "@/react/test-utils/nativeChange";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type CapturedBeforeEachGuard = (
  to: { fullPath: string },
  from: { fullPath: string },
  next: (target?: boolean) => void,
  options?: {
    historyAction?: "POP" | "PUSH" | "REPLACE";
    reset?: () => void;
    retry?: () => void;
  }
) => void;

const mocks = vi.hoisted(() => ({
  beforeEachGuard: undefined as CapturedBeforeEachGuard | undefined,
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
}));

vi.mock("@/react/components/ComponentPermissionGuard", () => ({
  ComponentPermissionGuard: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: ({
    fallback,
  }: {
    clickable?: boolean;
    fallback?: ReactNode;
    feature?: unknown;
  }) => createElement("span", {}, fallback),
}));

vi.mock("@/react/components/HighlightLabelText", () => ({
  HighlightLabelText: ({ text }: { text: string }) =>
    createElement("span", {}, text),
}));

vi.mock("@/react/components/UserCell", () => ({
  UserCell: ({ user }: { user?: { email?: string } }) =>
    createElement("span", {}, user?.email ?? ""),
}));

vi.mock("@/react/components/UserSelect", () => ({
  UserSelect: ({
    disabled,
    onChange,
    value,
  }: {
    disabled?: boolean;
    onChange: (value: string) => void;
    value: string;
  }) =>
    createElement("input", {
      "data-testid": "user-select",
      disabled,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        onChange(e.target.value),
      value,
    }),
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({
    children,
    description,
  }: {
    children?: ReactNode;
    description?: ReactNode;
  }) => createElement("div", {}, description, children),
}));

vi.mock("@/react/components/ui/alert-dialog", () => ({
  AlertDialog: ({
    children,
    open,
  }: {
    children: ReactNode;
    onOpenChange?: (open: boolean) => void;
    open: boolean;
  }) =>
    open
      ? createElement("div", { "data-testid": "alert-dialog" }, children)
      : null,
  AlertDialogContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  AlertDialogFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  AlertDialogTitle: ({ children }: { children: ReactNode }) =>
    createElement("h3", {}, children),
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children?: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
    variant: _variant,
    size: _size,
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    size?: string;
    variant?: string;
  }) => createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: (props: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: ({
    children,
    disabled,
  }: {
    children: ReactNode;
    disabled?: boolean;
    onValueChange?: (value: string | null) => void;
    value?: string | number;
  }) => createElement("div", { "aria-disabled": disabled }, children),
  SelectContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SelectItem: ({ children }: { children: ReactNode; value: unknown }) =>
    createElement("div", {}, children),
  SelectTrigger: ({ children }: { children: ReactNode }) =>
    createElement("button", { type: "button" }, children),
  SelectValue: ({ children }: { children?: ReactNode }) =>
    createElement("span", {}, typeof children === "function" ? null : children),
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? createElement("div", { "data-testid": "sheet" }, children) : null,
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("h2", {}, children),
}));

vi.mock("@/react/components/ui/table", () => ({
  Table: ({ children }: { children: ReactNode }) =>
    createElement("table", {}, children),
  TableBody: ({ children }: { children: ReactNode }) =>
    createElement("tbody", {}, children),
  TableCell: ({ children }: { children?: ReactNode }) =>
    createElement("td", {}, children),
  TableHead: ({ children }: { children?: ReactNode }) =>
    createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) =>
    createElement("thead", {}, children),
  TableRow: ({ children }: { children?: ReactNode }) =>
    createElement("tr", {}, children),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    email: "me@example.com",
    name: "users/me@example.com",
  }),
}));

vi.mock("@/react/hooks/usePagedData", () => ({
  PagedTableFooter: () => null,
  usePagedData: () => ({
    dataList: [],
    hasMore: false,
    isFetchingMore: false,
    isLoading: false,
    loadMore: vi.fn(),
    onPageSizeChange: vi.fn(),
    pageSize: 10,
    pageSizeOptions: [10],
    removeCache: vi.fn(),
    updateCache: vi.fn(),
  }),
}));

vi.mock("@/react/router", () => ({
  SETTING_ROUTE_WORKSPACE_GENERAL: "workspace-general",
  router: {
    afterEach: () => vi.fn(),
    beforeEach: (guard: CapturedBeforeEachGuard) => {
      mocks.beforeEachGuard = guard;
      return vi.fn();
    },
    currentRoute: { value: { query: {} } },
    push: mocks.routerPush,
    replace: mocks.routerReplace,
    resolve: () => ({ href: "/settings/general" }),
  },
}));

vi.mock("@/react/router/handles", () => ({
  SETTING_ROUTE_WORKSPACE_GENERAL: "workspace-general",
}));

vi.mock("@/react/stores/app", () => {
  const storeState = {
    batchGetOrFetchUsers: vi.fn(async () => []),
    createGroup: vi.fn(),
    deleteGroup: vi.fn(),
    fetchGroup: vi.fn(),
    getOrFetchUserByIdentifier: vi.fn(),
    getWorkspaceProfile: () => ({ domains: ["example.com"] }),
    hasInstanceFeature: () => true,
    isSaaSMode: () => true,
    listGroups: vi.fn(async () => ({ groups: [], nextPageToken: "" })),
    updateGroup: vi.fn(),
  };
  const useAppStore = (selector?: (state: typeof storeState) => unknown) =>
    selector ? selector(storeState) : storeState;
  useAppStore.getState = () => storeState;
  return { useAppStore };
});

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/store/modules/v1/common", () => ({
  extractUserEmail: (name: string) => name.replace(/^users\//, ""),
  groupNamePrefix: "groups/",
}));

vi.mock("@/types", () => ({
  UNKNOWN_USER_NAME: "users/unknown",
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: {
    FEATURE_DIRECTORY_SYNC: 1,
    FEATURE_USER_GROUPS: 2,
  },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => true,
  isValidEmail: (email: string) => email.includes("@"),
}));

vi.mock("./shared/AADSyncSheet", () => ({
  AADSyncSheet: () => null,
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

import { GroupsPage } from "./GroupsPage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.beforeEachGuard = undefined;
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

async function renderPage(): Promise<void> {
  await act(async () => {
    root.render(createElement(GroupsPage));
    await Promise.resolve();
  });
}

describe("GroupsPage create group sheet", () => {
  it("asks for explicit confirmation before closing with unsaved changes", async () => {
    await renderPage();

    const createButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.create"
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
    });

    const titleInput = container.querySelector(
      "input[maxlength='200']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(titleInput, "Developers");
      await Promise.resolve();
    });

    const cancelButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.cancel"
    ) as HTMLButtonElement;
    await act(async () => {
      cancelButton.click();
    });

    expect(container.querySelector("[data-testid='sheet']")).not.toBeNull();
    expect(
      container.querySelector("[data-testid='alert-dialog']")
    ).not.toBeNull();
    expect(container.textContent).toContain("common.leave-without-saving");

    const discardButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.discard-changes"
    ) as HTMLButtonElement;
    await act(async () => {
      discardButton.click();
    });

    expect(container.querySelector("[data-testid='sheet']")).toBeNull();
  });

  it("closes without a discard dialog after creating a modified group", async () => {
    await renderPage();

    const createButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.create"
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
    });

    const sheet = container.querySelector(
      "[data-testid='sheet']"
    ) as HTMLDivElement;
    const emailInput = sheet.querySelector(
      "input:not([data-testid='user-select']):not([maxlength])"
    ) as HTMLInputElement;
    const titleInput = sheet.querySelector(
      "input[maxlength='200']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(emailInput, "developers");
      nativeChange(titleInput, "Developers");
      await Promise.resolve();
    });

    const submitButton = [...container.querySelectorAll("button")]
      .filter((button) => button.textContent === "common.create")
      .at(-1) as HTMLButtonElement;
    await act(async () => {
      submitButton.click();
      await Promise.resolve();
    });

    expect(container.querySelector("[data-testid='alert-dialog']")).toBeNull();
    expect(container.querySelector("[data-testid='sheet']")).toBeNull();
  });

  it("replays blocked push navigation as push after discarding changes", async () => {
    await renderPage();

    const createButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.create"
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
    });

    const titleInput = container.querySelector(
      "input[maxlength='200']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(titleInput, "Developers");
      await Promise.resolve();
    });

    let blocked = false;
    await act(async () => {
      mocks.beforeEachGuard?.(
        { fullPath: "/settings/users" },
        { fullPath: "/settings/groups" },
        (target?: boolean) => {
          blocked = target === false;
        },
        { historyAction: "PUSH" }
      );
      await Promise.resolve();
    });

    expect(blocked).toBe(true);
    expect(
      container.querySelector("[data-testid='alert-dialog']")
    ).not.toBeNull();

    const discardButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.discard-changes"
    ) as HTMLButtonElement;
    await act(async () => {
      discardButton.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith("/settings/users");
    expect(mocks.routerReplace).not.toHaveBeenCalled();
  });

  it("resets blocked route navigation after canceling the discard dialog", async () => {
    await renderPage();

    const createButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.create"
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
    });

    const titleInput = container.querySelector(
      "input[maxlength='200']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(titleInput, "Developers");
      await Promise.resolve();
    });

    const reset = vi.fn();
    let blocked = false;
    await act(async () => {
      mocks.beforeEachGuard?.(
        { fullPath: "/settings/users" },
        { fullPath: "/settings/groups" },
        (target?: boolean) => {
          blocked = target === false;
        },
        { historyAction: "PUSH", reset }
      );
      await Promise.resolve();
    });

    expect(blocked).toBe(true);
    expect(
      container.querySelector("[data-testid='alert-dialog']")
    ).not.toBeNull();

    const alertDialog = container.querySelector(
      "[data-testid='alert-dialog']"
    ) as HTMLDivElement;
    const cancelButton = [...alertDialog.querySelectorAll("button")].find(
      (button) => button.textContent === "common.cancel"
    ) as HTMLButtonElement;
    await act(async () => {
      cancelButton.click();
      await Promise.resolve();
    });

    expect(reset).toHaveBeenCalledTimes(1);
    expect(container.querySelector("[data-testid='sheet']")).not.toBeNull();
    expect(container.querySelector("[data-testid='alert-dialog']")).toBeNull();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.routerReplace).not.toHaveBeenCalled();
  });

  it("retries blocked pop navigation after discarding changes", async () => {
    await renderPage();

    const createButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.create"
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
    });

    const titleInput = container.querySelector(
      "input[maxlength='200']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(titleInput, "Developers");
      await Promise.resolve();
    });

    const retry = vi.fn();
    let blocked = false;
    await act(async () => {
      mocks.beforeEachGuard?.(
        { fullPath: "/settings/users" },
        { fullPath: "/settings/groups" },
        (target?: boolean) => {
          blocked = target === false;
        },
        { historyAction: "POP", retry }
      );
      await Promise.resolve();
    });

    expect(blocked).toBe(true);
    expect(
      container.querySelector("[data-testid='alert-dialog']")
    ).not.toBeNull();

    const discardButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.discard-changes"
    ) as HTMLButtonElement;
    await act(async () => {
      discardButton.click();
      await Promise.resolve();
    });

    expect(retry).toHaveBeenCalledTimes(1);
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.routerReplace).not.toHaveBeenCalled();
  });
});
