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
  appStore: {
    getSavedQueryByName: vi.fn(),
  },
  editorSavedQueryStore: {
    abortAutoSave: vi.fn(),
    maybeUpdateSavedQuery: vi.fn().mockResolvedValue(undefined),
    createSavedQuery: vi.fn().mockResolvedValue(undefined),
  },
  sheetContext: {
    getPwdForSavedQuery: vi.fn(() => ""),
    getFoldersForSavedQuery: vi.fn(() => []),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/hooks/useSQLEditorEvent", () => ({
  useSQLEditorEvent: vi.fn(
    (event: string, handler: (payload: unknown) => void) => {
      eventHandlers[event] = handler;
    }
  ),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks.appStore },
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: (
    selector: (s: typeof mocks.editorSavedQueryStore) => unknown
  ) => selector(mocks.editorSavedQueryStore),
}));

vi.mock("@/modules/sql-editor/model/Sheet", () => ({
  useSheetContextByView: vi.fn(() => mocks.sheetContext),
}));

vi.mock("@/types", () => ({
  UNKNOWN_ID: 0,
}));

vi.mock("@/utils", () => ({
  extractSavedQueryID: vi.fn((savedQuery: string) => {
    if (!savedQuery) return "0";
    // Return non-zero for saved queries that look like real ones
    return savedQuery.includes("/savedQueries/") ? "123" : "0";
  }),
}));

// Mock Dialog so it renders into DOM without the Base UI portal
vi.mock("@/components/ui/dialog", () => ({
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
vi.mock("@/components/ui/button", () => ({
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
vi.mock("@/components/ui/input", () => ({
  Input: ({
    value,
    onChange,
    placeholder,
    maxLength,
    autoComplete,
  }: {
    value: string;
    onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
    placeholder?: string;
    maxLength?: number;
    autoComplete?: string;
  }) => (
    <input
      data-testid="title-input"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      maxLength={maxLength}
      autoComplete={autoComplete}
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
const tabWithoutSavedQuery = {
  id: "tab-1",
  title: "Untitled",
  savedQuery: undefined,
  connection: { database: "instances/inst1/databases/db1" },
  statement: "SELECT 1",
};

const savedTab = {
  id: "tab-2",
  title: "My Sheet",
  savedQuery: "projects/proj1/savedQueries/123",
  connection: { database: "instances/inst1/databases/db1" },
  statement: "SELECT 2",
};

let SaveSheetModal: typeof import("./SaveSheetModal").SaveSheetModal;

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.editorSavedQueryStore.abortAutoSave.mockReset();
  mocks.editorSavedQueryStore.maybeUpdateSavedQuery.mockResolvedValue(undefined);
  mocks.editorSavedQueryStore.createSavedQuery.mockResolvedValue(undefined);
  mocks.sheetContext.getPwdForSavedQuery.mockReturnValue("");
  mocks.sheetContext.getFoldersForSavedQuery.mockReturnValue([]);
  mocks.appStore.getSavedQueryByName.mockReturnValue(undefined);

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

    emitSaveSheet({ tab: tabWithoutSavedQuery });

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

  test("title input disables browser autocomplete", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    emitSaveSheet({ tab: tabWithoutSavedQuery });

    const input = container.querySelector(
      "[data-testid='title-input']"
    ) as HTMLInputElement;
    expect(input).not.toBeNull();
    expect(input.autocomplete).toBe("off");

    unmount();
  });

  test("2. Saved tab without editTitle saves silently — no modal, maybeUpdateSavedQuery called", async () => {
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
      mocks.editorSavedQueryStore.maybeUpdateSavedQuery
    ).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: savedTab.id,
        savedQuery: savedTab.savedQuery,
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

  test("4. Save button stays enabled with an empty title (saved query becomes Untitled)", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    // Open modal with a tab that has an empty title — Save is allowed; the
    // saved query is created with an empty title and the UI renders "Untitled"
    // placeholders for it elsewhere.
    const tabWithEmptyTitle = {
      ...tabWithoutSavedQuery,
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

    expect(mocks.editorSavedQueryStore.createSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: tabWithEmptyTitle.id,
        title: "",
      })
    );

    unmount();
  });

  test("5. Clicking Save on unsaved tab calls createSavedQuery with correct args", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    // Open modal for unsaved tab with a title set
    emitSaveSheet({ tab: tabWithoutSavedQuery });

    const saveButton = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "common.save"
    ) as HTMLButtonElement;
    expect(saveButton).not.toBeNull();
    expect(saveButton.disabled).toBe(false);

    await act(async () => {
      saveButton.click();
    });

    expect(mocks.editorSavedQueryStore.abortAutoSave).toHaveBeenCalled();
    expect(mocks.editorSavedQueryStore.createSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: tabWithoutSavedQuery.id,
        title: tabWithoutSavedQuery.title,
        statement: tabWithoutSavedQuery.statement,
        database: tabWithoutSavedQuery.connection.database,
      })
    );

    unmount();
  });

  test("6. Saved but untitled tab shows modal instead of saving silently", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SaveSheetModal />
    );
    render();

    // SavedQuery exists (has a name) but its title is empty — a manual save
    // should prompt for a title rather than silently re-persisting Untitled.
    const savedUntitledTab = { ...savedTab, title: "" };
    emitSaveSheet({ tab: savedUntitledTab });

    const dialog = container.querySelector("[data-testid='dialog']");
    expect(dialog?.getAttribute("data-open")).toBe("true");
    expect(
      mocks.editorSavedQueryStore.maybeUpdateSavedQuery
    ).not.toHaveBeenCalled();

    unmount();
  });
});
