import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { Plan_SpecSchema, type PlanCheckRun, type PlanCheckRun_Result } from "@/types/proto-es/v1/plan_service_pb";
import { CheckReleaseResponse_CheckResultSchema } from "@/types/proto-es/v1/release_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { openAgentWithPrompt } from "@/modules/agent/logic/instanceConnectionHelp";
import { getLocalSheetByName, removeLocalSheet, setLocalSheetStatement } from "../utils/localSheet";
import { PlanDetailDraftChecks } from "./PlanDetailDraftChecks";

const mocks = vi.hoisted(() => ({
  page: {} as Record<string, unknown>,
  databases: {} as Record<string, Database>,
  checkRelease: vi.fn(),
  diagnosticHelp: vi.fn(),
}));

vi.mock("@/api", () => ({ releaseServiceClientConnect: { checkRelease: mocks.checkRelease } }));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => {
  const getState = () => ({ databasesByName: mocks.databases, dbGroupsByName: {}, workspaceResourceName: () => "workspaces/example", currentUser: { name: "users/member@example.com" } });
  return { useAppStore: Object.assign((select: (state: unknown) => unknown) => select(getState()), { getState }) };
});
vi.mock("../shell/PlanDetailContext", () => ({ usePlanDetailContext: () => mocks.page }));
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({ openAgentWithPrompt: vi.fn() }));
vi.mock("@/modules/agent/logic/diagnosticHelp", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/modules/agent/logic/diagnosticHelp")>();
  return { ...original, openAgentDiagnosticHelp: (context: Parameters<typeof original.openAgentDiagnosticHelp>[0]) => {
    mocks.diagnosticHelp(context);
    return original.openAgentDiagnosticHelp(context);
  } };
});
vi.mock("./PlanTargetDisplay", () => ({ PlanTargetDisplay: () => null }));
vi.mock("@/components/plan-check/PlanCheckSection", () => ({
  PlanCheckSection: ({ planCheckRuns, renderResultAction, onRun, runDisabled }: {
    planCheckRuns: PlanCheckRun[];
    renderResultAction?: (result: PlanCheckRun_Result) => ReactNode;
    onRun: () => void;
    runDisabled: boolean;
  }) => <div>
    <button type="button" disabled={runDisabled} onClick={onRun}>Run checks</button>
    {planCheckRuns.flatMap((run) => run.results).map((result, index) => <div key={index}>{renderResultAction?.(result)}</div>)}
  </div>,
}));

const databaseName = "instances/main/databases/books";
const sheetName = "projects/example/sheets/-101";
const sql = "ALTER TABLE books ADD COLUMN note TEXT;";
const spec = () => create(Plan_SpecSchema, {
  id: "spec-1",
  config: { case: "changeDatabaseConfig", value: { sheet: sheetName, targets: [databaseName] } },
});
const results = (status = Advice_Level.WARNING) => [create(CheckReleaseResponse_CheckResultSchema, {
  target: databaseName,
  advices: [{ status, code: 101, title: "Nullable column", content: "Column note should not be nullable", startPosition: { line: 1, column: 1 } }],
})];
const checkedContent = () => getLocalSheetByName(sheetName).content;
const snapshot = () => {
  const prompt = vi.mocked(openAgentWithPrompt).mock.lastCall![0];
  return JSON.parse(prompt.slice(prompt.indexOf("\n") + 1));
};

beforeEach(() => {
  vi.clearAllMocks();
  removeLocalSheet(sheetName);
  setLocalSheetStatement(getLocalSheetByName(sheetName), sql);
  mocks.page = { projectId: "example", pageKey: "create-example", isCreating: true, plan: { name: "projects/example/plans/create" } };
  mocks.databases = { [databaseName]: create(DatabaseSchema$, { name: databaseName, project: "projects/example" }) };
  mocks.checkRelease.mockResolvedValue({ results: results() });
});
afterEach(() => { cleanup(); removeLocalSheet(sheetName); });

test.each([Advice_Level.WARNING, Advice_Level.ERROR])("launches a Database-bound diagnosis using checked draft SQL for status %s", (status) => {
  const props = { selectedSpec: spec(), checkedContent: checkedContent(), checkResults: results(status), onCheckResultsChange: vi.fn() };
  render(<PlanDetailDraftChecks {...props} />);
  expect(openAgentWithPrompt).not.toHaveBeenCalled();

  fireEvent.click(screen.getByRole("button", { name: "Ask Agent" }));

  expect(snapshot()).toMatchObject({
    resource: { kind: "database", name: databaseName, project: "projects/example" },
    kind: "sql-check",
    sql: { source: "draft", text: sql, sheet: sheetName },
  });
  expect(snapshot().sql.sha256).toBe("3d4b135093add2d059d776ddd5bf4a2c2c787356f93ff224c77d8323143086f0");
  expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toEqual({
    specId: "spec-1",
    sha256: snapshot().sql.sha256,
    target: databaseName,
  });
  expect(snapshot().sourceId).toContain("spec-1");
  expect(vi.mocked(openAgentWithPrompt).mock.lastCall![1]).toHaveLength(1);
  expect(vi.mocked(openAgentWithPrompt).mock.lastCall![3]?.capability?.resource.kind).toBe("database");
});

test("hides diagnostic help when checked SQL is unavailable", () => {
  render(<PlanDetailDraftChecks selectedSpec={spec()} checkResults={results()} onCheckResultsChange={vi.fn()} />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});

test("preserves the exact checked bytes when SQL begins with a UTF-8 BOM", () => {
  const original = "\uFEFFSELECT 1;";
  setLocalSheetStatement(getLocalSheetByName(sheetName), original);
  const props = { selectedSpec: spec(), checkedContent: checkedContent(), checkResults: results(), onCheckResultsChange: vi.fn() };
  render(<PlanDetailDraftChecks {...props} />);
  fireEvent.click(screen.getByRole("button", { name: "Ask Agent" }));

  expect(snapshot().sql.text).toBe(original);
  expect(snapshot().sql.sha256).toBe("a35432496aa3b58f1818bc547ae78c87d1a5e702dcc7f28e2e1fdb38ff3bbf52");
});

test("hides diagnostic help instead of replacing invalid checked UTF-8", () => {
  getLocalSheetByName(sheetName).content = new Uint8Array([0xff, 0x53, 0x51, 0x4c]);
  const props = { selectedSpec: spec(), checkedContent: checkedContent(), checkResults: results(), onCheckResultsChange: vi.fn() };
  render(<PlanDetailDraftChecks {...props} />);

  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});

test("does not substitute new editor SQL for a displayed draft check", () => {
  const props = { selectedSpec: spec(), checkedContent: checkedContent(), checkResults: results(), onCheckResultsChange: vi.fn() };
  const view = render(<PlanDetailDraftChecks {...props} />);
  expect(screen.getByRole("button", { name: "Ask Agent" })).toBeInTheDocument();

  setLocalSheetStatement(getLocalSheetByName(sheetName), "DROP TABLE books;");
  view.rerender(<PlanDetailDraftChecks {...props} />);

  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
  expect(openAgentWithPrompt).not.toHaveBeenCalled();
});

test("saved inline drafts opt in to diagnosis with the exact editor content", () => {
  mocks.page.isCreating = false;
  const editorSql = "DELETE FROM books;";
  const content = new TextEncoder().encode(editorSql);
  const props = { selectedSpec: spec(), content, checkedContent: content.slice(), checkResults: results(), onCheckResultsChange: vi.fn() };
  const view = render(<PlanDetailDraftChecks {...props} />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();

  view.rerender(<PlanDetailDraftChecks {...props} canDiagnose />);
  fireEvent.click(screen.getByRole("button", { name: "Ask Agent" }));

  expect(snapshot().sql.text).toBe(editorSql);
  expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toEqual({
    specId: "spec-1",
    sha256: snapshot().sql.sha256,
    target: databaseName,
  });

  view.rerender(<PlanDetailDraftChecks {...props} content={new TextEncoder().encode("SELECT 1;")} canDiagnose />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});

test("explicitly disabling diagnosis also applies on create pages", () => {
  render(<PlanDetailDraftChecks selectedSpec={spec()} checkedContent={checkedContent()} checkResults={results()} onCheckResultsChange={vi.fn()} canDiagnose={false} />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});

test("runs inline checks with override bytes rather than the local sheet", async () => {
  const content = new TextEncoder().encode("\uFEFFDELETE FROM books;");
  const onCheckResultsChange = vi.fn();
  render(<PlanDetailDraftChecks selectedSpec={spec()} content={content} onCheckResultsChange={onCheckResultsChange} />);

  fireEvent.click(screen.getByRole("button", { name: "Run checks" }));

  await waitFor(() => expect(onCheckResultsChange).toHaveBeenCalledWith(content, results()));
  expect(mocks.checkRelease.mock.lastCall?.[0].release.files[0].statement).toBe(content);
});

test("an empty content override does not fall back to the nonempty local sheet", () => {
  render(<PlanDetailDraftChecks selectedSpec={spec()} content={new Uint8Array()} checkedContent={checkedContent()} checkResults={results()} onCheckResultsChange={vi.fn()} />);

  expect(screen.getByRole("button", { name: "Run checks" })).toBeDisabled();
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});

test.each(["missing-database", "different-project", "different-target", "different-route", "success", "no-results"])("suppresses unsafe or non-problem draft diagnosis: %s", (change) => {
  const selectedSpec = spec();
  const checkResults = results();
  if (change === "missing-database") mocks.databases = {};
  if (change === "different-project") mocks.databases[databaseName] = create(DatabaseSchema$, { name: databaseName, project: "projects/other" });
  if (change === "different-target" && selectedSpec.config.case === "changeDatabaseConfig") selectedSpec.config.value.targets = ["instances/main/databases/other"];
  if (change === "different-route") mocks.page.projectId = "other";
  if (change === "success") checkResults[0].advices[0].status = Advice_Level.SUCCESS;
  const props = { selectedSpec, checkedContent: checkedContent(), checkResults: change === "no-results" ? [] : checkResults, onCheckResultsChange: vi.fn() };
  render(<PlanDetailDraftChecks {...props} />);
  expect(screen.queryByRole("button", { name: "Ask Agent" })).toBeNull();
});
