import type { ComponentProps } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Permission } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  route: {
    name: "workspace.settings",
    fullPath: "/settings",
    params: {},
    query: {},
    requiredPermissions: [] as Permission[],
  },
  workspacePermissions: new Set<Permission>(),
  projectPermissions: new Set<Permission>(),
  hasFeature: vi.fn(() => true),
}));

vi.mock("@/react/router", () => ({
  useCurrentRoute: () => mocks.route,
}));

vi.mock("@/react/components/ComponentPermissionGuard", () => {
  const basicPermissions: Permission[] = [
    "bb.roles.list",
    "bb.workspaces.getIamPolicy",
    "bb.settings.getWorkspaceProfile",
  ];
  const getState = ({
    permissions,
    project,
    checkBasicWorkspacePermissions,
  }: {
    permissions: Permission[];
    project?: unknown;
    checkBasicWorkspacePermissions?: boolean;
  }) => {
    const missedBasicPermissions = checkBasicWorkspacePermissions
      ? basicPermissions.filter((p) => !mocks.workspacePermissions.has(p))
      : [];
    const missedPermissions = permissions.filter((p) =>
      project
        ? !mocks.workspacePermissions.has(p) && !mocks.projectPermissions.has(p)
        : !mocks.workspacePermissions.has(p)
    );
    return {
      missedBasicPermissions,
      missedPermissions,
      permitted:
        missedBasicPermissions.length === 0 && missedPermissions.length === 0,
    };
  };
  return {
    useComponentPermissionState: getState,
    ComponentPermissionGuard: ({
      permissions,
      project,
      checkBasicWorkspacePermissions,
      path,
      className,
    }: {
      permissions: Permission[];
      project?: unknown;
      checkBasicWorkspacePermissions?: boolean;
      path?: string;
      className?: string;
    }) => {
      const { missedBasicPermissions, missedPermissions } = getState({
        permissions,
        project,
        checkBasicWorkspacePermissions,
      });
      const missed =
        missedBasicPermissions.length > 0
          ? missedBasicPermissions
          : missedPermissions;
      return (
        <div role="alert" className={className}>
          <span>{path}</span>
          {missed.map((permission) => (
            <span key={permission}>{permission}</span>
          ))}
        </div>
      );
    },
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

import { RoutePermissionGuardShell } from "./RoutePermissionGuardShell";

const BASIC_PERMISSIONS: Permission[] = [
  "bb.roles.list",
  "bb.workspaces.getIamPolicy",
  "bb.settings.getWorkspaceProfile",
];

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  mocks.route.requiredPermissions = [];
  mocks.route.fullPath = "/settings";
  mocks.workspacePermissions = new Set(BASIC_PERMISSIONS);
  mocks.projectPermissions = new Set();
  mocks.hasFeature.mockReturnValue(true);
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

const renderShell = async (
  props: Partial<ComponentProps<typeof RoutePermissionGuardShell>> = {}
) => {
  const onReady = vi.fn();
  await act(async () => {
    root.render(<RoutePermissionGuardShell onReady={onReady} {...props} />);
  });
  return onReady;
};

describe("RoutePermissionGuardShell", () => {
  test("exposes a content target when route permissions pass", async () => {
    mocks.route.requiredPermissions = ["bb.settings.get"];
    mocks.workspacePermissions.add("bb.settings.get");

    const onReady = await renderShell({ targetClassName: "h-full min-h-0" });

    expect(onReady).toHaveBeenLastCalledWith(expect.any(HTMLDivElement));
    expect(container.querySelector("[role='alert']")).toBeNull();
    expect(container.firstElementChild?.className).toBe("h-full min-h-0");
  });

  test("withholds the content target and renders an alert when permissions fail", async () => {
    mocks.route.requiredPermissions = ["bb.settings.set"];

    const onReady = await renderShell();

    expect(onReady).toHaveBeenLastCalledWith(null);
    expect(container.querySelector("[role='alert']")).not.toBeNull();
    expect(container.textContent).toContain("bb.settings.set");
  });

  test("checks basic workspace permissions before route permissions", async () => {
    mocks.workspacePermissions = new Set();
    mocks.route.requiredPermissions = ["bb.settings.set"];

    await renderShell();

    expect(container.textContent).toContain("bb.roles.list");
    expect(container.textContent).not.toContain("bb.settings.set");
  });

  test("re-emits the content target on route changes", async () => {
    mocks.route.requiredPermissions = ["bb.settings.get"];
    mocks.workspacePermissions.add("bb.settings.get");

    const onReady = vi.fn();
    await act(async () => {
      root.render(<RoutePermissionGuardShell onReady={onReady} />);
    });
    const initialCalls = onReady.mock.calls.length;
    expect(onReady).toHaveBeenLastCalledWith(expect.any(HTMLDivElement));

    mocks.route.fullPath = "/settings?next=foo";
    await act(async () => {
      root.render(<RoutePermissionGuardShell onReady={onReady} />);
    });

    expect(onReady).toHaveBeenCalledTimes(initialCalls + 1);
    expect(onReady).toHaveBeenLastCalledWith(expect.any(HTMLDivElement));
  });

  test("uses project permissions when a project is provided", async () => {
    mocks.route.requiredPermissions = ["bb.databases.get"];
    mocks.projectPermissions = new Set(["bb.databases.get"]);

    const onReady = await renderShell({
      project: {
        name: "projects/prod",
        allowRequestRole: true,
      } as never,
    });

    expect(onReady).toHaveBeenLastCalledWith(expect.any(HTMLDivElement));
    expect(container.querySelector("[role='alert']")).toBeNull();
  });
});
