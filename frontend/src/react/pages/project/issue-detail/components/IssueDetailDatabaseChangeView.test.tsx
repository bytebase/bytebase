import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const listInstanceRoles = vi.fn();
  const getDatabaseByName = vi.fn();
  const batchGetOrFetchDatabases = vi.fn();
  const getOrFetchSheetByName = vi.fn();
  const getDBGroupByName = vi.fn();
  const getOrFetchDBGroupByName = vi.fn();
  const getEnvironmentByName = vi.fn();
  const onSelectedSpecIdChange = vi.fn();

  return {
    Sheet: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    SheetBody: vi.fn(
      ({
        children,
        className,
      }: {
        children: React.ReactNode;
        className?: string;
      }) => (
        <div
          className={`flex flex-1 flex-col overflow-y-auto px-6 py-4 ${className ?? ""}`}
          data-testid="targets-sheet-body"
        >
          {children}
        </div>
      )
    ),
    SheetContent: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div data-testid="targets-sheet-content">{children}</div>
    )),
    SheetHeader: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    SheetTitle: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    SearchInput: vi.fn(
      ({
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
      )
    ),
    Select: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    SelectContent: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    SelectItem: vi.fn(
      ({ children, value }: { children: React.ReactNode; value: string }) => (
        <div data-value={value}>{children}</div>
      )
    ),
    SelectTrigger: vi.fn(
      ({
        children,
        className,
      }: {
        children: React.ReactNode;
        className?: string;
      }) => <div className={className}>{children}</div>
    ),
    SelectValue: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    Switch: vi.fn(() => <div />),
    Tooltip: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    IssueDetailStatementSection: vi.fn(() => <div data-testid="statement" />),
    ChevronRight: vi.fn(() => <div />),
    ExternalLink: vi.fn(() => <div />),
    FolderTree: vi.fn(() => <div />),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    cn: vi.fn((...values: Array<string | false | null | undefined>) =>
      values.filter(Boolean).join(" ")
    ),
    routerResolve: vi.fn(() => ({
      fullPath: "/database-group",
      href: "/plan",
    })),
    getDatabaseByName,
    batchGetOrFetchDatabases,
    getOrFetchSheetByName,
    getDBGroupByName,
    getOrFetchDBGroupByName,
    getEnvironmentByName,
    listInstanceRoles,
    onSelectedSpecIdChange,
    useIssueDetailContext: vi.fn(),
    useIssueDetailSpecValidation: vi.fn(() => ({
      emptySpecIdSet: new Set<string>(),
    })),
    parseStatement: vi.fn(() => ({})),
    getGhostConfig: vi.fn(() => undefined),
    allowGhostForDatabase: vi.fn(() => false),
    getInstanceResource: vi.fn(() => ({
      engine: 0,
    })),
    instanceV1SupportsTransactionMode: vi.fn(() => false),
    getDefaultTransactionMode: vi.fn(() => false),
    getStatementSize: vi.fn(() => 0),
    getLocalSheetByName: vi.fn(() => ({
      contentSize: 0,
    })),
    getSheetStatement: vi.fn(() => ""),
    sheetNameOfSpec: vi.fn(() => "projects/db333/sheets/-1"),
    extractSheetUID: vi.fn(() => "-1"),
    extractDatabaseResourceName: vi.fn(() => ({
      instance: "instances/inst1",
      databaseName: "db1",
    })),
    extractInstanceResourceName: vi.fn(() => "inst1"),
    extractPlanUID: vi.fn(() => "123"),
    extractProjectResourceName: vi.fn(() => "db333"),
    buildPlanDeployRouteFromPlanName: vi.fn(() => ({
      name: "plan.detail",
      params: {},
    })),
    isValidDatabaseName: vi.fn((name: string) => name.startsWith("instances/")),
    isValidDatabaseGroupName: vi.fn(() => false),
    extractDatabaseGroupName: vi.fn(() => "group"),
    getProjectNameAndDatabaseGroupName: vi.fn(() => ["db333", "group"]),
  };
});

type IssueDetailDatabaseChangeViewComponent =
  typeof import("./IssueDetailDatabaseChangeView").IssueDetailDatabaseChangeView;
let IssueDetailDatabaseChangeView: IssueDetailDatabaseChangeViewComponent;

vi.mock("lucide-react", () => ({
  ChevronRight: mocks.ChevronRight,
  ExternalLink: mocks.ExternalLink,
  FolderTree: mocks.FolderTree,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/connect", () => ({
  instanceRoleServiceClientConnect: {
    listInstanceRoles: mocks.listInstanceRoles,
  },
}));

vi.mock("@/react/components/instance/constants", () => ({
  EngineIconPath: {},
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: mocks.Sheet,
  SheetBody: mocks.SheetBody,
  SheetContent: mocks.SheetContent,
  SheetHeader: mocks.SheetHeader,
  SheetTitle: mocks.SheetTitle,
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: mocks.SearchInput,
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: mocks.Select,
  SelectContent: mocks.SelectContent,
  SelectItem: mocks.SelectItem,
  SelectTrigger: mocks.SelectTrigger,
  SelectValue: mocks.SelectValue,
}));

vi.mock("@/react/components/ui/switch", () => ({
  Switch: mocks.Switch,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: mocks.Tooltip,
}));

vi.mock("@/react/lib/utils", () => ({
  cn: mocks.cn,
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    resolve: mocks.routerResolve,
  },
}));

vi.mock("@/react/router/routeHelpers", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router/routeHelpers")>()),
  buildPlanDeployRouteFromPlanName: mocks.buildPlanDeployRouteFromPlanName,
}));

vi.mock("@/store", () => ({
  getProjectNameAndDatabaseGroupName: mocks.getProjectNameAndDatabaseGroupName,
}));

vi.mock("@/types", () => ({
  isValidDatabaseGroupName: mocks.isValidDatabaseGroupName,
  isValidDatabaseName: mocks.isValidDatabaseName,
  unknownDatabaseGroup: () => ({ name: "", matchedDatabases: [] }),
}));

vi.mock("@/types/v1/database", () => ({
  unknownDatabase: () => ({ name: "instances/-/databases/-" }),
}));

vi.mock("@/react/stores/app", () => {
  // The migrated component reads cache state two ways: imperative
  // `useAppStore.getState().<method>()` and reactive selectors
  // `useAppStore((s) => s.dbGroupsByName[name])` / `s.databasesByName[name]`.
  // Delegate the map lookups to the existing mocks so per-test setup keeps
  // working.
  const dbGroupsByName = new Proxy({} as Record<string, unknown>, {
    get: (_t, key) =>
      typeof key === "string" ? mocks.getDBGroupByName(key) : undefined,
  });
  const databasesByName = new Proxy({} as Record<string, unknown>, {
    get: (_t, key) =>
      typeof key === "string" ? mocks.getDatabaseByName(key) : undefined,
  });
  const useAppStore = Object.assign(
    (
      selector: (s: {
        dbGroupsByName: Record<string, unknown>;
        databasesByName: Record<string, unknown>;
        environmentList: unknown[];
      }) => unknown
    ) => selector({ dbGroupsByName, databasesByName, environmentList: [] }),
    {
      getState: () => ({
        getDBGroupByName: mocks.getDBGroupByName,
        getOrFetchDBGroupByName: mocks.getOrFetchDBGroupByName,
        getOrFetchSheetByName: mocks.getOrFetchSheetByName,
        batchGetOrFetchDatabases: mocks.batchGetOrFetchDatabases,
        getEnvironmentByName: mocks.getEnvironmentByName,
      }),
    }
  );
  return { useAppStore };
});

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: mocks.extractDatabaseResourceName,
  extractInstanceResourceName: mocks.extractInstanceResourceName,
  extractPlanUID: mocks.extractPlanUID,
  extractProjectResourceName: mocks.extractProjectResourceName,
  getDefaultTransactionMode: mocks.getDefaultTransactionMode,
  getInstanceResource: mocks.getInstanceResource,
}));

vi.mock("@/utils/sheet", () => ({
  getStatementSize: mocks.getStatementSize,
}));

vi.mock("@/utils/v1/databaseGroup", () => ({
  extractDatabaseGroupName: mocks.extractDatabaseGroupName,
}));

vi.mock("@/utils/v1/instance", () => ({
  instanceV1SupportsTransactionMode: mocks.instanceV1SupportsTransactionMode,
}));

vi.mock("@/utils/v1/issue/plan", () => ({
  sheetNameOfSpec: mocks.sheetNameOfSpec,
}));

vi.mock("@/utils/v1/sheet", () => ({
  extractSheetUID: mocks.extractSheetUID,
  getSheetStatement: mocks.getSheetStatement,
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: mocks.useIssueDetailContext,
}));

vi.mock("../hooks/useIssueDetailSpecValidation", () => ({
  useIssueDetailSpecValidation: mocks.useIssueDetailSpecValidation,
}));

vi.mock("../utils/databaseChange", () => ({
  allowGhostForDatabase: mocks.allowGhostForDatabase,
  BACKUP_AVAILABLE_ENGINES: [],
  getGhostConfig: mocks.getGhostConfig,
  isDatabaseChangeSpec: () => true,
  parseStatement: mocks.parseStatement,
}));

vi.mock("../utils/localSheet", () => ({
  getLocalSheetByName: mocks.getLocalSheetByName,
}));

vi.mock("./IssueDetailStatementSection", () => ({
  IssueDetailStatementSection: mocks.IssueDetailStatementSection,
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
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

beforeEach(async () => {
  mocks.listInstanceRoles.mockReset();
  mocks.listInstanceRoles.mockResolvedValue({ roles: [] });
  mocks.getDatabaseByName.mockReset();
  mocks.getDatabaseByName.mockReturnValue({
    name: "instances/inst1/databases/db1",
    effectiveEnvironment: "",
    instanceResource: {
      environment: "environments/test",
      title: "Instance 1",
    },
  });
  mocks.batchGetOrFetchDatabases.mockReset();
  mocks.batchGetOrFetchDatabases.mockResolvedValue([]);
  mocks.getOrFetchSheetByName.mockReset();
  mocks.getOrFetchSheetByName.mockResolvedValue({
    contentSize: 0,
  });
  mocks.getDBGroupByName.mockReset();
  mocks.getDBGroupByName.mockReturnValue({
    name: "",
    matchedDatabases: [],
  });
  mocks.getOrFetchDBGroupByName.mockReset();
  mocks.getOrFetchDBGroupByName.mockResolvedValue({
    matchedDatabases: [],
  });
  mocks.getEnvironmentByName.mockReset();
  mocks.getEnvironmentByName.mockReturnValue({
    title: "Test",
  });
  mocks.onSelectedSpecIdChange.mockReset();
  mocks.useIssueDetailContext.mockReset();
  mocks.useIssueDetailContext.mockReturnValue({
    plan: {
      hasRollout: false,
      name: "projects/db333/plans/123",
      specs: [
        {
          id: "spec-1",
          config: {
            case: "changeDatabaseConfig",
            value: {
              enablePriorBackup: false,
              release: false,
              targets: ["instances/inst1/databases/db1"],
            },
          },
        },
      ],
    },
  });

  ({ IssueDetailDatabaseChangeView } = await import(
    "./IssueDetailDatabaseChangeView"
  ));
});

describe("IssueDetailDatabaseChangeView", () => {
  test("renders a database change issue without entering an update loop", () => {
    const { render, unmount } = renderIntoContainer(
      <IssueDetailDatabaseChangeView
        onSelectedSpecIdChange={mocks.onSelectedSpecIdChange}
        selectedSpecId="spec-1"
      />
    );

    expect(() => render()).not.toThrow();
    expect(mocks.listInstanceRoles).not.toHaveBeenCalled();

    unmount();
  });

  test("renders all targets in a single scrollable sheet list", () => {
    const targets = Array.from(
      { length: 21 },
      (_, index) => `instances/inst1/databases/db${index}`
    );
    mocks.useIssueDetailContext.mockReturnValue({
      plan: {
        hasRollout: false,
        name: "projects/db333/plans/123",
        specs: [
          {
            id: "spec-1",
            config: {
              case: "changeDatabaseConfig",
              value: {
                enablePriorBackup: false,
                release: false,
                targets,
              },
            },
          },
        ],
      },
    });

    const { container, render, unmount } = renderIntoContainer(
      <IssueDetailDatabaseChangeView
        onSelectedSpecIdChange={mocks.onSelectedSpecIdChange}
        selectedSpecId="spec-1"
      />
    );

    render();

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

    unmount();
  });
});
