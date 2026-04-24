import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  DATABASE_ROUTE_DASHBOARD,
  SQL_EDITOR_HOME_MODULE,
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MEMBERS,
} from "@/react/router";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  currentRoute: {
    name: "workspace.landing",
    fullPath: "/",
    params: {},
    query: {},
  },
  databaseChangeMode: 1,
  isSaaSMode: false,
  workspaceLogo: "",
  record: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(({ name }: { name: string }) => ({
    fullPath: `/${name}`,
  })),
}));

const t = vi.hoisted(
  () => (key: string) =>
    ({
      "common.home": "Home",
      "common.projects": "Projects",
      "common.instances": "Instances",
      "common.databases": "Databases",
      "common.environments": "Environments",
      "common.users": "Users",
      "common.settings": "Settings",
      "settings.sidebar.iam-and-admin": "IAM & Admin",
      "settings.members.service-accounts": "Service Accounts",
      "settings.members.workload-identities": "Workload Identities",
      "settings.sidebar.members": "Members",
      "settings.members.groups.self": "Groups",
      "settings.sidebar.custom-roles": "Custom Roles",
      "settings.sidebar.sso": "SSO",
      "settings.sidebar.audit-log": "Audit Log",
      "sql-review.title": "SQL Review",
      "custom-approval.risk.self": "Risk Assessment",
      "custom-approval.self": "Custom Approval",
      "settings.sidebar.data-access": "Data Access",
      "settings.sensitive-data.semantic-types.self": "Semantic Types",
      "settings.sidebar.data-classification": "Data Classification",
      "settings.sidebar.global-masking": "Global Masking",
      "settings.sidebar.integration": "Integration",
      "settings.sidebar.im-integration": "IM Integration",
      "settings.sidebar.mcp": "MCP",
      "settings.sidebar.general": "General",
      "settings.sidebar.subscription": "Subscription",
    })[key] ?? key
);

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t,
  }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useAppFeature: () => mocks.databaseChangeMode,
  useIsSaaSMode: () => mocks.isSaaSMode,
  useWorkspace: () => ({
    logo: mocks.workspaceLogo,
  }),
  useRecentVisit: () => ({
    record: mocks.record,
  }),
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => mocks.currentRoute,
  useNavigate: () => ({
    push: mocks.push,
    resolve: mocks.resolve,
  }),
  DATABASE_ROUTE_DASHBOARD: "workspace.database",
  ENVIRONMENT_V1_ROUTE_DASHBOARD: "workspace.environment",
  INSTANCE_ROUTE_DASHBOARD: "workspace.instance",
  PROJECT_V1_ROUTE_DASHBOARD: "workspace.project",
  SETTING_ROUTE_WORKSPACE_GENERAL: "setting.workspace.general",
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION: "setting.workspace.subscription",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  WORKSPACE_ROUTE_AUDIT_LOG: "workspace.audit-log",
  WORKSPACE_ROUTE_CUSTOM_APPROVAL: "workspace.custom-approval",
  WORKSPACE_ROUTE_DATA_CLASSIFICATION: "workspace.data-classification",
  WORKSPACE_ROUTE_GLOBAL_MASKING: "workspace.global-masking",
  WORKSPACE_ROUTE_GROUPS: "workspace.groups",
  WORKSPACE_ROUTE_IDENTITY_PROVIDERS: "workspace.identity-providers",
  WORKSPACE_ROUTE_IM: "workspace.integration.im",
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  WORKSPACE_ROUTE_MCP: "workspace.integration.mcp",
  WORKSPACE_ROUTE_MEMBERS: "workspace.members",
  WORKSPACE_ROUTE_RISK_ASSESSMENT: "workspace.risk-assessment",
  WORKSPACE_ROUTE_ROLES: "workspace.roles",
  WORKSPACE_ROUTE_SEMANTIC_TYPES: "workspace.semantic-types",
  WORKSPACE_ROUTE_SERVICE_ACCOUNTS: "workspace.service-accounts",
  WORKSPACE_ROUTE_SQL_REVIEW: "workspace.sql-review",
  WORKSPACE_ROUTE_USER_PROFILE: "workspace.user-profile",
  WORKSPACE_ROUTE_USERS: "workspace.users",
  WORKSPACE_ROUTE_WORKLOAD_IDENTITIES: "workspace.workload-identities",
}));

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

vi.mock("./WorkspaceSwitcher", () => ({
  WorkspaceSwitcher: () => <div data-testid="workspace-switcher" />,
}));

let DashboardSidebar: typeof import("./DashboardSidebar").DashboardSidebar;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.currentRoute.name = WORKSPACE_ROUTE_LANDING;
  mocks.currentRoute.fullPath = "/";
  mocks.databaseChangeMode = DatabaseChangeMode.PIPELINE;
  mocks.isSaaSMode = false;
  mocks.workspaceLogo = "";
  ({ DashboardSidebar } = await import("./DashboardSidebar"));
});

describe("DashboardSidebar", () => {
  test("renders non-SaaS IAM routes and hides SaaS-only exclusions", () => {
    mocks.currentRoute.name = WORKSPACE_ROUTE_MEMBERS;
    const { container, render, unmount } = renderIntoContainer(
      <DashboardSidebar />
    );
    render();

    expect(container.textContent).toContain("Users");
    expect(container.textContent).toContain("Service Accounts");
    expect(container.textContent).toContain("Workload Identities");

    unmount();

    mocks.isSaaSMode = true;
    const saas = renderIntoContainer(<DashboardSidebar />);
    saas.render();

    expect(saas.container.textContent).not.toContain("Users");
    expect(saas.container.textContent).not.toContain("Service Accounts");
    expect(saas.container.textContent).not.toContain("Workload Identities");
    expect(saas.container.textContent).toContain("Members");

    saas.unmount();
  });

  test("uses database-change mode for the logo route and records visits", () => {
    mocks.databaseChangeMode = DatabaseChangeMode.EDITOR;
    mocks.workspaceLogo = "https://example.com/logo.png";
    const { container, render, unmount } = renderIntoContainer(
      <DashboardSidebar />
    );
    render();

    const logoLink = container.querySelector<HTMLAnchorElement>("nav > a");
    expect(logoLink?.getAttribute("href")).toBe(`/${SQL_EDITOR_HOME_MODULE}`);
    expect(logoLink?.querySelector("img")?.getAttribute("src")).toBe(
      "https://example.com/logo.png"
    );

    act(() => {
      logoLink?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(mocks.record).toHaveBeenCalledWith(`/${SQL_EDITOR_HOME_MODULE}`);
    expect(mocks.push).toHaveBeenCalledWith({ name: SQL_EDITOR_HOME_MODULE });

    unmount();
  });

  test("highlights the active route", () => {
    mocks.currentRoute.name = DATABASE_ROUTE_DASHBOARD;
    const { container, render, unmount } = renderIntoContainer(
      <DashboardSidebar />
    );
    render();

    const databaseLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent?.includes("Databases")
    );
    expect(databaseLink?.className).toContain("router-link-active");

    unmount();
  });

  test("leaves modifier-click navigation to the browser", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DashboardSidebar />
    );
    render();

    const projectsLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent?.includes("Projects")
    );
    projectsLink?.addEventListener("click", (event) => event.preventDefault());
    const event = new MouseEvent("click", {
      bubbles: true,
      cancelable: true,
      ctrlKey: true,
    });
    projectsLink?.dispatchEvent(event);

    expect(mocks.push).not.toHaveBeenCalled();

    unmount();
  });
});
