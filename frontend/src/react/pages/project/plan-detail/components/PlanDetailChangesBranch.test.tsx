import type { ReactNode } from "react";
import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto-es/v1/release_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailProvider } from "../shell/PlanDetailContext";
import {
  DatabaseGroupTarget,
  PlanDetailChangesBranch,
} from "./PlanDetailChangesBranch";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDatabases: vi.fn(),
  fetchDBGroupListByProjectName: vi.fn(),
  getDatabaseByName: vi.fn(),
  getDBGroupByName: vi.fn(),
  getInstanceResource: vi.fn(),
  getPlanOptionVisibility: vi.fn(),
  getProjectByName: vi.fn(),
  selectedSelectValueChange: undefined as
    | ((value: string | null | undefined) => void)
    | undefined,
  instanceRoleServiceClientConnect: {
    listInstanceRoles: vi.fn(async () => ({
      roles: [] as Array<{ roleName: string }>,
    })),
  },
  localSheets: new Map<
    string,
    { content: Uint8Array; contentSize: bigint; name: string }
  >(),
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
      matchedDatabases: [] as Array<{ name: string }>,
    })),
    createSheet: vi.fn(async (_projectName, sheet) => sheet),
    getOrFetchSheetByName: vi.fn(async () => undefined),
  },
  environmentStore: {
    getEnvironmentByName: vi.fn(),
  },
  projectStore: {
    getProjectByName: vi.fn(),
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
  SearchInput: ({
    wrapperClassName,
    ...props
  }: React.InputHTMLAttributes<HTMLInputElement> & {
    wrapperClassName?: string;
  }) => (
    <div
      className={`relative flex-1 ${wrapperClassName ?? ""}`}
      data-testid="targets-search-wrapper"
      data-wrapper-class-name={wrapperClassName}
    >
      <input {...props} />
    </div>
  ),
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: ({
    children,
    onValueChange,
  }: {
    children: ReactNode;
    onValueChange?: (value: string | null | undefined) => void;
  }) => {
    mocks.selectedSelectValueChange = onValueChange;
    return <div>{children}</div>;
  },
  SelectContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SelectItem: ({
    children,
    value,
  }: {
    children: ReactNode;
    value?: string;
  }) => {
    const onValueChange = mocks.selectedSelectValueChange;
    return (
      <button onClick={() => onValueChange?.(value)} type="button">
        {children}
      </button>
    );
  },
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
  SheetBody: ({
    children,
    className,
  }: {
    children: ReactNode;
    className?: string;
  }) => (
    <div
      className={`flex flex-1 flex-col overflow-y-auto px-6 py-4 ${className ?? ""}`}
      data-testid="targets-sheet-body"
    >
      {children}
    </div>
  ),
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
}));

vi.mock("@/react/components/ui/switch", () => ({
  Switch: ({
    checked,
    disabled,
    onCheckedChange,
  }: {
    checked: boolean;
    disabled?: boolean;
    onCheckedChange: (checked: boolean) => void;
  }) => (
    <button
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      type="button"
    >
      {checked ? "switch-on" : "switch-off"}
    </button>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  PopoverContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  PopoverTrigger: ({ children }: { children: ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    push: mocks.routerPush,
    resolve: () => ({ href: "/database-group-detail" }),
  },
}));

vi.mock("@/types", () => ({
  isValidDatabaseGroupName: (name: string) =>
    name?.includes("/databaseGroups/"),
  isValidDatabaseName: (name: string) => name?.includes("/databases/"),
  isValidReleaseName: (name: string) => name?.includes("/releases/"),
  unknownDatabaseGroup: () => ({ name: "", matchedDatabases: [] }),
}));

vi.mock("@/connect", () => ({
  instanceRoleServiceClientConnect: mocks.instanceRoleServiceClientConnect,
  planServiceClientConnect: {
    updatePlan: vi.fn(async (request) => request.plan),
  },
}));

vi.mock("@/store", () => ({
  getProjectNameAndDatabaseGroupName: (name: string) =>
    name.split("/databaseGroups/"),
  pushNotification: vi.fn(),
}));

// The migrated component reads resource state via imperative
// `useAppStore.getState().<method>()` and reactive selectors over app-store
// maps. Delegate map lookups to the existing store mocks so per-test setup
// keeps working.
vi.mock("@/react/stores/app", async () => {
  const { DatabaseGroupView } = await vi.importActual<
    typeof import("@/types/proto-es/v1/database_group_service_pb")
  >("@/types/proto-es/v1/database_group_service_pb");
  const databasesByName = new Proxy({} as Record<string, unknown>, {
    get: (_t, key) =>
      typeof key === "string"
        ? mocks.databaseStore.getDatabaseByName(key)
        : undefined,
  });
  const dbGroupsByName = new Proxy({} as Record<string, unknown>, {
    get: (_t, key) =>
      typeof key === "string"
        ? mocks.dbGroupStore.getDBGroupByName(key)
        : undefined,
  });
  const dbGroupViewByName = new Proxy({} as Record<string, unknown>, {
    get: (_t, key) =>
      typeof key === "string" ? DatabaseGroupView.FULL : undefined,
  });
  const projectsByName = new Proxy({} as Record<string, unknown>, {
    get: (_t, key) =>
      typeof key === "string"
        ? mocks.projectStore.getProjectByName(key)
        : undefined,
  });
  const useAppStore = Object.assign(
    (
      selector: (s: {
        databasesByName: Record<string, unknown>;
        dbGroupsByName: Record<string, unknown>;
        dbGroupViewByName: Record<string, unknown>;
        projectsByName: Record<string, unknown>;
        environmentList: unknown[];
      }) => unknown
    ) =>
      selector({
        databasesByName,
        dbGroupsByName,
        dbGroupViewByName,
        projectsByName,
        environmentList: [],
      }),
    {
      getState: () => ({
        ...mocks.databaseStore,
        ...mocks.dbGroupStore,
        getProjectByName: mocks.projectStore.getProjectByName,
        getEnvironmentByName: mocks.environmentStore.getEnvironmentByName,
      }),
    }
  );
  return { useAppStore };
});

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    name: "users/me@example.com",
    email: "me@example.com",
  }),
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => ({
    databaseName: name.split("/databases/")[1] ?? name,
    instance: name.split("/databases/")[0],
  }),
  getDatabaseEnvironment: () => undefined,
  getDefaultTransactionMode: () => false,
  getInstanceResource: mocks.getInstanceResource,
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
  getLocalSheetByName: (name: string) => {
    const existing = mocks.localSheets.get(name);
    if (existing) return existing;
    const sheet = { name, content: new Uint8Array(), contentSize: 0n };
    mocks.localSheets.set(name, sheet);
    return sheet;
  },
  getNextLocalSheetUID: () => "-1",
  getSpecStatementContent: (spec: {
    config?: { case?: string; value?: { sheet?: string } };
  }) => {
    if (spec.config?.case !== "changeDatabaseConfig") return undefined;
    const sheetName = spec.config.value?.sheet;
    const sheet = sheetName ? mocks.localSheets.get(sheetName) : undefined;
    return sheet?.content;
  },
  isSameStatementContent: (a?: Uint8Array, b?: Uint8Array) => {
    if (a === b) return true;
    if (!a || !b || a.length !== b.length) return false;
    return a.every((byte, index) => byte === b[index]);
  },
  removeLocalSheet: vi.fn(),
}));

vi.mock("../utils/options", () => ({
  allowGhostForDatabase: () => false,
  getPlanOptionVisibility: mocks.getPlanOptionVisibility,
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

vi.mock("./PlanDetailAggregateChecks", () => ({
  PlanDetailAggregateChecks: () => null,
}));

vi.mock("./PlanDetailDraftChecks", () => ({
  PlanDetailDraftChecks: ({
    onCheckResultsChange,
    selectedSpec,
  }: {
    checkResults?: CheckReleaseResponse_CheckResult[];
    onCheckResultsChange: (
      content: Uint8Array | undefined,
      results: CheckReleaseResponse_CheckResult[] | undefined
    ) => void;
    selectedSpec: {
      config?: { case?: string; value?: { sheet?: string } };
      id: string;
    };
  }) => {
    return (
      <button
        onClick={() => {
          const sheetName =
            selectedSpec.config?.case === "changeDatabaseConfig"
              ? selectedSpec.config.value?.sheet
              : undefined;
          const sheet = sheetName
            ? mocks.localSheets.get(sheetName)
            : undefined;
          const results = [
            {
              advices: [],
              target: `check-run-for-${selectedSpec.id}`,
            } as unknown as CheckReleaseResponse_CheckResult,
          ];
          onCheckResultsChange(sheet?.content, results);
        }}
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
    statementVersion,
  }: {
    planCheckRuns?: PlanCheckRun[];
    statementVersion?: number;
    spec: { id: string };
  }) => (
    <div data-testid="statement-section">
      {spec.id}:{planCheckRuns?.map((run) => run.name).join(",") ?? ""}:
      {statementVersion ?? 0}
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
  mocks.localSheets.clear();
  mocks.getPlanOptionVisibility.mockReturnValue({
    shouldShow: false,
    showGhost: false,
    showInstanceRole: false,
    showIsolationLevel: false,
    showPreBackup: false,
    showTransactionMode: false,
  });
  mocks.getInstanceResource.mockReturnValue({
    engine: Engine.MYSQL,
    engineVersion: "",
    name: "instances/test",
    title: "test",
  });
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
    expandPhase: vi.fn(),
    isCreating: true,
    isEditing: false,
    isInitializing: false,
    isRefreshing: false,
    isRunningChecks: false,
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
    layoutMode: "DESKTOP",
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
  it("renders all targets in a single scrollable sheet list", async () => {
    const page = buildPageState();
    const targets = Array.from(
      { length: 21 },
      (_, index) => `instances/test/databases/db_${index}`
    );
    page.plan.specs = [
      {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: {
            targets,
          },
        },
      },
    ] as unknown as PlanDetailPageState["plan"]["specs"];

    act(() => {
      root.render(
        <PlanDetailProvider value={page}>
          <PlanDetailChangesBranch
            selectedSpecId="spec-1"
            onSelectedSpecIdChange={vi.fn()}
          />
        </PlanDetailProvider>
      );
    });
    await flush();

    const viewAllButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "plan.targets.view-all"
    );
    expect(viewAllButton).toBeTruthy();

    await act(async () => {
      viewAllButton?.click();
    });
    await flush();

    const sheetBody = container.querySelector(
      "[data-testid='targets-sheet-body']"
    );
    const elements = Array.from(sheetBody?.querySelectorAll("*") ?? []);
    const scrollArea = elements.find(
      (element) =>
        element.className.includes("min-h-0") &&
        element.className.includes("flex-1") &&
        element.className.includes("overflow-y-auto")
    );
    const targetList = scrollArea
      ? Array.from(scrollArea.querySelectorAll("*")).find(
          (element) =>
            element.className.includes("flex") &&
            element.className.includes("flex-col") &&
            element.className.includes("gap-2")
        )
      : undefined;
    const searchWrapper = container.querySelector(
      "[data-testid='targets-search-wrapper']"
    );
    const targetRows =
      targetList?.querySelectorAll(".w-full.rounded-lg.border") ?? [];

    expect(sheetBody?.className).toContain("overflow-hidden");
    expect(searchWrapper?.getAttribute("data-wrapper-class-name")).toContain(
      "flex-none"
    );
    expect(scrollArea).toBeTruthy();
    expect(targetList).toBeTruthy();
    expect(targetRows.length).toBe(targets.length);
  });

  it("renders database group overflow as a popover action", async () => {
    mocks.dbGroupStore.getDBGroupByName.mockReturnValue({
      name: "projects/foo/databaseGroups/group-a",
      matchedDatabases: Array.from({ length: 8 }, (_, index) => ({
        name: `instances/test/databases/db_${index}`,
      })),
    });
    mocks.dbGroupStore.getOrFetchDBGroupByName.mockResolvedValue({
      name: "projects/foo/databaseGroups/group-a",
      matchedDatabases: Array.from({ length: 8 }, (_, index) => ({
        name: `instances/test/databases/db_${index}`,
      })),
    });

    act(() => {
      root.render(
        <DatabaseGroupTarget target="projects/foo/databaseGroups/group-a" />
      );
    });
    await flush();

    const moreButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.n-more"
    );

    expect(moreButton).toBeTruthy();
    expect(container.textContent).toContain("common.databases");
  });

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

  it("keeps hook order stable when the selected spec appears after an empty render", async () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => undefined);
    const emptyPage = buildPageState();
    emptyPage.plan.specs =
      [] as unknown as PlanDetailPageState["plan"]["specs"];
    const page = buildPageState();

    try {
      act(() => {
        root.render(
          <PlanDetailProvider value={emptyPage}>
            <PlanDetailChangesBranch
              selectedSpecId="spec-1"
              onSelectedSpecIdChange={vi.fn()}
            />
          </PlanDetailProvider>
        );
      });
      await flush();

      expect(container.textContent).toContain("common.no-data");

      act(() => {
        root.render(
          <PlanDetailProvider value={page}>
            <PlanDetailChangesBranch
              selectedSpecId="spec-1"
              onSelectedSpecIdChange={vi.fn()}
            />
          </PlanDetailProvider>
        );
      });
      await flush();
    } finally {
      consoleError.mockRestore();
    }

    expect(
      consoleError.mock.calls.some((call) =>
        String(call[0]).includes(
          "React has detected a change in the order of Hooks"
        )
      )
    ).toBe(false);
  });

  it("hides stale draft check runs after the create-plan statement changes", async () => {
    const sheetName = "projects/foo/sheets/-1";
    mocks.localSheets.set(sheetName, {
      name: sheetName,
      content: new TextEncoder().encode("select 1;"),
      contentSize: 0n,
    });
    const page = buildPageState();
    page.plan.specs = [
      {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: {
            sheet: sheetName,
            targets: [],
          },
        },
      },
    ] as unknown as PlanDetailPageState["plan"]["specs"];

    const render = () => {
      root.render(
        <PlanDetailProvider value={page}>
          <PlanDetailChangesBranch
            selectedSpecId="spec-1"
            onSelectedSpecIdChange={vi.fn()}
          />
        </PlanDetailProvider>
      );
    };

    act(render);
    await flush();

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "run draft checks"
        ) as HTMLButtonElement
      ).click();
    });
    await flush();

    expect(container.textContent).toContain("spec-1:check-run-for-spec-1");

    mocks.localSheets.set(sheetName, {
      name: sheetName,
      content: new TextEncoder().encode("create table t(id int);"),
      contentSize: 0n,
    });

    act(render);
    await flush();

    expect(container.textContent).toContain("spec-1::0");
    expect(container.textContent).not.toContain("check-run-for-spec-1");
  });

  it("keeps draft check runs when the statement is reverted to the checked text", async () => {
    const sheetName = "projects/foo/sheets/-1";
    mocks.localSheets.set(sheetName, {
      name: sheetName,
      content: new TextEncoder().encode("select 1;"),
      contentSize: 0n,
    });
    const page = buildPageState();
    page.plan.specs = [
      {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: {
            sheet: sheetName,
            targets: [],
          },
        },
      },
    ] as unknown as PlanDetailPageState["plan"]["specs"];

    const render = () => {
      root.render(
        <PlanDetailProvider value={page}>
          <PlanDetailChangesBranch
            selectedSpecId="spec-1"
            onSelectedSpecIdChange={vi.fn()}
          />
        </PlanDetailProvider>
      );
    };

    act(render);
    await flush();

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "run draft checks"
        ) as HTMLButtonElement
      ).click();
    });
    await flush();

    expect(container.textContent).toContain("spec-1:check-run-for-spec-1");

    // Reverting to the same text mints a fresh Uint8Array (new reference) with
    // identical bytes; the prior checks are still valid, so they must stay.
    mocks.localSheets.set(sheetName, {
      name: sheetName,
      content: new TextEncoder().encode("select 1;"),
      contentSize: 0n,
    });

    act(render);
    await flush();

    expect(container.textContent).toContain("spec-1:check-run-for-spec-1");
  });

  it("refreshes the statement section after changing create-plan options", async () => {
    mocks.getPlanOptionVisibility.mockReturnValue({
      shouldShow: true,
      showGhost: false,
      showInstanceRole: false,
      showIsolationLevel: false,
      showPreBackup: false,
      showTransactionMode: true,
    });
    const page = buildPageState();
    page.plan.specs = [
      {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: {
            sheet: "projects/foo/sheets/-1",
            targets: [DB_WIDGETS],
          },
        },
      },
    ] as unknown as PlanDetailPageState["plan"]["specs"];

    act(() => {
      root.render(
        <PlanDetailProvider value={page}>
          <PlanDetailChangesBranch
            selectedSpecId="spec-1"
            onSelectedSpecIdChange={vi.fn()}
          />
        </PlanDetailProvider>
      );
    });
    await flush();

    expect(container.textContent).toContain("spec-1::0");

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "switch-off"
        ) as HTMLButtonElement | undefined
      )?.click();
    });
    await flush();

    expect(container.textContent).toContain("spec-1::1");
  });

  it("clears the selected role with the default role option", async () => {
    mocks.getPlanOptionVisibility.mockReturnValue({
      shouldShow: true,
      showGhost: false,
      showInstanceRole: true,
      showIsolationLevel: false,
      showPreBackup: false,
      showTransactionMode: false,
    });
    mocks.instanceRoleServiceClientConnect.listInstanceRoles.mockResolvedValue({
      roles: [{ roleName: "bbsample" }],
    });
    const sheetName = "projects/foo/sheets/-1";
    const statement =
      "/* === Bytebase Role Setter. DO NOT EDIT. === */\nSET ROLE bbsample;\nSELECT 1;";
    mocks.localSheets.set(sheetName, {
      name: sheetName,
      content: new TextEncoder().encode(statement),
      contentSize: 0n,
    });
    const page = buildPageState();
    page.plan.specs = [
      {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: {
            sheet: sheetName,
            targets: [DB_WIDGETS],
          },
        },
      },
    ] as unknown as PlanDetailPageState["plan"]["specs"];

    act(() => {
      root.render(
        <PlanDetailProvider value={page}>
          <PlanDetailChangesBranch
            selectedSpecId="spec-1"
            onSelectedSpecIdChange={vi.fn()}
          />
        </PlanDetailProvider>
      );
    });
    await flush();

    expect(container.textContent).toContain("instance.default-role");
    expect(container.textContent).toContain("bbsample");

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "instance.default-role"
        ) as HTMLButtonElement | undefined
      )?.click();
    });
    await flush();

    expect(container.textContent).toContain("spec-1::1");
    expect(
      new TextDecoder().decode(mocks.localSheets.get(sheetName)?.content)
    ).toBe("SELECT 1;");
  });
});
