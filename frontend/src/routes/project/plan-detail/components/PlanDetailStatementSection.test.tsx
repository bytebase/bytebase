import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { type Sheet, SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { PlanDetailStatementSection } from "./PlanDetailStatementSection";

const mocks = vi.hoisted(() => ({
  page: undefined as unknown,
  sheets: new Map<string, Sheet>(),
  threadFocus: undefined as
    | { commentName: string; specId: string; lineNumber: number; nonce: number }
    | undefined,
  clearThreadFocus: vi.fn(),
  revealLineInCenter: vi.fn(),
  focus: vi.fn(),
  scrollIntoView: vi.fn(),
  setSelection: vi.fn(),
  setScrollTop: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/monaco", async () => ({
  ...(await vi.importActual<typeof import("@/components/monaco/viewAnchor")>(
    "@/components/monaco/viewAnchor"
  )),
  MonacoEditor: ({
    onChange,
    onReady,
  }: {
    onChange: (value: string) => void;
    onReady: (monaco: unknown, editor: unknown) => void;
  }) => (
    <>
      <button
        onClick={() =>
          onReady({}, {
            getModel: () => ({ getLineCount: () => 3 }),
            revealLineInCenter: mocks.revealLineInCenter,
            focus: mocks.focus,
            getDomNode: () => ({ scrollIntoView: mocks.scrollIntoView }),
            setSelection: mocks.setSelection,
            setScrollTop: mocks.setScrollTop,
            getTopForLineNumber: (lineNumber: number) => lineNumber * 20,
          })
        }
      >
        Editor ready
      </button>
      <button onClick={() => onChange("SELECT 2;\n".repeat(220_000))}>
        Make large draft
      </button>
      <button onClick={() => onChange("SELECT 2;")}>Make small draft</button>
    </>
  ),
  ReadonlyMonaco: ({
    content,
    onReady,
  }: {
    content: string;
    onReady?: (monaco: unknown, editor: unknown) => void;
  }) => (
    <>
      <div data-testid="statement">{content.slice(0, 24)}</div>
      <button
        onClick={() =>
          onReady?.({}, {
            getModel: () => ({}),
            getVisibleRanges: () => [{ startLineNumber: 2, endLineNumber: 3 }],
            getSelection: () => ({
              selectionStartLineNumber: 3,
              selectionStartColumn: 2,
              positionLineNumber: 3,
              positionColumn: 5,
            }),
          })
        }
      >
        Readonly editor ready
      </button>
    </>
  ),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: { databasesByName: object }) => unknown) =>
      selector({ databasesByName: {} }),
    {
      getState: () => ({
        getSheetByName: (name: string) => mocks.sheets.get(name),
        getOrFetchSheetByName: vi.fn(),
      }),
    }
  ),
}));
vi.mock("../shared/stores/usePlanDetailStore", () => ({
  usePlanDetailStore: (selector: (state: unknown) => unknown) =>
    selector({
      threadFocus: mocks.threadFocus,
      clearThreadFocus: mocks.clearThreadFocus,
    }),
}));
vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));
vi.mock("./SchemaEditorSheet", () => ({ SchemaEditorSheet: () => null }));
vi.mock("./threads/StatementThreadsLayer", () => ({
  StatementThreadsLayer: () => null,
}));

afterEach(() => {
  cleanup();
  mocks.sheets.clear();
  mocks.threadFocus = undefined;
  vi.clearAllMocks();
});

test("shows the large-sheet warning as soon as the saved sheet changes", async () => {
  const smallName = `projects/p/sheets/${"a".repeat(64)}`;
  const largeName = `projects/p/sheets/${"b".repeat(64)}`;
  const specOf = (sheet: string) =>
    create(Plan_SpecSchema, {
      id: "spec",
      config: {
        case: "changeDatabaseConfig",
        value: create(Plan_ChangeDatabaseConfigSchema, { sheet }),
      },
    });
  const smallSpec = specOf(smallName);
  const largeSpec = specOf(largeName);
  const addSheet = (name: string, statement: string) => {
    const content = new TextEncoder().encode(statement);
    mocks.sheets.set(
      name,
      create(SheetSchema, { name, content, contentSize: BigInt(content.length) })
    );
  };
  addSheet(smallName, "SELECT 1;");
  addSheet(largeName, "SELECT 2;\n".repeat(220_000));
  const page = {
    currentUser: { name: "users/test" },
    isCreating: false,
    plan: create(PlanSchema, { specs: [smallSpec] }),
    project: undefined,
    readonly: true,
    setEditing: vi.fn(),
  };
  mocks.page = page;

  const { queryByText, rerender, getByTestId } = render(
    <PlanDetailStatementSection spec={smallSpec} />
  );
  expect(queryByText("issue.statement-from-sheet-warning")).toBeNull();

  mocks.page = {
    ...page,
    plan: create(PlanSchema, { specs: [largeSpec] }),
  };
  rerender(<PlanDetailStatementSection spec={largeSpec} />);
  await waitFor(() =>
    expect(queryByText("issue.statement-from-sheet-warning")).not.toBeNull()
  );
  expect(getByTestId("statement").textContent).toContain("SELECT 2;");
});

const renderEditableSpec = (statement: string) => {
  const sheetName = `projects/p/sheets/${"c".repeat(64)}`;
  const spec = create(Plan_SpecSchema, {
    id: "spec",
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, { sheet: sheetName }),
    },
  });
  const content = new TextEncoder().encode(statement);
  mocks.sheets.set(
    sheetName,
    create(SheetSchema, { name: sheetName, content, contentSize: BigInt(content.length) })
  );
  mocks.page = {
    currentUser: { name: "users/test" },
    isCreating: false,
    plan: create(PlanSchema, { creator: "users/test", specs: [spec] }),
    project: { name: "projects/p" },
    readonly: false,
    setEditing: vi.fn(),
  };
  const view = render(<PlanDetailStatementSection spec={spec} />);
  return {
    ...view,
    rerender: () => view.rerender(<PlanDetailStatementSection spec={spec} />),
  };
};

const THREE_LINES = "SELECT 1;\nSELECT 2;\nSELECT 3;";

test("shows a draft-specific warning before saving and removes it below the limit", () => {
  const { getByText, queryByText } = renderEditableSpec("SELECT 1;");
  fireEvent.click(getByText("common.edit"));
  fireEvent.click(getByText("Make large draft"));
  expect(getByText("issue.statement-exceeds-preview-limit")).toBeTruthy();
  expect(queryByText("common.download")).toBeNull();

  fireEvent.click(getByText("Make small draft"));
  expect(queryByText("issue.statement-exceeds-preview-limit")).toBeNull();
});

test("reveals an Activity anchor after the edit editor is ready", () => {
  const { getByText, rerender } = renderEditableSpec(THREE_LINES);
  fireEvent.click(getByText("common.edit"));
  mocks.threadFocus = {
    commentName: "comments/1",
    specId: "spec",
    lineNumber: 3,
    nonce: 7,
  };
  rerender();
  expect(mocks.revealLineInCenter).not.toHaveBeenCalled();

  fireEvent.click(getByText("Editor ready"));
  expect(mocks.setSelection).toHaveBeenCalledWith({
    selectionStartLineNumber: 3,
    selectionStartColumn: 1,
    positionLineNumber: 3,
    positionColumn: 1,
  });
  expect(mocks.revealLineInCenter).toHaveBeenCalledWith(3);
  expect(mocks.focus).toHaveBeenCalled();
  expect(mocks.scrollIntoView).toHaveBeenCalledWith({ block: "nearest" });
  expect(mocks.clearThreadFocus).toHaveBeenCalledWith(7);

  mocks.threadFocus = {
    commentName: "comments/2",
    specId: "spec",
    lineNumber: 99,
    nonce: 8,
  };
  rerender();
  expect(mocks.revealLineInCenter).toHaveBeenLastCalledWith(3);
  expect(mocks.clearThreadFocus).toHaveBeenCalledWith(8);
});

test("keeps the reading position and cursor when entering edit mode", () => {
  const { getByText } = renderEditableSpec(THREE_LINES);
  fireEvent.click(getByText("Readonly editor ready"));
  fireEvent.click(getByText("common.edit"));
  fireEvent.click(getByText("Editor ready"));

  expect(mocks.setSelection).toHaveBeenCalledWith({
    selectionStartLineNumber: 3,
    selectionStartColumn: 2,
    positionLineNumber: 3,
    positionColumn: 5,
  });
  expect(mocks.setScrollTop).toHaveBeenCalledWith(40);
  expect(mocks.revealLineInCenter).not.toHaveBeenCalled();
  expect(mocks.focus).toHaveBeenCalled();
  expect(mocks.scrollIntoView).not.toHaveBeenCalled();
});

test("an Activity anchor wins over the reading position captured by Edit", () => {
  const { getByText, rerender } = renderEditableSpec(THREE_LINES);
  fireEvent.click(getByText("Readonly editor ready"));
  fireEvent.click(getByText("common.edit"));
  mocks.threadFocus = {
    commentName: "comments/1",
    specId: "spec",
    lineNumber: 1,
    nonce: 3,
  };
  rerender();
  fireEvent.click(getByText("Editor ready"));

  expect(mocks.setSelection).toHaveBeenCalledTimes(1);
  expect(mocks.revealLineInCenter).toHaveBeenCalledWith(1);
  expect(mocks.setScrollTop).not.toHaveBeenCalled();
  expect(mocks.clearThreadFocus).toHaveBeenCalledWith(3);
});
