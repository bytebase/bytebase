import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render } from "@testing-library/react";
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
  setPosition: vi.fn(),
  revealLineInCenter: vi.fn(),
  focus: vi.fn(),
  scrollIntoView: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/monaco", () => ({
  MonacoEditor: ({ onReady }: { onReady: (monaco: unknown, editor: unknown) => void }) => (
    <button
      onClick={() =>
        onReady({}, {
          getModel: () => ({ getLineCount: () => 3 }),
          setPosition: mocks.setPosition,
          revealLineInCenter: mocks.revealLineInCenter,
          focus: mocks.focus,
          getDomNode: () => ({ scrollIntoView: mocks.scrollIntoView }),
        })
      }
    >
      Editor ready
    </button>
  ),
  ReadonlyMonaco: () => <div data-testid="statement" />,
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

test("reveals an Activity anchor after the edit editor is ready", () => {
  const sheetName = `projects/p/sheets/${"c".repeat(64)}`;
  const spec = create(Plan_SpecSchema, {
    id: "spec",
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, { sheet: sheetName }),
    },
  });
  const content = new TextEncoder().encode("SELECT 1;\nSELECT 2;\nSELECT 3;");
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

  const { getByText, rerender } = render(
    <PlanDetailStatementSection spec={spec} />
  );
  fireEvent.click(getByText("common.edit"));
  mocks.threadFocus = {
    commentName: "comments/1",
    specId: "spec",
    lineNumber: 3,
    nonce: 7,
  };
  rerender(<PlanDetailStatementSection spec={spec} />);
  expect(mocks.revealLineInCenter).not.toHaveBeenCalled();

  fireEvent.click(getByText("Editor ready"));
  expect(mocks.setPosition).toHaveBeenCalledWith({ lineNumber: 3, column: 1 });
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
  rerender(<PlanDetailStatementSection spec={spec} />);
  expect(mocks.revealLineInCenter).toHaveBeenLastCalledWith(3);
  expect(mocks.clearThreadFocus).toHaveBeenCalledWith(8);
});
