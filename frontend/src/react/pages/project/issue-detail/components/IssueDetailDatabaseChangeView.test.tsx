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
    Dialog: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    DialogClose: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    DialogContent: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    DialogTitle: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    )),
    SearchInput: vi.fn((props: React.InputHTMLAttributes<HTMLInputElement>) => (
      <input {...props} />
    )),
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
    X: vi.fn(() => <div />),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    useVueState: vi.fn(<T,>(getter: () => T) => getter()),
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
  X: mocks.X,
}));

vi.mock("react-i18next", () => ({
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

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: mocks.Dialog,
  DialogClose: mocks.DialogClose,
  DialogContent: mocks.DialogContent,
  DialogTitle: mocks.DialogTitle,
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

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/react/lib/utils", () => ({
  cn: mocks.cn,
}));

vi.mock("@/router", () => ({
  router: {
    resolve: mocks.routerResolve,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL: "database-group.detail",
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS: "plan.detail.specs",
}));

vi.mock("@/router/dashboard/projectV1RouteHelpers", () => ({
  buildPlanDeployRouteFromPlanName: mocks.buildPlanDeployRouteFromPlanName,
}));

vi.mock("@/store", () => ({
  getProjectNameAndDatabaseGroupName: mocks.getProjectNameAndDatabaseGroupName,
  useDatabaseV1Store: () => ({
    getDatabaseByName: mocks.getDatabaseByName,
    batchGetOrFetchDatabases: mocks.batchGetOrFetchDatabases,
  }),
  useDBGroupStore: () => ({
    getDBGroupByName: mocks.getDBGroupByName,
    getOrFetchDBGroupByName: mocks.getOrFetchDBGroupByName,
  }),
  useEnvironmentV1Store: () => ({
    getEnvironmentByName: mocks.getEnvironmentByName,
  }),
  useSheetV1Store: () => ({
    getOrFetchSheetByName: mocks.getOrFetchSheetByName,
  }),
}));

vi.mock("@/types", () => ({
  isValidDatabaseGroupName: mocks.isValidDatabaseGroupName,
  isValidDatabaseName: mocks.isValidDatabaseName,
}));

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
  const root = createRoot(container);

  return {
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
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
});
