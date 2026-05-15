import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ---- hoisted mocks ----------------------------------------------------------

const eventHandlers: Record<string, (payload: unknown) => void> = {};

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  worksheetStore: {
    getWorksheetByName: vi.fn(),
  },
  editorWorksheetStore: {
    abortAutoSave: vi.fn(),
    maybeUpdateWorksheet: vi.fn().mockResolvedValue(undefined),
    createWorksheet: vi.fn().mockResolvedValue(undefined),
  },
  sheetContext: {
    getPwdForWorksheet: vi.fn(() => ""),
    getFoldersForWorksheet: vi.fn(() => []),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useSQLEditorEvent", () => ({
  useSQLEditorEvent: vi.fn(
    (event: string, handler: (payload: unknown) => void) => {
      eventHandlers[event] = handler;
    }
  ),
}));

vi.mock("@/store", () => ({
  useWorkSheetStore: vi.fn(() => mocks.worksheetStore),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: typeof mocks.editorWorksheetStore) => unknown
  ) => selector(mocks.editorWorksheetStore),
}));

vi.mock("@/views/sql-editor/Sheet", () => ({
  useSheetContextByView: vi.fn(() => mocks.sheetContext),
}));

vi.mock("@/types", () => ({
  UNKNOWN_ID: 0,
}));

vi.mock("@/utils", () => ({
  extractWorksheetID: vi.fn((worksheet: string) => {
    if (!worksheet) return "0";
    // Return non-zero for worksheets that look like real ones
    return worksheet.includes("/worksheets/") ? "123" : "0";
  }),
}));

// Mock Dialog so it renders into DOM without the Base UI portal
vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({
    children,
    open,
    onOpenChange,
  }: {
    children: React.ReactNode;
    open: boolean;
    onOpenChange?: (next: boolean) => void;
  }) => (
    <div
      data-testid="dialog"
      data-open={String(open)}
      onClick={() => onOpenChange?.(false)}
    >
      {open ? children : null}
    </div>
  ),
  DialogContent: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
  }) => (
    <div data-testid="dialog-content" className={className}>
      {children}
    </div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="dialog-title">{children}</h2>
  ),
}));

// Mock Button — pass through disabled + onClick
vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    variant,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    variant?: string;
  }) => (
    <button data-variant={variant} disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
}));

// Mock Input — standard <input>
vi.mock("@/react/components/ui/input", () => ({
  Input: ({
    value,
    onChange,
    placeholder,
    maxLength,
  }: {
    value: string;
    onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
    placeholder?: string;
    maxLength?: number;
  }) => (
    <input
      data-testid="title-input"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      maxLength={maxLength}
    />
  ),
}));

// Mock FolderForm so we don't re-test tree behavior
vi.mock("./FolderForm", () => ({
  FolderForm: () => <div data-testid="folder-form-mock" />,
}));

// ---- helpers ----------------------------------------------------------------

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
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

const emitSaveSheet = (payload: unknown) => {
  act(() => {
    eventHandlers["save-sheet"]?.(payload);
  });
};

// Tab fixtures
const tabWithoutWorksheet = {
  id: "tab-1",
  title: "Untitled",
  worksheet: undefined,
  connection: { database: "instances/inst1/databases/db1" },
  statement: "SELECT 1",
};

const savedTab = {
  id: "tab-2",
  title: "My Sheet",
  worksheet: "projects/proj1/worksheets/123",
  connection: { database: "instances/inst1/databases/db1" },
  statement: "SELECT 2",
};

let SaveSheetModal: typeof import("./SaveSheetModal").SaveSheetModal;

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.editorWorksheetStore.abortAutoSave.mockReset();
  mocks.editorWorksheetStore.maybeUpdateWorksheet.mockResolvedValue(undefined);
  mocks.editorWorksheetStore.createWorksheet.mockResolvedValue(undefined);
  mocks.sheetContext.getPwdForWorksheet.mockReturnValue("");
  mocks.sheetContext.getFoldersForWorksheet.mockReturnValue([]);
  mocks.worksheetStore.getWorksheetByName.mockReturnValue(undefined);

  ({ SaveSheetModal } = await import("./SaveSheetModal"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

// ---- tests ------------------------------------------------------------------

describe("SaveSheetModal", () => {
  test("1. Unsaved tab shows modal with prefilled title", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    emitSaveSheet({ tab: tabWithoutWorksheet });

    const dialog = container.querySelector("[data-testid='dialog']");
    expect(dialog).not.toBeNull();
    expect(dialog?.getAttribute("data-open")).toBe("true");

    const input = container.querySelector(
      "[data-testid='title-input']"
    ) as HTMLInputElement;
    expect(input).not.toBeNull();
    expect(input.value).toBe("Untitled");

    unmount();
  });

  test("2. Saved tab without editTitle saves silently — no modal, maybeUpdateWorksheet called", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    await act(async () => {
      eventHandlers["save-sheet"]?.({ tab: savedTab });
    });

    const dialog = container.querySelector("[data-testid='dialog']");
    expect(dialog?.getAttribute("data-open")).toBe("false");

    expect(
      mocks.editorWorksheetStore.maybeUpdateWorksheet
    ).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: savedTab.id,
        worksheet: savedTab.worksheet,
        title: savedTab.title,
        database: savedTab.connection.database,
        statement: savedTab.statement,
      })
    );

    unmount();
  });

  test("3. Saved tab with editTitle: true shows modal", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    emitSaveSheet({ tab: savedTab, editTitle: true });

    const dialog = container.querySelector("[data-testid='dialog']");
    expect(dialog?.getAttribute("data-open")).toBe("true");

    unmount();
  });

  test("4. Save button stays enabled with an empty title (worksheet becomes Untitled)", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    // Open modal with a tab that has an empty title — Save is allowed; the
    // worksheet is created with an empty title and the UI renders "Untitled"
    // placeholders for it elsewhere.
    const tabWithEmptyTitle = {
      ...tabWithoutWorksheet,
      title: "",
    };
    emitSaveSheet({ tab: tabWithEmptyTitle });

    const saveButton = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "common.save"
    ) as HTMLButtonElement;
    expect(saveButton).not.toBeNull();
    expect(saveButton.disabled).toBe(false);

    await act(async () => {
      saveButton.click();
    });

    expect(mocks.editorWorksheetStore.createWorksheet).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: tabWithEmptyTitle.id,
        title: "",
      })
    );

    unmount();
  });

  test("5. Clicking Save on unsaved tab calls createWorksheet with correct args", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    // Open modal for unsaved tab with a title set
    emitSaveSheet({ tab: tabWithoutWorksheet });

    const saveButton = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "common.save"
    ) as HTMLButtonElement;
    expect(saveButton).not.toBeNull();
    expect(saveButton.disabled).toBe(false);

    await act(async () => {
      saveButton.click();
    });

    expect(mocks.editorWorksheetStore.abortAutoSave).toHaveBeenCalled();
    expect(mocks.editorWorksheetStore.createWorksheet).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: tabWithoutWorksheet.id,
        title: tabWithoutWorksheet.title,
        statement: tabWithoutWorksheet.statement,
        database: tabWithoutWorksheet.connection.database,
      })
    );

    unmount();
  });
});
