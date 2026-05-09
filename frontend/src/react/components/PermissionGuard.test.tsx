import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { Permission } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  workspacePermissions: new Set<Permission>(),
  projectPermissions: new Set<Permission>(),
  loadWorkspacePermissionState: vi.fn(async () => {}),
  loadProjectIamPolicy: vi.fn(async (_project: string) => undefined),
}));

vi.mock("monaco-editor", () => ({}));

vi.mock(
  "@codingame/monaco-vscode-editor-api/vscode/src/vs/editor/standalone/browser/standalone-tokens.css",
  () => ({})
);

vi.mock("@/react/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      currentUser: { name: string };
      roles: unknown[];
      workspacePolicy: undefined;
      projectPoliciesByName: Record<string, unknown>;
      hasWorkspacePermission: (permission: Permission) => boolean;
      hasProjectPermission: (
        _project: Project,
        permission: Permission
      ) => boolean;
      loadWorkspacePermissionState: () => Promise<void>;
      loadProjectIamPolicy: (project: string) => Promise<undefined>;
    }) => unknown
  ) =>
    selector({
      currentUser: { name: "users/alice@example.com" },
      roles: [],
      workspacePolicy: undefined,
      projectPoliciesByName: {},
      hasWorkspacePermission: (permission) =>
        mocks.workspacePermissions.has(permission),
      hasProjectPermission: (_project, permission) =>
        mocks.workspacePermissions.has(permission) ||
        mocks.projectPermissions.has(permission),
      loadWorkspacePermissionState: mocks.loadWorkspacePermissionState,
      loadProjectIamPolicy: mocks.loadProjectIamPolicy,
    }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
  BlockTooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

import { PermissionGuard } from "./PermissionGuard";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  mocks.workspacePermissions = new Set();
  mocks.projectPermissions = new Set();
  mocks.loadWorkspacePermissionState.mockReset();
  mocks.loadWorkspacePermissionState.mockResolvedValue(undefined);
  mocks.loadProjectIamPolicy.mockReset();
  mocks.loadProjectIamPolicy.mockResolvedValue(undefined);
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

describe("PermissionGuard", () => {
  test("renders children enabled when the user has the workspace permission", async () => {
    mocks.workspacePermissions.add("bb.instances.create");

    await act(async () => {
      root.render(
        <PermissionGuard permissions={["bb.instances.create"]}>
          {({ disabled }) => <button disabled={disabled}>create</button>}
        </PermissionGuard>
      );
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(container.querySelector("button")?.disabled).toBe(false);
  });

  test("renders children enabled when the user has the project permission", async () => {
    const project = { name: "projects/demo" } as Project;
    mocks.projectPermissions.add("bb.sql.select");

    await act(async () => {
      root.render(
        <PermissionGuard permissions={["bb.sql.select"]} project={project}>
          {({ disabled }) => <button disabled={disabled}>connect</button>}
        </PermissionGuard>
      );
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(container.querySelector("button")?.disabled).toBe(false);
  });

  test("disables children when required permission is missing", async () => {
    await act(async () => {
      root.render(
        <PermissionGuard permissions={["bb.instances.create"]}>
          {({ disabled }) => <button disabled={disabled}>create</button>}
        </PermissionGuard>
      );
    });
    await act(async () => {
      await Promise.resolve();
    });

    expect(container.querySelector("button")?.disabled).toBe(true);
  });
});
