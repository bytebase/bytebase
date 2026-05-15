import type { ReactNode } from "react";
import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto-es/v1/release_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailProvider } from "../shell/PlanDetailContext";
import { PlanDetailChangesBranch } from "./PlanDetailChangesBranch";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDatabases: vi.fn(),
  fetchDBGroupListByProjectName: vi.fn(),
  getDatabaseByName: vi.fn(),
  getDBGroupByName: vi.fn(),
  getProjectByName: vi.fn(),
  patchState: vi.fn(),
  routerPush: vi.fn(),
  databaseStore: {
    batchGetOrFetchDatabases: vi.fn(async () => []),
    fetchDatabases: vi.fn(),
    getDatabaseByName: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(async () => ({})),
  },
  dbGroupStore: {
    fetchDBGroupListByProjectName: vi.fn(),
    getDBGroupByName: vi.fn(),
    getOrFetchDBGroupByName: vi.fn(async () => ({
      name: "projects/foo/databaseGroups/group-a",
      matchedDatabases: [],
    })),
  },
  environmentStore: {
    getEnvironmentByName: vi.fn(),
  },
  projectStore: {
    getProjectByName: vi.fn(),
  },
  sheetStore: {
    createSheet: vi.fn(async (_projectName, sheet) => sheet),
    getOrFetchSheetByName: vi.fn(async () => undefined),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@bufbuild/protobuf", () => ({
  clone: (_schema: unknown, message: unknown) =>
    JSON.parse(JSON.stringify(message)),
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/react/hooks/useSessionPageSize", () => ({
  useSessionPageSize: () => [20, () => {}],
}));

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: () => null,
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => null,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
  }: {
    children: ReactNode;
    disabled?: boolean;
    onClick?: () => void;
  }) => (
    <button disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuItem: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuTrigger: ({ children }: { children: ReactNode }) => (
    <button>{children}</button>
  ),
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SelectItem: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectTrigger: ({ children }: { children: ReactNode }) => (
    <button>{children}</button>
  ),
  SelectValue: ({ placeholder }: { placeholder?: string }) => (
    <span>{placeholder}</span>
  ),
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
}));

vi.mock("@/react/components/ui/switch", () => ({
  Switch: () => null,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL: "database-group-detail",
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL: "plan-detail-spec-detail",
}));

vi.mock("@/types", () => ({
  isValidDatabaseGroupName: (name: string) =>
    name?.includes("/databaseGroups/"),
  isValidDatabaseName: (name: string) => name?.includes("/databases/"),
  isValidReleaseName: (name: string) => name?.includes("/releases/"),
}));

vi.mock("@/connect", () => ({
  instanceRoleServiceClientConnect: {
    listInstanceRoles: vi.fn(async () => ({ roles: [] })),
  },
  planServiceClientConnect: {
    updatePlan: vi.fn(async (request) => request.plan),
  },
}));

vi.mock("@/store", () => ({
  getProjectNameAndDatabaseGroupName: (name: string) =>
    name.split("/databaseGroups/"),
  pushNotification: vi.fn(),
  useCurrentUserV1: () => ({
    value: { name: "users/me@example.com", email: "me@example.com" },
  }),
  useDatabaseV1Store: () => mocks.databaseStore,
  useDBGroupStore: () => mocks.dbGroupStore,
  useEnvironmentV1Store: () => mocks.environmentStore,
  useProjectV1Store: () => mocks.projectStore,
  useSheetV1Store: () => mocks.sheetStore,
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => ({
    databaseName: name.split("/databases/")[1] ?? name,
    instance: name.split("/databases/")[0],
  }),
  getDatabaseEnvironment: () => undefined,
  getDefaultTransactionMode: () => false,
  getInstanceResource: () => undefined,
  hasProjectPermissionV2: () => true,
}));

vi.mock("@/utils/sheet", () => ({
  getStatementSize: () => 0,
}));

vi.mock("@/utils/v1/databaseGroup", () => ({
  extractDatabaseGroupName: (name: string) =>
    name.split("/databaseGroups/")[1] ?? name,
}));

vi.mock("../utils/localSheet", () => ({
  getLocalSheetByName: (name: string) => ({ name, content: "" }),
  getNextLocalSheetUID: () => "-1",
  removeLocalSheet: vi.fn(),
}));

vi.mock("../utils/options", () => ({
  allowGhostForDatabase: () => false,
  getPlanOptionVisibility: () => ({
    showGhost: false,
    showPreBackup: false,
    showRole: false,
    showTransactionMode: false,
  }),
}));

vi.mock("../utils/planCheck", () => ({
  planCheckRunListForSpec: () => [],
  transformReleaseCheckResultsToPlanCheckRuns: (
    results: CheckReleaseResponse_CheckResult[]
  ) =>
    results.map(
      (result) =>
        ({
          name: result.target,
          results: [],
        }) as unknown as PlanCheckRun
    ),
}));

vi.mock("../utils/specMutation", () => ({
  updateSpecSheetWithStatement: vi.fn(),
}));

vi.mock("./PlanDetailChecks", () => ({
  PlanDetailChecks: () => null,
}));

vi.mock("./PlanDetailDraftChecks", () => ({
  PlanDetailDraftChecks: ({
    onCheckResultsChange,
    selectedSpec,
  }: {
    checkResults?: CheckReleaseResponse_CheckResult[];
    onCheckResultsChange: (
      results: CheckReleaseResponse_CheckResult[] | undefined
    ) => void;
    selectedSpec: { id: string };
  }) => {
    return (
      <button
        onClick={() =>
          onCheckResultsChange([
            {
              advices: [],
              target: `check-run-for-${selectedSpec.id}`,
            } as unknown as CheckReleaseResponse_CheckResult,
          ])
        }
      >
        run draft checks
      </button>
    );
  },
}));

vi.mock("./PlanDetailStatementSection", () => ({
  PlanDetailStatementSection: ({
    planCheckRuns,
    spec,
  }: {
    planCheckRuns?: PlanCheckRun[];
    spec: { id: string };
  }) => (
    <div data-testid="statement-section">
      {spec.id}:{planCheckRuns?.map((run) => run.name).join(",") ?? ""}
    </div>
  ),
}));

vi.mock("./PlanDetailTabStrip", () => ({
  PlanDetailTabItem: ({
    children,
    onSelect,
  }: {
    children: ReactNode;
    onSelect?: () => void;
  }) => <button onClick={onSelect}>{children}</button>,
  PlanDetailTabStrip: ({
    action,
    children,
  }: {
    action?: ReactNode;
    children: ReactNode;
  }) => (
    <div>
      {action}
      {children}
    </div>
  ),
}));

const DB_WIDGETS = "instances/test/databases/widgets";
const DB_COGS = "instances/test/databases/cogs";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.useFakeTimers();
  vi.clearAllMocks();
  mocks.databaseStore.fetchDatabases.mockResolvedValue({
    databases: [{ name: DB_WIDGETS }, { name: DB_COGS }],
    nextPageToken: "",
  });
  mocks.dbGroupStore.fetchDBGroupListByProjectName.mockResolvedValue([
    {
      name: "projects/foo/databaseGroups/group-a",
      title: "group-a",
    },
  ]);
  mocks.databaseStore.getDatabaseByName.mockImplementation((name: string) => ({
    name,
    effectiveEnvironment: "environments/prod",
  }));
  mocks.dbGroupStore.getDBGroupByName.mockReturnValue({
    name: "",
    matchedDatabases: [],
  });
  mocks.projectStore.getProjectByName.mockReturnValue({
    name: "projects/foo",
    title: "Foo",
  });

  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  vi.useRealTimers();
  document.body.removeChild(container);
});

function buildPageState(): PlanDetailPageState {
  return {
    activePhases: new Set(["changes"]),
    bypassLeaveGuardOnce: vi.fn(),
    closeTaskPanel: vi.fn(),
    containerWidth: 1200,
    currentUser: {
      name: "users/me@example.com",
    } as PlanDetailPageState["currentUser"],
    desktopSidebarWidth: 320,
    expandPhase: vi.fn(),
    isCreating: true,
    isEditing: false,
    isInitializing: false,
    isRefreshing: false,
    isRunningChecks: false,
    lastRefreshTime: 0,
    mobileSidebarOpen: false,
    pageKey: "foo/create/spec-1",
    patchState: mocks.patchState,
    pendingLeaveConfirm: false,
    plan: {
      name: "projects/foo/plans/create",
      creator: "users/me@example.com",
      hasRollout: false,
      state: State.ACTIVE,
      specs: [
        {
          id: "spec-1",
          config: {
            case: "changeDatabaseConfig",
            value: {
              targets: [],
            },
          },
        },
      ],
    } as unknown as PlanDetailPageState["plan"],
    planCheckRuns: [],
    planId: "create",
    project: {
      name: "projects/foo",
      title: "Foo",
    } as PlanDetailPageState["project"],
    projectCanCreateRollout: true,
    projectId: "foo",
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    projectTitle: "Foo",
    readonly: false,
    ready: true,
    refreshState: vi.fn(async () => {}),
    resolveLeaveConfirm: vi.fn(),
    routeName: undefined,
    routePhase: "changes",
    selectedTaskName: undefined,
    setEditing: vi.fn(),
    setIsRunningChecks: vi.fn(),
    setMobileSidebarOpen: vi.fn(),
    sidebarMode: "DESKTOP",
    taskRuns: [],
    togglePhase: vi.fn(),
  };
}

function TestHarness({ rerenderToken }: { rerenderToken: number }) {
  return (
    <PlanDetailProvider
      value={{ ...buildPageState(), pageKey: String(rerenderToken) }}
    >
      <PlanDetailChangesBranch
        selectedSpecId="spec-1"
        onSelectedSpecIdChange={vi.fn()}
      />
    </PlanDetailProvider>
  );
}

function StatefulSpecHarness() {
  const [selectedSpecId, setSelectedSpecId] = useState("spec-1");
  const page = buildPageState();
  page.plan.specs = [
    {
      id: "spec-1",
      config: {
        case: "changeDatabaseConfig",
        value: {
          targets: [],
        },
      },
    },
    {
      id: "spec-2",
      config: {
        case: "changeDatabaseConfig",
        value: {
          targets: [],
        },
      },
    },
  ] as unknown as PlanDetailPageState["plan"]["specs"];

  return (
    <PlanDetailProvider value={page}>
      <PlanDetailChangesBranch
        selectedSpecId={selectedSpecId}
        onSelectedSpecIdChange={setSelectedSpecId}
      />
    </PlanDetailProvider>
  );
}

function renderHarness(rerenderToken: number) {
  act(() => {
    root.render(<TestHarness rerenderToken={rerenderToken} />);
  });
}

async function flush() {
  await act(async () => {
    vi.runOnlyPendingTimers();
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe("PlanDetailChangesBranch", () => {
  it("keeps Add Change target selection when the page rerenders", async () => {
    renderHarness(0);

    const addChangeButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "plan.add-spec"
    );
    expect(addChangeButton).toBeTruthy();

    await act(async () => {
      addChangeButton?.click();
    });
    await flush();

    const groupTab = [...container.querySelectorAll("button")].find((button) =>
      button.textContent?.includes("common.database-group")
    );
    expect(groupTab).toBeTruthy();

    await act(async () => {
      groupTab?.click();
    });
    await flush();
    expect(container.textContent).toContain("group-a");

    renderHarness(1);
    await flush();

    expect(container.textContent).toContain("group-a");
  });

  it("does not navigate while adding a change to a creating plan", async () => {
    const onSelectedSpecIdChange = vi.fn();

    act(() => {
      root.render(
        <PlanDetailProvider value={buildPageState()}>
          <PlanDetailChangesBranch
            selectedSpecId="spec-1"
            onSelectedSpecIdChange={onSelectedSpecIdChange}
          />
        </PlanDetailProvider>
      );
    });

    const addChangeButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "plan.add-spec"
    );
    expect(addChangeButton).toBeTruthy();

    await act(async () => {
      addChangeButton?.click();
    });
    await flush();

    const groupTab = [...container.querySelectorAll("button")].find((button) =>
      button.textContent?.includes("common.database-group")
    );
    expect(groupTab).toBeTruthy();

    await act(async () => {
      groupTab?.click();
    });
    await flush();

    const groupRow = [...container.querySelectorAll("tr")].find((row) =>
      row.textContent?.includes("group-a")
    );
    expect(groupRow).toBeTruthy();

    await act(async () => {
      (groupRow as HTMLElement | undefined)?.click();
    });
    await flush();

    const confirmButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.confirm"
    );
    expect(confirmButton).toBeTruthy();
    expect(confirmButton?.disabled).toBe(false);

    await act(async () => {
      confirmButton?.click();
    });
    await flush();

    expect(onSelectedSpecIdChange).toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
  });

  it("does not copy draft check runs between creating plan changes", async () => {
    act(() => {
      root.render(<StatefulSpecHarness />);
    });

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "run draft checks"
        ) as HTMLButtonElement
      ).click();
    });
    await flush();

    expect(container.textContent).toContain("spec-1:check-run-for-spec-1");

    const specTabs = [...container.querySelectorAll("button")].filter(
      (button) => button.textContent?.includes("plan.spec.type.database-change")
    );
    expect(specTabs.length).toBe(2);

    await act(async () => {
      specTabs[1]?.click();
    });
    await flush();

    expect(container.textContent).toContain("spec-2:");
    expect(container.textContent).not.toContain("spec-2:check-run-for-spec-1");
  });
});
