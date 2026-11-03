import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, waitFor, within } from "@testing-library/react";
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
  getRelease: vi.fn(),
  openedFile: undefined as unknown,
}));

vi.mock("@/api", () => ({
  planServiceClientConnect: {},
  sheetServiceClientConnect: {},
  releaseServiceClientConnect: { getRelease: mocks.getRelease },
}));
vi.mock("./release/ReleaseFileViewer", () => ({
  useOpenReleaseFile: () => (target: unknown) => {
    mocks.openedFile = target;
  },
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
vi.mock("./SchemaEditorSheet", () => ({ SchemaEditorSheet: () => null }));
vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children, open }: { children: React.ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => (
    <div role="dialog">{children}</div>
  ),
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => (
    <p>{children}</p>
  ),
  AlertDialogFooter: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
  }) => (
    <div className={className}>{children}</div>
  ),
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2>{children}</h2>
  ),
}));
vi.mock("./threads/StatementThreadsLayer", () => ({
  StatementThreadsLayer: () => null,
}));

afterEach(() => {
  cleanup();
  for (const name of mocks.localSheetNames) removeLocalSheet(name);
  mocks.localSheetNames = [];
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

const dispatchPlanDraft = (database: string, sql: string) => {
  const detail: AgentPlanDraftEventDetail = {
    project: "projects/p",
    database,
    sql,
    handled: false,
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
  expect(trigger.querySelector("svg")).toBeNull();
  expect(trigger).toHaveClass("max-w-full", "whitespace-normal");
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

  const event = dispatchPlanDraft(database, "CREATE TABLE notes (id INT);");
  expect(event.handled).toBe(true);
  expect(getSheetStatement(sheet)).toBe("CREATE TABLE notes (id INT);");
  expect(getByTestId("editing-statement").textContent).toBe(
    "CREATE TABLE notes (id INT);"
  );
});

test("requires an explicit append or replace choice when the Plan already has SQL", () => {
  const database = "instances/mysql/databases/app";
  const { sheet, getByRole, queryByRole, getByTestId } =
    renderPlanCreationSpec(database, "SELECT 1;");

  dispatchPlanDraft(database, "CREATE TABLE notes (id INT);");
  expect(getByRole("dialog")).toBeTruthy();
  expect(
    getByRole("button", { name: "agent.plan-draft.replace-sql" }).parentElement
  ).toHaveClass("flex-wrap", "gap-y-2");
  expect(mocks.minimizeAgent).toHaveBeenCalledOnce();
  fireEvent.click(getByRole("button", { name: "agent.plan-draft.append-sql" }));
  expect(getSheetStatement(sheet)).toBe(
    "SELECT 1;\n\nCREATE TABLE notes (id INT);"
  );
  expect(queryByRole("dialog")).toBeNull();

  dispatchPlanDraft(database, "ALTER TABLE notes ADD COLUMN name text;");
  fireEvent.click(getByRole("button", { name: "agent.plan-draft.replace-sql" }));
  expect(getSheetStatement(sheet)).toBe(
    "ALTER TABLE notes ADD COLUMN name text;"
  );
  expect(getByTestId("editing-statement").textContent).toBe(
    "ALTER TABLE notes ADD COLUMN name text;"
  );
});

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

const releaseSpec = create(Plan_SpecSchema, {
  id: "spec",
  config: {
    case: "changeDatabaseConfig",
    value: create(Plan_ChangeDatabaseConfigSchema, {
      release: "projects/p/releases/r1",
    }),
  },
});
const releasePage = {
  currentUser: { name: "users/test" },
  isCreating: false,
  readonly: true,
  project: { name: "projects/p" },
  plan: create(PlanSchema, { name: "projects/p/plans/1", specs: [releaseSpec] }),
  setEditing: vi.fn(),
};
const fileOf = (version: string, path = `migrations/${version}.sql`) => ({
  version,
  path,
  sheet: `projects/p/sheets/${version}`,
});
const renderRail = async (files: ReturnType<typeof fileOf>[], type = 1) => {
  mocks.page = releasePage;
  mocks.getRelease.mockResolvedValue({ name: "projects/p/releases/r1", type, files });
  const view = render(<PlanDetailStatementSection spec={releaseSpec} />);
  await waitFor(() => expect(view.getByTestId("release-changes")).toBeDefined());
  return view;
};
const rowVersions = (view: ReturnType<typeof render>) =>
  view.queryAllByTestId("release-change-row").map((row) => row.getAttribute("data-version"));

test("the Changes list puts the newest version on top whatever its shape", async () => {
  for (const versions of [
    ["202610051200", "202610051400", "202610051530"],
    ["001", "002", "010"],
    ["1.2.9", "1.10.0", "1.10.2"],
    // The executor's order: equal leading segments, the shorter version first.
    ["1", "1.0", "1.0.1"],
  ]) {
    const view = await renderRail(versions.map((v) => fileOf(v)));
    expect(rowVersions(view)).toEqual([...versions].reverse());
    cleanup();
  }
});

test("timestamp versions group by day for the newest group and by month for the rest, newest open", async () => {
  const view = await renderRail(
    ["202608011200", "202609150900", "202609200900", "202610050900", "202610051530"].map((v) => fileOf(v))
  );
  const groups = view.getAllByTestId("release-change-group");
  expect(groups.map((g) => [g.getAttribute("data-group"), g.getAttribute("aria-expanded")])).toEqual([
    ["2026-10-05", "true"],
    ["2026-09", "false"],
    ["2026-08", "false"],
  ]);
  expect(groups[1].textContent).toContain("2");
  expect(rowVersions(view)).toEqual(["202610051530", "202610050900"]);
  fireEvent.click(groups[1]);
  expect(rowVersions(view)).toContain("202609200900");
});

test("a group shows 12 rows, loads 12 more or all in place, and Show fewer returns to 12", async () => {
  const versions = Array.from({ length: 30 }, (_, i) => String(i + 1).padStart(3, "0"));
  const view = await renderRail(versions.map((v) => fileOf(v)));
  const load = within(view.getByTestId("release-change-load"));
  expect(rowVersions(view)).toHaveLength(12);
  expect(view.queryByRole("button", { name: "release.changes.show-fewer" })).toBeNull();
  fireEvent.click(load.getByRole("button", { name: "release.changes.load-more" }));
  expect(rowVersions(view)).toHaveLength(24);
  // Six left: "Load 6 more" covers them, so there is no Load all.
  expect(view.queryByRole("button", { name: "release.changes.load-all" })).toBeNull();
  fireEvent.click(view.getByRole("button", { name: "release.changes.show-fewer" }));
  expect(rowVersions(view)).toHaveLength(12);
  fireEvent.click(view.getByRole("button", { name: "release.changes.load-all" }));
  expect(rowVersions(view)).toHaveLength(30);
  expect(view.queryByRole("button", { name: "release.changes.load-more" })).toBeNull();
});

test("over a hundred counter or dotted versions are one flat list with no group headers", async () => {
  for (const versions of [
    Array.from({ length: 250 }, (_, i) => String(i + 1).padStart(4, "0")),
    Array.from({ length: 119 }, (_, i) => `3.${Math.floor(i / 12)}.${String(i % 12).padStart(4, "0")}`),
  ]) {
    const view = await renderRail(versions.map((v) => fileOf(v)));
    expect(view.queryAllByTestId("release-change-group")).toHaveLength(0);
    expect(rowVersions(view)).toHaveLength(12);
    expect(rowVersions(view)[0]).toBe(versions.at(-1));
    fireEvent.click(view.getByRole("button", { name: "release.changes.load-all" }));
    expect(rowVersions(view)).toHaveLength(versions.length);
    cleanup();
  }
});

test("a backfilled low version lands in its own old group, not on top", async () => {
  const view = await renderRail(
    ["202607010000", "202610051500", "202610051530"].map((v) => fileOf(v))
  );
  expect(rowVersions(view)).toEqual(["202610051530", "202610051500"]);
  expect(view.getAllByTestId("release-change-group").map((g) => g.getAttribute("data-group"))).toEqual([
    "2026-10-05",
    "2026-07",
  ]);
});

test("a row shows the file name, its folder muted and the version, and opens the viewer from anywhere on it", async () => {
  const view = await renderRail([
    fileOf("3.10.0007", "migrations/3.10/3.10.0007_remove_exporter_role.sql"),
    fileOf("3.10.0008", "3.10.0008_top_level.sql"),
  ]);
  const [rootFile, nestedFile] = view.getAllByTestId("release-change-row");
  const part = (row: HTMLElement, id: string) => within(row).queryByTestId(id)?.textContent;
  expect(part(nestedFile, "release-change-name")).toBe("3.10.0007_remove_exporter_role.sql");
  expect(part(nestedFile, "release-change-folder")).toBe("migrations/3.10");
  expect(part(nestedFile, "release-change-version")).toBe("3.10.0007");
  // A file at the repository root has no folder to show.
  expect(part(rootFile, "release-change-name")).toBe("3.10.0008_top_level.sql");
  expect(part(rootFile, "release-change-folder")).toBeUndefined();

  fireEvent.click(within(nestedFile).getByTestId("release-change-version"));
  expect((mocks.openedFile as { file: { path: string } }).file.path).toBe(
    "migrations/3.10/3.10.0007_remove_exporter_role.sql"
  );
  expect((mocks.openedFile as { context?: unknown }).context).toBeUndefined();
});

test("a declarative release is one line: the schema at its version, with Open", async () => {
  const view = await renderRail([fileOf("20261005", "schema/schema.sql")], 2);
  expect(view.queryAllByTestId("release-change-row")).toHaveLength(0);
  const line = view.getByTestId("release-changes");
  expect(line.textContent).toContain("schema/schema.sql");
  expect(line.textContent).toContain("20261005");
  fireEvent.click(view.getByRole("button", { name: "common.open" }));
  expect((mocks.openedFile as { file: { path: string } }).file.path).toBe("schema/schema.sql");
});
