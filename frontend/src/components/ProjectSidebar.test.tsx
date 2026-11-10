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
    } as Record<string, string>,
    query: {},
  },
  viewedPlan: undefined as
    | { name: string; isReleaseBacked: boolean }
    | undefined,
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
      "common.home": "Home",
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
      viewedPlan: typeof mocks.viewedPlan;
    }) => unknown
  ) =>
    selector({
      getOrFetchProjectByName: mocks.getOrFetchProjectByName,
      serverInfo: {
        defaultProject: mocks.defaultProject,
      },
      viewedPlan: mocks.viewedPlan,
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
  mocks.viewedPlan = undefined;
  ({ ProjectSidebar } = await import("./ProjectSidebar"));
});

describe("ProjectSidebar", () => {
  test("shows Home as the active root route, including for the default project", () => {
    mocks.defaultProject = "projects/sample";
    mocks.currentRoute.name = "workspace.project.detail";
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    const home = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent?.trim() === "Home"
    );
    expect(home?.className).toContain("router-link-active");
    expect(home?.getAttribute("href")).toContain("workspace.project.detail");

    unmount();
  });

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

  test("renders project instances in release builds", () => {
    mocks.dev = false;
    const { container, render, unmount } = renderIntoContainer(
      <ProjectSidebar />
    );
    render();

    expect(container.textContent).toContain("Instances");

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

  describe("CI/CD entries", () => {
    const activeLabels = (container: HTMLElement) =>
      Array.from(container.querySelectorAll("a"))
        .filter((link) => link.className.includes("router-link-active"))
        .map((link) => link.textContent?.trim());

    const renderAt = (name: string, params: Record<string, string> = {}) => {
      mocks.currentRoute.name = name;
      mocks.currentRoute.params = { projectId: "sample", ...params };
      const view = renderIntoContainer(<ProjectSidebar />);
      view.render();
      return view;
    };

    test("lists Plans and Releases without a GitOps entry", () => {
      const { container, unmount } = renderAt("workspace.project.plan");

      const labels = Array.from(container.querySelectorAll("a")).map((link) =>
        link.textContent?.trim()
      );
      expect(labels).toEqual(expect.arrayContaining(["Plans", "Releases"]));
      expect(labels).not.toContain("GitOps");

      unmount();
    });

    test("highlights Releases on the GitOps setup page", () => {
      const { container, unmount } = renderAt("workspace.project.gitops");

      expect(activeLabels(container)).toEqual(["Releases"]);

      unmount();
    });

    test("highlights Releases on a GitOps Plan's detail page", () => {
      mocks.viewedPlan = {
        name: "projects/sample/plans/7",
        isReleaseBacked: true,
      };
      const { container, unmount } = renderAt(
        "workspace.project.plan.detail.rollout",
        { planId: "7" }
      );

      expect(activeLabels(container)).toEqual(["Releases"]);

      unmount();
    });

    test("highlights Plans on a ClickOps Plan's detail page", () => {
      mocks.viewedPlan = {
        name: "projects/sample/plans/7",
        isReleaseBacked: false,
      };
      const { container, unmount } = renderAt("workspace.project.plan.detail", {
        planId: "7",
      });

      expect(activeLabels(container)).toEqual(["Plans"]);

      unmount();
    });

    test("ignores a GitOps Plan viewed before the current one", () => {
      mocks.viewedPlan = {
        name: "projects/sample/plans/6",
        isReleaseBacked: true,
      };
      const { container, unmount } = renderAt("workspace.project.plan.detail", {
        planId: "7",
      });

      expect(activeLabels(container)).toEqual(["Plans"]);

      unmount();
    });

    test("hides CI/CD for the default project", () => {
      mocks.defaultProject = "projects/sample";
      const { container, unmount } = renderAt("workspace.project.detail");

      expect(container.textContent).not.toContain("Plans");
      expect(container.textContent).not.toContain("Releases");

      unmount();
    });
  });
});
