import { render, waitFor } from "@testing-library/react";
import { StrictMode, type ReactNode } from "react";
import { describe, expect, test, vi } from "vitest";
import { ProjectsPage } from "./ProjectsPage";

const mocks = vi.hoisted(() => ({ fetchProjectList: vi.fn() }));
vi.mock("@/stores/app", () => {
  const state = {
    isLoggedIn: () => true,
    fetchProjectList: mocks.fetchProjectList,
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
  router: {},
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
  ProjectTable: ({ projectList }: { projectList: { name: string }[] }) => (
    <div>{projectList.map((p) => p.name).join(",")}</div>
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
