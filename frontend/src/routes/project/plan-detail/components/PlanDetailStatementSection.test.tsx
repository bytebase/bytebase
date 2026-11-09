import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { AGENT_PLAN_DRAFT_EVENT } from "@/lib/plan/agentDraftHandoff";
import type { AgentPlanDraftEventDetail } from "@/lib/plan/agentDraftHandoff";
import { openAgentWithPrompt } from "@/modules/agent/logic/instanceConnectionHelp";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { type Sheet, SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { getSheetStatement } from "@/utils/v1/sheet";
import {
  getLocalSheetByName,
  removeLocalSheet,
  setLocalSheetStatement,
} from "../utils/localSheet";
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
  minimizeAgent: vi.fn(),
  openAgentWithPrompt: vi.fn(),
  localSheetNames: [] as string[],
  failLocalWrite: false,
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
    content,
    onChange,
    onReady,
  }: {
    onChange: (value: string) => void;
    onReady: (monaco: unknown, editor: unknown) => void;
    content: string;
  }) => (
    <>
      <div data-testid="editing-statement">{content}</div>
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
vi.mock("@/stores", () => ({
  environmentNamePrefix: "environments/",
  pushNotification: vi.fn(),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: { databasesByName: object }) => unknown) =>
      selector({ databasesByName: {} }),
    {
      getState: () => ({
        getSheetByName: (name: string) => mocks.sheets.get(name),
        getOrFetchSheetByName: vi.fn(),
      batchGetOrFetchDatabases: vi.fn(),
      }),
    }
  ),
}));
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: mocks.openAgentWithPrompt,
}));
vi.mock("@/modules/agent/store/agent", () => ({
  useAgentStore: {
    getState: () => ({ visible: true, minimize: mocks.minimizeAgent }),
  },
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
vi.mock("../utils/localSheet", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../utils/localSheet")>();
  return {
    ...actual,
    setLocalSheetStatement: (...args: Parameters<typeof actual.setLocalSheetStatement>) => {
      if (mocks.failLocalWrite) throw new Error("Local sheet unavailable");
      return actual.setLocalSheetStatement(...args);
    },
  };
});
vi.mock("./SchemaEditorSheet", () => ({ SchemaEditorSheet: () => null }));
vi.mock("./threads/StatementThreadsLayer", () => ({
  StatementThreadsLayer: () => null,
}));

afterEach(() => {
  cleanup();
  for (const name of mocks.localSheetNames) removeLocalSheet(name);
  mocks.localSheetNames = [];
  mocks.sheets.clear();
  mocks.threadFocus = undefined;
  mocks.failLocalWrite = false;
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

const renderPlanCreationSpec = (
  database: string,
  initialSQL = "",
  targets = [database]
) => {
  const sheetName = "projects/p/sheets/-agent-draft-test";
  mocks.localSheetNames.push(sheetName);
  const sheet = getLocalSheetByName(sheetName);
  setLocalSheetStatement(sheet, initialSQL);
  const spec = create(Plan_SpecSchema, {
    id: "new-spec",
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet: sheetName,
        targets,
      }),
    },
  });
  mocks.page = {
    currentUser: { name: "users/test" },
    isCreating: true,
    plan: create(PlanSchema, { specs: [] }),
    project: { name: "projects/p" },
    readonly: false,
    setEditing: vi.fn(),
  };
  return {
    ...render(<PlanDetailStatementSection spec={spec} />),
    sheet,
  };
};

const dispatchPlanDraft = (
  database: string,
  sql: string,
  onApplied?: () => void
) => {
  const detail: AgentPlanDraftEventDetail = {
    project: "projects/p",
    database,
    sql,
    handled: false,
    onApplied,
  };
  act(() => {
    window.dispatchEvent(
      new CustomEvent<AgentPlanDraftEventDetail>(AGENT_PLAN_DRAFT_EVENT, {
        detail,
      })
    );
  });
  return detail;
};

test("opens the Agent for the selected database and applies its DDL to this Plan", () => {
  const database = "instances/mysql/databases/app";
  const { sheet, getByRole, getByTestId } = renderPlanCreationSpec(database);
  const trigger = getByRole("button", {
    name: "agent.plan-assistance.ask-draft-sql",
  });
  expect(trigger.querySelector(".lucide-bot")).not.toBeNull();
  expect(trigger).toHaveClass("h-6", "bg-transparent", "text-accent");
  expect(trigger).not.toHaveClass("whitespace-normal", "h-auto");
  expect(trigger.querySelector(".truncate")).toHaveTextContent(
    "agent.plan-assistance.ask-draft-sql"
  );
  expect(trigger.parentElement).toHaveClass("w-full", "max-w-full", "flex-wrap");
  expect(trigger.parentElement?.parentElement).toHaveClass("flex-wrap");

  fireEvent.click(trigger);
  expect(openAgentWithPrompt).toHaveBeenCalledWith(
    expect.stringContaining("current Plan"),
    [
      expect.objectContaining({
        kind: "database",
        name: database,
        project: "projects/p",
      }),
    ]
  );

  const onApplied = vi.fn();
  const event = dispatchPlanDraft(
    database,
    "CREATE TABLE notes (id INT);",
    onApplied
  );
  expect(event.handled).toBe(true);
  expect(getSheetStatement(sheet)).toBe("CREATE TABLE notes (id INT);");
  expect(getByTestId("editing-statement").textContent).toBe(
    "CREATE TABLE notes (id INT);"
  );
  expect(onApplied).toHaveBeenCalledOnce();
  expect(mocks.minimizeAgent).not.toHaveBeenCalled();
});

test("requires an explicit append or replace choice when the Plan already has SQL", () => {
  const database = "instances/mysql/databases/app";
  const { sheet, getByRole, queryByRole, getByTestId } =
    renderPlanCreationSpec(database, "SELECT 1;");

  const onApplied = vi.fn();
  const request = dispatchPlanDraft(
    database,
    "CREATE TABLE notes (id INT);",
    onApplied
  );
  expect(request.handled).toBe(true);
  const dialog = getByRole("alertdialog");
  expect(dialog.closest("[data-bb-layer-family]")).toHaveAttribute(
    "data-bb-layer-family",
    "agent"
  );
  expect(
    getByRole("button", { name: "agent.plan-draft.replace-sql" }).parentElement
  ).toHaveClass("flex-wrap", "gap-y-2");
  expect(mocks.minimizeAgent).not.toHaveBeenCalled();
  expect(onApplied).not.toHaveBeenCalled();
  fireEvent.click(getByRole("button", { name: "agent.plan-draft.append-sql" }));
  expect(getSheetStatement(sheet)).toBe(
    "SELECT 1;\n\nCREATE TABLE notes (id INT);"
  );
  expect(queryByRole("alertdialog")).toBeNull();
  expect(onApplied).toHaveBeenCalledOnce();

  const onReplaced = vi.fn();
  dispatchPlanDraft(
    database,
    "ALTER TABLE notes ADD COLUMN name text;",
    onReplaced
  );
  expect(onReplaced).not.toHaveBeenCalled();
  fireEvent.click(getByRole("button", { name: "agent.plan-draft.replace-sql" }));
  expect(getSheetStatement(sheet)).toBe(
    "ALTER TABLE notes ADD COLUMN name text;"
  );
  expect(getByTestId("editing-statement").textContent).toBe(
    "ALTER TABLE notes ADD COLUMN name text;"
  );
  expect(onReplaced).toHaveBeenCalledOnce();
});

test("does not report an applied draft when confirmation is canceled", () => {
  const database = "instances/mysql/databases/app";
  const { sheet, getByRole, queryByRole } = renderPlanCreationSpec(
    database,
    "SELECT 1;"
  );
  const onApplied = vi.fn();
  dispatchPlanDraft(database, "CREATE TABLE notes (id INT);", onApplied);

  fireEvent.click(getByRole("button", { name: "common.cancel" }));
  expect(queryByRole("alertdialog")).toBeNull();
  expect(getSheetStatement(sheet)).toBe("SELECT 1;");
  expect(onApplied).not.toHaveBeenCalled();
  expect(mocks.minimizeAgent).not.toHaveBeenCalled();
});

test("does not report an invalid draft as applied", () => {
  const database = "instances/mysql/databases/app";
  const { sheet } = renderPlanCreationSpec(database);
  const onApplied = vi.fn();
  const request = dispatchPlanDraft(database, " ", onApplied);

  expect(request.handled).toBe(true);
  expect(getSheetStatement(sheet)).toBe("");
  expect(onApplied).not.toHaveBeenCalled();
  expect(mocks.minimizeAgent).not.toHaveBeenCalled();
});

test.each(["", "SELECT 1;"])(
  "keeps the Agent open if applying SQL to %j fails",
  (initialSQL) => {
    const database = "instances/mysql/databases/app";
    const { sheet, getByRole, queryByRole } = renderPlanCreationSpec(
      database,
      initialSQL
    );
    const onApplied = vi.fn();
    mocks.failLocalWrite = true;

    dispatchPlanDraft(database, "CREATE TABLE notes (id INT);", onApplied);
    if (initialSQL) {
      fireEvent.click(
        getByRole("button", { name: "agent.plan-draft.replace-sql" })
      );
      expect(queryByRole("alertdialog")).not.toBeNull();
    }

    expect(getSheetStatement(sheet)).toBe(initialSQL);
    expect(onApplied).not.toHaveBeenCalled();
    expect(mocks.minimizeAgent).not.toHaveBeenCalled();
  }
);

test("does not offer or apply a draft when the Plan has an invalid extra target", () => {
  const database = "instances/mysql/databases/app";
  const { sheet, queryByRole } = renderPlanCreationSpec(
    database,
    "",
    [database, ""]
  );

  expect(
    queryByRole("button", { name: "agent.plan-assistance.ask-draft-sql" })
  ).toBeNull();
  const event = dispatchPlanDraft(database, "CREATE TABLE notes (id INT);");
  expect(event.handled).toBe(false);
  expect(getSheetStatement(sheet)).toBe("");
  expect(
    queryByRole("button", { name: "agent.plan-draft.replace-sql" })
  ).toBeNull();
});

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
