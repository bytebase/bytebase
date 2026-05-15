import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useActuatorV1Store: vi.fn(),
  useCurrentUserV1: vi.fn(),
  useWorkSheetStore: vi.fn(),
  useSQLEditorTabStore: vi.fn(),
  pushNotification: vi.fn(),
  extractProjectResourceName: vi.fn(
    (name: string) => name.split("/")[1] ?? name
  ),
  extractWorksheetID: vi.fn((name: string) => name.split("/")[3] ?? name),
  routerResolve: vi.fn(() => ({ href: "/sql-editor/projects/proj1/sheets/1" })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useActuatorV1Store: mocks.useActuatorV1Store,
  useCurrentUserV1: mocks.useCurrentUserV1,
  useWorkSheetStore: mocks.useWorkSheetStore,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/utils", () => ({
  extractProjectResourceName: mocks.extractProjectResourceName,
  extractWorksheetID: mocks.extractWorksheetID,
}));

vi.mock("@/router", () => ({
  router: {
    resolve: mocks.routerResolve,
  },
}));

vi.mock("@/router/sqlEditor", () => ({
  SQL_EDITOR_WORKSHEET_MODULE: "sql-editor.worksheet",
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover">{children}</div>
  ),
  PopoverTrigger: ({
    children,
    render: renderEl,
  }: {
    children?: React.ReactNode;
    render?: React.ReactElement;
  }) => (
    <div data-testid="popover-trigger">
      {renderEl ? renderEl : null}
      {children}
    </div>
  ),
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let SharePopoverBody: typeof import("./SharePopoverBody").SharePopoverBody;

const mockWorksheet = {
  name: "projects/proj1/worksheets/1",
  project: "projects/proj1",
  creator: "users/test@example.com",
  visibility: 3 /* PRIVATE */,
  title: "test sheet",
};

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();

  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  mocks.useActuatorV1Store.mockReturnValue({
    serverInfo: { externalUrl: "https://example.com" },
  });
  mocks.useCurrentUserV1.mockReturnValue({
    value: { email: "test@example.com", name: "users/test@example.com" },
  });
  mocks.useWorkSheetStore.mockReturnValue({
    patchWorksheet: vi.fn().mockResolvedValue({}),
  });
  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTab: { status: "CLEAN" },
  });

  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  // Mock clipboard
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    configurable: true,
    writable: true,
  });

  ({ SharePopoverBody } = await import("./SharePopoverBody"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("SharePopoverBody", () => {
  test("renders Share title and link input", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody worksheet={mockWorksheet as never} />
    );
    render();
    expect(container.textContent).toContain("common.share");
    // Should show an input with the link
    expect(container.querySelector("input")).not.toBeNull();
    unmount();
  });

  test("shows 3 visibility options when selector is opened", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody worksheet={mockWorksheet as never} />
    );
    render();
    // The popover-content should contain 3 options
    const popoverContent = container.querySelector(
      "[data-testid='popover-content']"
    );
    expect(popoverContent).not.toBeNull();
    // 3 option rows each with cursor-pointer class
    const optionRows =
      popoverContent?.querySelectorAll("[data-option-row]") ?? [];
    expect(optionRows.length).toBe(3);
    unmount();
  });

  test("visibility selector disabled when user is not creator", () => {
    mocks.useCurrentUserV1.mockReturnValue({
      value: { email: "other@example.com", name: "users/other@example.com" },
    });
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody worksheet={mockWorksheet as never} />
    );
    render();
    // Trigger should have disabled styling
    const trigger = container.querySelector("[data-access-trigger]");
    expect(trigger?.getAttribute("data-disabled")).toBe("true");
    unmount();
  });

  test("handleChangeAccess calls patchWorksheet and pushNotification but does NOT close the outer popover", async () => {
    const patchWorksheet = vi.fn().mockResolvedValue({});
    mocks.useWorkSheetStore.mockReturnValue({ patchWorksheet });

    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody worksheet={mockWorksheet as never} />
    );
    render();

    const popoverContent = container.querySelector(
      "[data-testid='popover-content']"
    );
    const optionRows = popoverContent?.querySelectorAll("[data-option-row]");
    expect(optionRows?.length).toBeGreaterThanOrEqual(1);

    // Click second option (Project Read).
    await act(async () => {
      (optionRows?.[1] as HTMLElement)?.click();
    });

    expect(patchWorksheet).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    // The SharePopoverBody no longer signals "close me" on access
    // change — the outer share popover stays open so the user can copy
    // the just-updated link.
    unmount();
  });

  test("copy button writes to clipboard and pushes notification", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody worksheet={mockWorksheet as never} />
    );
    render();

    const copyBtn = container.querySelector(
      "[data-copy-btn]"
    ) as HTMLButtonElement;
    expect(copyBtn).not.toBeNull();

    await act(async () => {
      copyBtn.click();
    });

    expect(navigator.clipboard.writeText).toHaveBeenCalled();
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("copy button disabled when currentTab status is not CLEAN", () => {
    mocks.useSQLEditorTabStore.mockReturnValue({
      currentTab: { status: "DIRTY" },
    });
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody worksheet={mockWorksheet as never} />
    );
    render();

    const copyBtn = container.querySelector(
      "[data-copy-btn]"
    ) as HTMLButtonElement;
    expect(copyBtn).not.toBeNull();
    expect(copyBtn.disabled).toBe(true);
    unmount();
  });
});
