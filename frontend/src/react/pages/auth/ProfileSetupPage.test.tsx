import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { Workspace } from "@/types/proto-es/v1/workspace_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const componentDir = dirname(fileURLToPath(import.meta.url));

const mocks = vi.hoisted(() => ({
  canUpdateWorkspace: true,
  currentUser: {
    name: "users/ed@example.com",
    email: "ed@example.com",
    title: "Ed",
  } as User,
  workspace: {
    name: "workspaces/ws1",
    title: "Workspace One",
  } as Workspace,
  workspacePolicy: {
    bindings: [
      {
        role: "roles/workspaceAdmin",
        members: ["users/ed@example.com"],
      },
    ],
  } as IamPolicy,
  updateUser: vi.fn(),
  updateWorkspace: vi.fn(),
  createProject: vi.fn(),
  setRecentProject: vi.fn(),
  routerReplace: vi.fn(),
  pushNotification: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => false),
  canCreateProject: true,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
  useWorkspace: () => mocks.workspace,
  useWorkspacePermission: (permission: string) =>
    permission === "bb.projects.create"
      ? mocks.canCreateProject
      : mocks.canUpdateWorkspace,
  useCreateProject: () => ({
    createProject: mocks.createProject,
    setRecentProject: mocks.setRecentProject,
  }),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      workspacePolicy: IamPolicy;
      updateUser: typeof mocks.updateUser;
      updateWorkspace: typeof mocks.updateWorkspace;
    }) => unknown
  ) =>
    selector({
      workspacePolicy: mocks.workspacePolicy,
      updateUser: mocks.updateUser,
      updateWorkspace: mocks.updateWorkspace,
    }),
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    replace: mocks.routerReplace,
  },
}));

vi.mock("@/react/components/ResourceIdField", async () => {
  const React = await import("react");
  return {
    ResourceIdField: ({
      value,
      resourceTitle,
      onChange,
      onValidationChange,
    }: {
      value: string;
      resourceTitle?: string;
      onChange?: (value: string) => void;
      onValidationChange?: (valid: boolean) => void;
    }) => {
      React.useEffect(() => {
        if (!resourceTitle) return;
        onChange?.("new-project");
        onValidationChange?.(true);
      }, [onChange, onValidationChange, resourceTitle]);
      return <input data-testid="project-resource-id" readOnly value={value} />;
    },
  };
});

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils", () => ({
  extractProjectResourceName: (name: string) => name.split("/").pop() ?? "",
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/react/components/UserAvatar", () => ({
  UserAvatar: () => null,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) =>
      (
        ({
          "settings.profile.setup-title": "Welcome! Set up your workspace",
          "settings.profile.setup-first-project": "Setup 1st project",
          "settings.profile.default-project-name": "New project",
          "settings.profile.create-project-description":
            "Optional. If you create a project here, we will take you to its database page next.",
        }) as Record<string, string>
      )[key] ?? key,
  }),
}));

let ProfileSetupPage: typeof import("./ProfileSetupPage").ProfileSetupPage;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
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
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.canUpdateWorkspace = true;
  mocks.canCreateProject = true;
  mocks.workspacePolicy = {
    bindings: [
      {
        role: "roles/workspaceAdmin",
        members: ["users/ed@example.com"],
      },
    ],
  } as IamPolicy;
  mocks.createProject.mockResolvedValue({
    name: "projects/new-project",
    title: "New Project",
  });
  ({ ProfileSetupPage } = await import("./ProfileSetupPage"));
});

describe("ProfileSetupPage", () => {
  test("uses the shared product intro query key after creating a project", () => {
    const source = readFileSync(join(componentDir, "ProfileSetupPage.tsx"), {
      encoding: "utf8",
    });

    expect(source).toContain(
      "query: { [PRODUCT_INTRO_QUERY_KEY]: CONNECT_DATABASE_PRODUCT_INTRO }"
    );
    expect(source).toContain(
      "query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO }"
    );
  });

  test("shows workspace name when the sole member can update the workspace", () => {
    const page = renderIntoContainer(<ProfileSetupPage />);

    page.render();

    expect(page.container.textContent).toContain(
      "Welcome! Set up your workspace"
    );
    expect(page.container.textContent).toContain(
      "settings.profile.workspace-name"
    );
    const displayNameInput = page.container.querySelector(
      "[data-testid='profile-display-name']"
    );
    const workspaceNameInput = page.container.querySelector(
      "[data-testid='profile-workspace-title']"
    );
    expect(displayNameInput).toBeTruthy();
    expect(workspaceNameInput).toBeTruthy();
    expect(displayNameInput?.closest("[data-slot='form-field']")).toBeTruthy();
    expect(
      workspaceNameInput?.closest("[data-slot='form-field']")
    ).toBeTruthy();
    expect(mocks.hasWorkspacePermissionV2).not.toHaveBeenCalled();

    page.unmount();
  });

  test("hides workspace name for users joining an existing workspace", () => {
    mocks.workspacePolicy = {
      bindings: [
        {
          role: "roles/workspaceAdmin",
          members: ["users/ed@example.com"],
        },
        {
          role: "roles/workspaceMember",
          members: ["users/teammate@example.com"],
        },
      ],
    } as IamPolicy;
    const page = renderIntoContainer(<ProfileSetupPage />);

    page.render();

    expect(page.container.textContent).not.toContain(
      "settings.profile.workspace-name"
    );

    page.unmount();
  });

  test("can optionally create a project and continue to its databases page", async () => {
    const page = renderIntoContainer(<ProfileSetupPage />);

    page.render();

    expect(page.container.textContent).toContain("Setup 1st project");
    expect(page.container.textContent).not.toContain(
      "settings.profile.create-project-description"
    );
    expect(page.container.textContent).not.toContain(
      "Optional. If you create a project here, we will take you to its database page next."
    );
    expect(
      page.container.querySelector("[data-testid='create-project-toggle']")
    ).toBeNull();

    const projectNameInput = page.container.querySelector(
      "[data-testid='profile-project-title']"
    ) as HTMLInputElement;
    expect(projectNameInput.value).toBe("New project");
    expect(
      page.container.querySelector("[data-testid='project-resource-id']")
    ).toBeTruthy();
    expect(
      projectNameInput.closest("[data-slot='form-field']")?.className
    ).not.toContain("border-control-border");
    expect(
      [...page.container.querySelectorAll("[data-slot='form-field']")].some(
        (field) => field.className.includes("border-control-border")
      )
    ).toBe(false);

    const save = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.save")
    ) as HTMLButtonElement;
    await act(async () => {
      save.click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.createProject).toHaveBeenCalledWith(
      "New project",
      "new-project"
    );
    expect(mocks.setRecentProject).toHaveBeenCalledWith("projects/new-project");
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "new-project" },
      query: { intro: "connect-database" },
    });

    page.unmount();
  });

  test("does not create a project when the optional project name is empty", async () => {
    const page = renderIntoContainer(<ProfileSetupPage />);

    page.render();

    const projectNameInput = page.container.querySelector(
      "[data-testid='profile-project-title']"
    ) as HTMLInputElement;
    await act(async () => {
      fireEvent.change(projectNameInput, { target: { value: "" } });
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(
      page.container.querySelector("[data-testid='project-resource-id']")
    ).toBeNull();

    const save = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.save")
    ) as HTMLButtonElement;
    await act(async () => {
      save.click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.createProject).not.toHaveBeenCalled();
    expect(mocks.setRecentProject).not.toHaveBeenCalled();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project",
      query: { intro: "create-project" },
    });

    page.unmount();
  });

  test("skips profile setup to the projects page with create project highlighted", async () => {
    const page = renderIntoContainer(<ProfileSetupPage />);

    page.render();

    const skip = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("settings.profile.setup-skip")
    ) as HTMLButtonElement;
    await act(async () => {
      skip.click();
      await Promise.resolve();
    });

    expect(mocks.createProject).not.toHaveBeenCalled();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project",
      query: { intro: "create-project" },
    });

    page.unmount();
  });
});
