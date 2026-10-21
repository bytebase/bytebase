import { fireEvent, render, waitFor } from "@testing-library/react";
import { StrictMode, type MouseEvent, type ReactNode } from "react";
import { describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_DETAIL } from "@/app/router/handles";
import { ProjectsPage } from "./ProjectsPage";

const mocks = vi.hoisted(() => ({
  fetchProjectList: vi.fn(),
  routerPush: vi.fn(),
  routerResolve: vi.fn(() => ({ fullPath: "/projects/visible" })),
  isSaaSMode: true,
}));

vi.mock("@/utils/featureGates", () => ({
  isSaaSOrDev: (isSaaSMode: boolean) => isSaaSMode,
}));
vi.mock("@/stores/app", () => {
  const state = {
    isLoggedIn: () => true,
    fetchProjectList: mocks.fetchProjectList,
    isSaaSMode: () => mocks.isSaaSMode,
  };
  return {
    useAppStore: Object.assign(
      (selector: (s: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/app/router", () => ({
  router: { push: mocks.routerPush, resolve: mocks.routerResolve },
  useCurrentRoute: () => ({ query: {} }),
}));
vi.mock("@/app/router/NavigationScrollRestoration", () => ({
  useListScrollRestorationLoadMore: () => {},
  markListScrollRestorationEntry: () => {},
}));
vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => null,
  getValueFromScopes: () => "ACTIVE",
}));
vi.mock("@/components/header/ProjectCreateDialog", () => ({
  ProjectCreateDialog: () => null,
}));
vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: () => null,
}));
vi.mock("@/components/ProjectTable", () => ({
  ProjectTable: ({
    projectList,
    onRowClick,
  }: {
    projectList: { name: string }[];
    onRowClick?: (project: { name: string }, event: MouseEvent<HTMLElement>) => void;
  }) => (
    <div>
      {projectList.map((project) => (
        <button
          key={project.name}
          type="button"
          onClick={(event) => onRowClick?.(project, event)}
        >
          {project.name}
        </button>
      ))}
    </div>
  ),
}));
vi.mock("@/components/WorkspacePageLayout", () => {
  const Container = ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  );
  return {
    WorkspacePageLayout: Container,
    WorkspacePageContent: Container,
    WorkspacePageFooter: Container,
    WorkspacePageToolbar: Container,
  };
});
vi.mock("@/hooks/usePagedData", () => ({ PagedTableFooter: () => null }));
vi.mock("@/hooks/useSessionPageSize", () => ({
  useSessionPageSize: () => [50, () => {}],
  getPageSizeOptions: () => [50],
}));
vi.mock("@/hooks/useURLSearchParam", () => {
  const params = { query: "", scopes: [] };
  return {
    legacyStateSearchParams: () => params,
    createAdvancedSearchParser: () => () => params,
    serializeAdvancedSearch: () => "",
    useURLSearchParam: () => [params, () => {}],
  };
});
vi.mock("@/lib/productIntro", () => ({
  useProductIntro: () => {},
  CREATE_PROJECT_PRODUCT_INTRO: "create",
  CONNECT_DATABASE_PRODUCT_INTRO: "connect",
  PRODUCT_INTRO_QUERY_KEY: "intro",
}));
vi.mock("@/utils", async (original) => ({
  ...(await original<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: () => false,
}));

describe("ProjectsPage", () => {
  test("opens Issues from a project row outside SaaS", async () => {
    mocks.isSaaSMode = false;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/visible" }],
      nextPageToken: "",
    });
    const view = render(<ProjectsPage />);
    const row = await view.findByRole("button", { name: "projects/visible" });

    fireEvent.click(row);

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.issue",
      params: { projectId: "visible" },
    });
  });

  test("opens Project Home from a project row and in a new tab", async () => {
    mocks.isSaaSMode = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/visible" }],
      nextPageToken: "",
    });
    const open = vi.spyOn(window, "open").mockImplementation(() => null);
    const view = render(<ProjectsPage />);
    const row = await view.findByRole("button", { name: "projects/visible" });
    const destination = {
      name: PROJECT_V1_ROUTE_DETAIL,
      params: { projectId: "visible" },
    };

    fireEvent.click(row);
    expect(mocks.routerPush).toHaveBeenCalledWith(destination);

    fireEvent.click(row, { metaKey: true });
    expect(mocks.routerResolve).toHaveBeenLastCalledWith(destination);
    expect(open).toHaveBeenCalledWith("/projects/visible", "_blank");

    view.unmount();
    open.mockRestore();
  });

  test("restarts the canceled initial fetch during StrictMode replay", async () => {
    const signals: AbortSignal[] = [];
    mocks.fetchProjectList.mockImplementation(
      async ({ signal }: { signal: AbortSignal }) => {
        signals.push(signal);
        await Promise.resolve();
        signal.throwIfAborted();
        return { projects: [{ name: "projects/visible" }], nextPageToken: "" };
      }
    );
    const view = render(
      <StrictMode>
        <ProjectsPage />
      </StrictMode>
    );
    await waitFor(() =>
      expect(view.getByText("projects/visible")).toBeTruthy()
    );
    expect(signals).toHaveLength(2);
    expect(signals[0].aborted).toBe(true);
    expect(signals[1].aborted).toBe(false);
    view.unmount();
    expect(signals[1].aborted).toBe(true);
  });
});
