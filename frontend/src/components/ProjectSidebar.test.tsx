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
  dev: true,
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
}));

const t = vi.hoisted(
  () => (key: string, options?: { count?: number }) =>
    ({
      "common.database": "Database",
      "common.databases": "Databases",
      "common.groups": "Groups",
      "common.issues": "Issues",
      "common.instances": "Instances",
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

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  isDev: () => mocks.dev,
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
  mocks.dev = true;
  ({ ProjectSidebar } = await import("./ProjectSidebar"));
});

describe("ProjectSidebar", () => {
  test("renders project instances before databases", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    const labels = Array.from(container.querySelectorAll("a")).map((link) =>
      link.textContent?.trim()
    );
    expect(labels.indexOf("Instances")).toBeGreaterThan(-1);
    expect(labels.indexOf("Instances")).toBeLessThan(
      labels.indexOf("Databases")
    );

    unmount();
  });

  test("hides project instances for the default project", () => {
    mocks.defaultProject = "projects/sample";
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    expect(container.textContent).not.toContain("Instances");

    unmount();
  });

  test("hides project instances in release builds", () => {
    mocks.dev = false;
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    expect(container.textContent).not.toContain("Instances");

    unmount();
  });

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

  test("does not render the workspace logo in the sidebar", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    expect(container.querySelector("nav > a > img")).toBeNull();
    expect(
      container.querySelector("nav > div:last-child")?.className
    ).toContain("pt-3");

    unmount();
  });
});
