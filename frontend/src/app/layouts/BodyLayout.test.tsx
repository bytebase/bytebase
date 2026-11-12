import { act, useEffect, useState, type ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_LANDING,
} from "@/app/router/handles";
import { useAppStore } from "@/stores/app";
import type { Permission } from "@/types/iam";
import { BodyLayout } from "./BodyLayout";

const mocks = vi.hoisted(() => ({
  permissions: new Set<Permission>(),
  loadWorkspacePermissionState: vi.fn<() => Promise<void>>(),
  loadProjectIamPolicy: vi.fn(async () => undefined),
  loadSubscription: vi.fn(async () => undefined),
}));

vi.mock("@/stores/app", async () => {
  const { create } = await import("zustand");
  return {
    useAppStore: create(() => ({
      currentUser: { name: "users/developer@example.com" },
      roles: [],
      workspacePolicy: undefined,
      projectPoliciesByName: {},
      hasWorkspacePermission: (permission: Permission) =>
        mocks.permissions.has(permission),
      hasProjectPermission: (_project: unknown, permission: Permission) =>
        mocks.permissions.has(permission),
      loadWorkspacePermissionState: mocks.loadWorkspacePermissionState,
      loadProjectIamPolicy: mocks.loadProjectIamPolicy,
      loadSubscription: mocks.loadSubscription,
    })),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/header/DashboardHeader", () => ({
  DashboardHeader: () => null,
}));

vi.mock("@/components/DashboardSidebar", () => ({
  DashboardSidebar: () => null,
}));

vi.mock("@/components/ProjectSidebar", () => ({
  ProjectSidebar: () => null,
}));

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));

vi.mock("@/modules/workspace-setup-guide/GuideContextProvider", () => ({
  GuideContextProvider: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/modules/workspace-setup-guide/WorkspaceSetupGuide", () => ({
  WorkspaceSetupGuide: () => null,
}));

beforeEach(() => {
  mocks.permissions = new Set([
    "bb.roles.list",
    "bb.workspaces.getIamPolicy",
    "bb.settings.getWorkspaceProfile",
  ]);
  mocks.loadWorkspacePermissionState.mockReset();
  mocks.loadWorkspacePermissionState.mockResolvedValue(undefined);
});

function renderLayout(initialPath = "/landing") {
  const mountedPaths: string[] = [];
  function Page({ name }: { name: string }) {
    const [count, setCount] = useState(0);
    useEffect(() => {
      mountedPaths.push(name);
    }, [name]);
    return (
      <button type="button" onClick={() => setCount((value) => value + 1)}>
        {name}: {count}
      </button>
    );
  }

  const router = createMemoryRouter(
    [
      {
        path: "/",
        Component: BodyLayout,
        children: [
          {
            path: "landing",
            element: <Page name="Landing" />,
            handle: { name: WORKSPACE_ROUTE_LANDING },
          },
          {
            path: "instances",
            element: <Page name="Instances" />,
            handle: {
              name: INSTANCE_ROUTE_DASHBOARD,
              requiredPermissionList: () => ["bb.instances.list"],
            },
          },
          {
            path: "projects",
            element: <Page name="Projects" />,
            handle: { name: PROJECT_V1_ROUTE_DASHBOARD },
          },
        ],
      },
    ],
    { initialEntries: [initialPath] }
  );
  render(<RouterProvider router={router} />);
  return { router, mountedPaths };
}

describe("BodyLayout", () => {
  test("does not mount a forbidden page on direct entry", async () => {
    const { mountedPaths } = renderLayout("/instances");

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "bb.instances.list"
    );
    expect(mountedPaths).toEqual([]);
  });

  test("does not mount a forbidden page when navigating from an allowed page", async () => {
    const { router, mountedPaths } = renderLayout();
    await screen.findByRole("button", { name: "Landing: 0" });

    await act(async () => {
      await router.navigate("/instances");
    });

    expect(screen.getByRole("alert")).toHaveTextContent("bb.instances.list");
    expect(mountedPaths).toEqual(["Landing"]);
  });

  test("waits for permission data before mounting page content", async () => {
    let resolvePermissions: () => void = () => {};
    mocks.loadWorkspacePermissionState.mockReturnValue(
      new Promise<void>((resolve) => {
        resolvePermissions = resolve;
      })
    );
    const { mountedPaths } = renderLayout();

    expect(screen.queryByRole("button", { name: "Landing: 0" })).toBeNull();
    expect(mountedPaths).toEqual([]);

    await act(async () => resolvePermissions());

    expect(screen.getByRole("button", { name: "Landing: 0" })).toBeVisible();
    expect(mountedPaths).toEqual(["Landing"]);
  });

  test("renders an allowed destination after a forbidden page", async () => {
    const { router, mountedPaths } = renderLayout("/instances");
    await screen.findByRole("alert");

    await act(async () => {
      await router.navigate("/projects");
    });

    expect(screen.getByRole("button", { name: "Projects: 0" })).toBeVisible();
    expect(mountedPaths).toEqual(["Projects"]);
  });

  test("keeps allowed page state across query and hash navigation", async () => {
    const { router, mountedPaths } = renderLayout();
    const button = await screen.findByRole("button", { name: "Landing: 0" });
    act(() => button.click());

    await act(async () => {
      await router.navigate("/landing?view=recent#activity");
    });

    expect(screen.getByRole("button", { name: "Landing: 1" })).toBeVisible();
    expect(mountedPaths).toEqual(["Landing"]);
  });

  test("removes page content when its permission is revoked", async () => {
    mocks.permissions.add("bb.instances.list");
    renderLayout("/instances");
    await screen.findByRole("button", { name: "Instances: 0" });

    act(() => {
      mocks.permissions.delete("bb.instances.list");
      useAppStore.setState({ roles: [] });
    });

    expect(screen.getByRole("alert")).toHaveTextContent("bb.instances.list");
    expect(screen.queryByRole("button", { name: "Instances: 0" })).toBeNull();
  });
});
