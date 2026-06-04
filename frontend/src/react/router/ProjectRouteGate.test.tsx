import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Permission } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  route: {
    name: "workspace.project.database",
    fullPath: "/projects/prod/databases",
    params: { projectId: "prod" },
    query: {},
    requiredPermissions: [] as Permission[],
  },
  params: { projectId: "prod" } as Record<string, string>,
  projectsByName: {} as Record<string, unknown>,
  projectErrorsByName: {} as Record<string, { message: string }>,
  workspacePermissions: new Set<Permission>(),
  projectPermissions: new Set<Permission>(),
  fetchProject: vi.fn(async (name: string) => mocks.projectsByName[name]),
  replace: vi.fn(),
  resolve: vi.fn(() => ({ fullPath: "/projects/prod" })),
  // Stable ref — mirrors the real `useNotify` (returns `state.notify`). A fresh
  // function each render would make the load effect's deps unstable and loop.
  notify: vi.fn(),
}));

vi.mock("react-router-dom", () => ({
  Outlet: () => <div data-testid="outlet" />,
  useParams: () => mocks.params,
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => mocks.route,
  useNavigate: () => ({ replace: mocks.replace, resolve: mocks.resolve }),
  WORKSPACE_ROUTE_LANDING: "workspace.landing",
  PROJECT_V1_ROUTE_DETAIL: "workspace.project.detail",
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useNotify: () => mocks.notify,
}));

vi.mock("@/react/pages/settings/RequestRoleSheet", () => ({
  RequestRoleSheet: () => <div data-testid="request-role-sheet" />,
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => <span data-testid="feature-badge" />,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    currentUser: { name: "users/alice@example.com" },
    roles: [],
    subscription: { plan: 3 },
    workspacePolicy: undefined,
    projectPoliciesByName: {},
    get projectsByName() {
      return mocks.projectsByName;
    },
    get projectErrorsByName() {
      return mocks.projectErrorsByName;
    },
    serverInfo: { defaultProject: "projects/default" },
    fetchProject: mocks.fetchProject,
    loadCurrentUser: vi.fn(async () => undefined),
    loadServerInfo: vi.fn(async () => undefined),
    loadWorkspacePermissionState: vi.fn(async () => {}),
    loadProjectIamPolicy: vi.fn(async () => undefined),
    loadSubscription: vi.fn(async () => undefined),
    setRecentProject: vi.fn(),
    removeRecentVisit: vi.fn(),
    hasWorkspacePermission: (permission: Permission) =>
      mocks.workspacePermissions.has(permission),
    hasProjectPermission: (_project: unknown, permission: Permission) =>
      mocks.workspacePermissions.has(permission) ||
      mocks.projectPermissions.has(permission),
  };
  const useAppStore = (selector?: (s: typeof state) => unknown) =>
    selector ? selector(state) : state;
  useAppStore.getState = () => state;
  return {
    useAppStore,
    projectResourceNameFromId: (id: string) => `projects/${id}`,
  };
});

import { ProjectRouteGate } from "./ProjectRouteGate";

const BASIC_PERMISSIONS: Permission[] = [
  "bb.roles.list",
  "bb.workspaces.getIamPolicy",
  "bb.settings.getWorkspaceProfile",
];

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  mocks.route.requiredPermissions = ["bb.projects.get", "bb.databases.list"];
  mocks.params = { projectId: "prod" };
  mocks.projectsByName = {
    "projects/prod": { name: "projects/prod", allowRequestRole: true },
  };
  mocks.projectErrorsByName = {};
  mocks.workspacePermissions = new Set(BASIC_PERMISSIONS);
  mocks.projectPermissions = new Set();
  mocks.fetchProject.mockClear();
  mocks.replace.mockClear();
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  document.body.removeChild(container);
});

const render = async () => {
  await act(async () => {
    root.render(<ProjectRouteGate />);
  });
  // Flush the async project-load effect + the permission-ready effect.
  for (let i = 0; i < 5; i++) {
    await act(async () => {
      await Promise.resolve();
    });
  }
};

describe("ProjectRouteGate", () => {
  test("renders the Outlet when project route permissions pass", async () => {
    mocks.projectPermissions = new Set([
      "bb.projects.get",
      "bb.databases.list",
    ]);

    await render();

    expect(container.querySelector("[data-testid='outlet']")).not.toBeNull();
    expect(container.querySelector("[role='alert']")).toBeNull();
    expect(mocks.fetchProject).toHaveBeenCalledWith("projects/prod");
  });

  test("renders the permission-denied fallback when a route permission is missing", async () => {
    // Has bb.projects.get (so the project loads) but lacks bb.databases.list.
    mocks.projectPermissions = new Set(["bb.projects.get"]);

    await render();

    expect(container.querySelector("[data-testid='outlet']")).toBeNull();
    expect(container.querySelector("[role='alert']")).not.toBeNull();
    expect(container.textContent).toContain("bb.databases.list");
  });

  test("redirects to landing when the project cannot be loaded", async () => {
    mocks.projectsByName = {};
    mocks.projectErrorsByName = {};

    await render();

    expect(mocks.replace).toHaveBeenCalledWith({ name: "workspace.landing" });
    expect(container.querySelector("[data-testid='outlet']")).toBeNull();
  });
});
