import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  currentRoute: {
    name: "workspace.project.masking-exemption",
    fullPath: "/projects/sample/masking-exemption",
    params: {
      projectId: "sample",
    },
    query: {},
  },
  defaultProject: "",
  getOrFetchProjectByName: vi.fn(),
  record: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(
    (target: string | { name?: string; params?: Record<string, string> }) => {
      const name = typeof target === "string" ? target : target.name;
      return {
        href: `/${name ?? ""}`,
        fullPath: `/${name ?? ""}`,
      };
    }
  ),
  workspace: {
    logo: "",
  },
}));

const t = vi.hoisted(
  () => (key: string, options?: { count?: number }) =>
    ({
      "common.database": "Database",
      "common.databases": "Databases",
      "common.groups": "Groups",
      "common.issues": "Issues",
      "common.manage": "Manage",
      "common.members": options?.count === 2 ? "Members" : "Member",
      "common.setting": "Setting",
      "common.settings": "Settings",
      "common.webhooks": "Webhooks",
      "database.sync-schema.title": "Sync Schema",
      "gitops.self": "GitOps",
      "plan.plans": "Plans",
      "project.masking-exemption.self": "Masking Exemptions",
      "release.releases": "Releases",
      "settings.members.service-accounts": "Service Accounts",
      "settings.members.workload-identities": "Workload Identities",
      "settings.sidebar.audit-log": "Audit Logs",
      "settings.sidebar.data-access": "Data Access",
      "sql-editor.access-grants": "Access Grants",
    })[key] ?? key
);

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({
    t,
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useWorkspace: () => mocks.workspace,
}));

vi.mock("@/hooks/useRecentVisit", () => ({
  useRecentVisit: () => ({
    record: mocks.record,
  }),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.push,
    resolve: mocks.resolve,
  },
  useCurrentRoute: () => mocks.currentRoute,
}));

vi.mock("@/stores/app", () => {
  const useAppStore = (
    selector: (state: {
      getOrFetchProjectByName: typeof mocks.getOrFetchProjectByName;
      serverInfo: { defaultProject: string };
    }) => unknown
  ) =>
    selector({
      getOrFetchProjectByName: mocks.getOrFetchProjectByName,
      serverInfo: {
        defaultProject: mocks.defaultProject,
      },
    });
  useAppStore.getState = () => ({
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
  });
  return { useAppStore };
});

vi.mock("@/assets/logo-full.svg", () => ({
  default: "/assets/logo-full.svg",
}));

let ProjectSidebar: typeof import("./ProjectSidebar").ProjectSidebar;

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
  mocks.currentRoute.name = "workspace.project.masking-exemption";
  mocks.currentRoute.params = {
    projectId: "sample",
  };
  mocks.defaultProject = "";
  mocks.workspace.logo = "";
  ({ ProjectSidebar } = await import("./ProjectSidebar"));
});

describe("ProjectSidebar", () => {
  test("renders the settings route with a plural label", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    const settingsLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent?.includes("Settings")
    );

    expect(settingsLink).toBeTruthy();

    unmount();
  });

  test("keeps child route labels on one line", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    const maskingExemptionsLink = Array.from(
      container.querySelectorAll("a")
    ).find((link) => link.textContent?.includes("Masking Exemptions"));

    expect(maskingExemptionsLink?.className).toContain("whitespace-nowrap");

    unmount();
  });

  test("gives the custom logo an explicit rendered size", () => {
    mocks.workspace.logo = "https://example.com/logo.png";
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    const logoLink = container.querySelector("nav > a");
    const logo = container.querySelector("nav > a img");
    expect(logoLink?.className).toContain("h-20");
    expect(logoLink?.className).toContain("w-full");
    expect(logo?.getAttribute("src")).toBe("https://example.com/logo.png");
    expect(logo?.className).toContain("h-full");
    expect(logo?.className).toContain("w-full");
    expect(logo?.className).toContain("max-w-44");
    expect(logo?.className).toContain("object-contain");

    unmount();
  });
});
