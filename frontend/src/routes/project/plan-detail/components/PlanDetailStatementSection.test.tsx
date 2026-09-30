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
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/monaco", () => ({
  MonacoEditor: ({ onChange }: { onChange: (value: string) => void }) => (
    <>
      <button onClick={() => onChange("SELECT 2;\n".repeat(220_000))}>
        Make large draft
      </button>
      <button onClick={() => onChange("SELECT 2;")}>Make small draft</button>
    </>
  ),
  ReadonlyMonaco: ({ content }: { content: string }) => (
    <div data-testid="statement">{content.slice(0, 24)}</div>
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

test("shows a draft-specific warning before saving and removes it below the limit", () => {
  const sheetName = `projects/p/sheets/${"c".repeat(64)}`;
  const spec = create(Plan_SpecSchema, {
    id: "spec",
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, { sheet: sheetName }),
    },
  });
  const content = new TextEncoder().encode("SELECT 1;");
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

  const { getByText, queryByText } = render(
    <PlanDetailStatementSection spec={spec} />
  );
  fireEvent.click(getByText("common.edit"));
  fireEvent.click(getByText("Make large draft"));
  expect(getByText("issue.statement-exceeds-preview-limit")).toBeTruthy();
  expect(queryByText("common.download")).toBeNull();

  fireEvent.click(getByText("Make small draft"));
  expect(queryByText("issue.statement-exceeds-preview-limit")).toBeNull();
});
