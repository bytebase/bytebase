import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ---------------------------------------------------------------------------
// UI primitive mocks
// ---------------------------------------------------------------------------

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => null,
}));

vi.mock("@/react/components/IssueLabelSelect", () => ({
  IssueLabelSelect: () => null,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? createElement("div", { "data-testid": "sheet" }, children) : null,
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("h2", {}, children),
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
    variant: _v,
    size: _s,
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: string;
    size?: string;
  }) => createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: ({ className: _c, ...props }: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: ({
    className: _c,
    wrapperClassName: _w,
    ...props
  }: InputHTMLAttributes<HTMLInputElement> & {
    wrapperClassName?: string;
  }) => createElement("input", props),
}));

vi.mock("@/react/components/ui/switch", () => ({
  Switch: ({
    checked,
    onCheckedChange,
  }: {
    checked: boolean;
    onCheckedChange: (v: boolean) => void;
  }) =>
    createElement("input", {
      type: "checkbox",
      checked,
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        onCheckedChange(e.target.checked),
    }),
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...cls: (string | false | null | undefined)[]) =>
    cls.filter(Boolean).join(" "),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/react/hooks/useSessionPageSize", () => ({
  useSessionPageSize: () => [20, () => {}],
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

// ---------------------------------------------------------------------------
// Infra / external mocks
// ---------------------------------------------------------------------------

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@/types/proto-es/v1/common_pb", () => ({
  ExportFormat: { CSV: 1, JSON: 2, SQL: 3, XLSX: 4 },
}));

vi.mock("@/types/proto-es/v1/database_group_service_pb", () => ({
  DatabaseGroupView: { BASIC: 1, FULL: 2 },
}));

vi.mock("@/types/proto-es/v1/issue_service_pb", () => ({
  Issue_Type: { DATABASE_EXPORT: 1 },
  IssueSchema: {},
}));

vi.mock("@/types/proto-es/v1/plan_service_pb", () => ({
  Plan_ExportDataConfigSchema: {},
  Plan_SpecSchema: {},
  PlanSchema: {},
}));

vi.mock("@/types/proto-es/v1/sheet_service_pb", () => ({
  SheetSchema: {},
}));

vi.mock("@/router", () => ({ router: { push: vi.fn() } }));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_ISSUE_DETAIL: "issue-detail",
}));

vi.mock("@/react/components/instance/constants", () => ({
  EngineIconPath: {},
}));

vi.mock("@/types", () => ({
  isValidDatabaseName: (name: string) =>
    typeof name === "string" && name.includes("/databases/"),
  isValidDatabaseGroupName: (name: string) =>
    typeof name === "string" && name.includes("/databaseGroups/"),
}));

vi.mock("@/utils", () => ({
  DEFAULT_MAX_RESULT_SIZE_IN_MB: 100,
  extractDatabaseGroupName: (name: string) =>
    name.split("/databaseGroups/")[1] ?? name,
  extractDatabaseResourceName: (name: string) => {
    const parts = name.split("/databases/");
    return { databaseName: parts[1] ?? name };
  },
  extractIssueUID: (name: string) => name.split("/").pop() ?? "",
  extractProjectResourceName: (name: string) => name.split("/")[1] ?? name,
  generatePlanTitle: (_template: string, names?: string[]) => {
    if (!names || names.length === 0) return "[All databases] export";
    if (names.length === 1) return `[${names[0]}] export`;
    return `[${names.length} databases] export`;
  },
  getDatabaseEnvironment: () => undefined,
  getInstanceResource: () => undefined,
  normalizeTitle: (s: string) => s.trim(),
  setSheetStatement: () => {},
}));

// ---------------------------------------------------------------------------
// Store mocks — stable singletons to avoid useEffect dependency churn
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  getProjectByName: vi.fn(),
  getOrFetchDatabaseByName: vi.fn(),
  getDatabaseByName: vi.fn(),
  fetchDatabases: vi.fn(),
  getOrFetchDBGroupByName: vi.fn(),
  fetchDBGroupListByProjectName: vi.fn(),
  createSheet: vi.fn(),
  experimentalCreateIssueByPlan: vi.fn(),
  pushNotification: vi.fn(),
  currentUser: { name: "users/me@example.com", email: "me@example.com" },
}));

const stableProjectStore = {
  get getProjectByName() {
    return mocks.getProjectByName;
  },
};

const stableDatabaseStore = {
  get fetchDatabases() {
    return mocks.fetchDatabases;
  },
  get getOrFetchDatabaseByName() {
    return mocks.getOrFetchDatabaseByName;
  },
  get getDatabaseByName() {
    return mocks.getDatabaseByName;
  },
};

const stableDBGroupStore = {
  get getOrFetchDBGroupByName() {
    return mocks.getOrFetchDBGroupByName;
  },
  get fetchDBGroupListByProjectName() {
    return mocks.fetchDBGroupListByProjectName;
  },
};

const stableSheetStore = {
  get createSheet() {
    return mocks.createSheet;
  },
};

const stableSettingStore = {
  workspaceProfile: { sqlResultSize: BigInt(100 * 1024 * 1024) },
};

vi.mock("@/store", () => ({
  DEFAULT_MAX_RESULT_SIZE_IN_MB: 100,
  useProjectV1Store: () => stableProjectStore,
  useDatabaseV1Store: () => stableDatabaseStore,
  useDBGroupStore: () => stableDBGroupStore,
  useSheetV1Store: () => stableSheetStore,
  useSettingV1Store: () => stableSettingStore,
  useCurrentUserV1: () => ({ value: mocks.currentUser }),
  experimentalCreateIssueByPlan: mocks.experimentalCreateIssueByPlan,
  pushNotification: mocks.pushNotification,
}));

import { nativeChange } from "@/react/test-utils/nativeChange";
import { DataExportPrepSheet } from "./DataExportPrepSheet";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const DB_WIDGETS = "instances/test/databases/widgets";
const DB_COGS = "instances/test/databases/cogs";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.fetchDatabases.mockResolvedValue({
    databases: [
      { name: DB_WIDGETS, engine: 1 },
      { name: DB_COGS, engine: 1 },
    ],
    nextPageToken: "",
  });
  mocks.fetchDBGroupListByProjectName.mockResolvedValue([]);
  mocks.getOrFetchDatabaseByName.mockResolvedValue(undefined);
  mocks.getDatabaseByName.mockReturnValue(undefined);

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

function setupProjectMock(enforceIssueTitle: boolean): void {
  mocks.getProjectByName.mockReturnValue({
    name: "projects/foo",
    enforceIssueTitle,
    forceIssueLabels: false,
    issueLabels: [],
  });
}

/** Render with the widgets database pre-selected at step 1, then advance to
 *  step 2. Going via step 1 (rather than seeding `step: 2`) keeps the "Back"
 *  button able to return to step 1 — when `seed.step` is set, handleCancel
 *  closes the sheet instead of navigating back. */
async function renderSheet(enforceIssueTitle: boolean): Promise<void> {
  setupProjectMock(enforceIssueTitle);
  await act(async () => {
    root.render(
      createElement(DataExportPrepSheet, {
        open: true,
        onClose: () => {},
        projectName: "projects/foo",
        seed: { selectedDatabaseNames: [DB_WIDGETS] },
      })
    );
    await Promise.resolve();
    await Promise.resolve();
  });
  // Wait for async database list load.
  await flush();
  // Advance to step 2 with widgets selected.
  await act(async () => {
    getNextButton().click();
  });
  await flush();
}

function getTitleInput(): HTMLInputElement {
  return container.querySelector(
    "input[placeholder='common.title']"
  ) as HTMLInputElement;
}

function getStatementTextarea(): HTMLTextAreaElement {
  return container.querySelector(
    "textarea[placeholder='SELECT ...']"
  ) as HTMLTextAreaElement;
}

function getCreateButton(): HTMLButtonElement {
  return [...container.querySelectorAll("button")].find((b) =>
    b.textContent?.includes("common.create")
  ) as HTMLButtonElement;
}

function getBackButton(): HTMLButtonElement {
  return [...container.querySelectorAll("button")].find((b) =>
    b.textContent?.includes("common.back")
  ) as HTMLButtonElement;
}

function getNextButton(): HTMLButtonElement {
  return [...container.querySelectorAll("button")].find((b) =>
    b.textContent?.includes("common.next")
  ) as HTMLButtonElement;
}

async function flush(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

/** Toggle a database checkbox in step 1 by matching its row text. */
async function toggleDatabaseRow(databaseName: string): Promise<void> {
  const rows = [...container.querySelectorAll("tr")];
  const row = rows.find((tr) => tr.textContent?.includes(databaseName));
  if (!row) throw new Error(`Row for ${databaseName} not found`);
  await act(async () => {
    row.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
}

describe("DataExportPrepSheet — enforceIssueTitle (BYT-9310)", () => {
  it("auto-fills title from targets when enforceIssueTitle is false", async () => {
    await renderSheet(false);

    const titleInput = getTitleInput();
    expect(titleInput).toBeTruthy();
    expect(titleInput.value).toBe(`[widgets] export`);
  });

  it("does not auto-fill title when enforceIssueTitle is true", async () => {
    await renderSheet(true);

    const titleInput = getTitleInput();
    expect(titleInput).toBeTruthy();
    expect(titleInput.value).toBe("");
  });

  it("preserves a user-typed title across target changes", async () => {
    await renderSheet(false);

    expect(getTitleInput().value).toBe(`[widgets] export`);

    // User overrides the auto-filled title.
    await act(async () => {
      nativeChange(getTitleInput(), "my custom title");
    });
    expect(getTitleInput().value).toBe("my custom title");

    // Navigate back to step 1 (state is preserved across step nav).
    await act(async () => {
      getBackButton().click();
    });
    await flush();

    // Add another database to the selection.
    await toggleDatabaseRow("cogs");
    await flush();

    // Advance back to step 2 with the expanded target list.
    await act(async () => {
      getNextButton().click();
    });
    await flush();

    // Custom title must be preserved — auto-fill must not overwrite it.
    expect(getTitleInput().value).toBe("my custom title");
  });

  it("disables Create when enforceIssueTitle is true and title is empty, enables when typed", async () => {
    await renderSheet(true);

    // Fill statement so the only gating condition left is the title.
    await act(async () => {
      nativeChange(getStatementTextarea(), "SELECT 1");
    });
    await flush();

    expect(getTitleInput().value).toBe("");
    expect(getCreateButton().disabled).toBe(true);

    await act(async () => {
      nativeChange(getTitleInput(), "my title");
    });
    await flush();

    expect(getCreateButton().disabled).toBe(false);
  });
});
