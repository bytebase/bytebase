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
  prepareSampleProjectInstance: vi.fn(),
  setRecentProject: vi.fn(),
  routerReplace: vi.fn(),
  captureMetric: vi.fn(),
  clearGuideWorkspaceUsage: vi.fn(),
  clearSelectedGuideScenarioId: vi.fn(),
  saveGuideWorkspaceUsage: vi.fn(),
  saveSelectedGuideScenarioId: vi.fn(),
  pushNotification: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => false),
  canCreateProject: true,
  sampleAvailable: true,
  sampleInstances: [] as Array<{ instance: string }>,
  isSaaSMode: false,
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: {
    captureMetric: mocks.captureMetric,
  },
}));

vi.mock("@/modules/workspace-setup-guide/selection", () => ({
  clearGuideWorkspaceUsage: mocks.clearGuideWorkspaceUsage,
  clearSelectedGuideScenarioId: mocks.clearSelectedGuideScenarioId,
  isGuideWorkspaceUsage: (value: unknown) =>
    value === "team" || value === "solo",
  isGuideScenarioId: (value: unknown) =>
    value === "query-data" || value === "create-database-change",
  saveGuideWorkspaceUsage: mocks.saveGuideWorkspaceUsage,
  saveSelectedGuideScenarioId: mocks.saveSelectedGuideScenarioId,
}));

vi.mock("@/hooks/useAppState", () => ({
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

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      workspacePolicy: IamPolicy;
      updateUser: typeof mocks.updateUser;
      updateWorkspace: typeof mocks.updateWorkspace;
      prepareSampleProjectInstance: typeof mocks.prepareSampleProjectInstance;
      serverInfo: {
        sample: { available: boolean; instances: Array<{ instance: string }> };
      };
      isSaaSMode: () => boolean;
    }) => unknown
  ) =>
    selector({
      workspacePolicy: mocks.workspacePolicy,
      updateUser: mocks.updateUser,
      updateWorkspace: mocks.updateWorkspace,
      prepareSampleProjectInstance: mocks.prepareSampleProjectInstance,
      serverInfo: {
        sample: {
          available: mocks.sampleAvailable,
          instances: mocks.sampleInstances,
        },
      },
      isSaaSMode: () => mocks.isSaaSMode,
    }),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    replace: mocks.routerReplace,
  },
}));

vi.mock("@/components/ResourceIdField", async () => {
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
      return (
        <input
          data-testid="project-resource-id"
          value={value}
          onChange={(event) => {
            onChange?.(event.target.value);
            onValidationChange?.(event.target.value.length > 0);
          }}
        />
      );
    },
  };
});

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils", () => ({
  extractInstanceResourceName: (name: string) =>
    name.match(/(?:^|\/)instances\/([^/]+)/)?.[1] ?? "",
  extractProjectResourceName: (name: string) => name.split("/").pop() ?? "",
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/components/UserAvatar", () => ({
  UserAvatar: () => <div data-testid="user-avatar" />,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) =>
      (
        ({
          "common.back": "Back",
          "settings.profile.setup-scenario.outcome-title":
            "What would you like to do with Bytebase?",
          "settings.profile.setup-scenario.create-database-change.title":
            "Create a database change",
          "settings.profile.setup-scenario.create-database-change.description":
            "Define a change and create its issue.",
          "settings.profile.setup-scenario.query-data.title":
            "Query data",
          "settings.profile.setup-scenario.query-data.description":
            "Open SQL Editor and run a statement.",
          "settings.profile.setup-scenario.workspace-usage.title":
            "Who will use Bytebase with you?",
          "settings.profile.setup-scenario.workspace-usage.team.title":
            "My team",
          "settings.profile.setup-scenario.workspace-usage.team.description":
            "I will work with other people.",
          "settings.profile.setup-scenario.workspace-usage.solo.title":
            "Just me",
          "settings.profile.setup-scenario.workspace-usage.solo.description":
            "I will use Bytebase on my own.",
          "settings.profile.setup-scenario.continue": "Continue",
          "settings.profile.setup-steps.scenario": "Choose a goal",
          "settings.profile.setup-steps.workspace": "Set up workspace",
          "settings.profile.setup-first-project": "Setup 1st project",
          "settings.profile.setup-skip": "I'll do this later",
          "settings.profile.enable-sample-databases":
            "Enable sample databases",
          "settings.profile.enable-sample-databases-query-data":
            "Enable sample databases to start querying immediately",
          "settings.profile.enable-sample-databases-create-change":
            "Enable sample databases as a safe change target",
          "settings.profile.setup-submit": "Setup my workspace",
          "instance.prepare-sample-instance-failed":
            "Failed to prepare Sample Project Instance.",
          "settings.profile.default-project-name": "New project",
          "settings.profile.create-project-description":
            "Optional. If you create a project here, we will take you to its database page next.",
        }) as Record<string, string>
      )[key] ?? key,
  }),
}));

let WorkspaceSetupPage: typeof import("./WorkspaceSetupPage").WorkspaceSetupPage;

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

const renderWorkspaceForm = () => {
  const page = renderIntoContainer(<WorkspaceSetupPage />);
  page.render();
  const continueWithoutSelection = [
    ...page.container.querySelectorAll("button"),
  ].find((button) => button.textContent === "Continue");
  if (!continueWithoutSelection) throw new Error("Continue not found");
  act(() => fireEvent.click(continueWithoutSelection));
  return page;
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.canUpdateWorkspace = true;
  mocks.canCreateProject = true;
  mocks.sampleAvailable = true;
  mocks.sampleInstances = [];
  mocks.isSaaSMode = false;
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
  mocks.prepareSampleProjectInstance.mockResolvedValue({
    name: "projects/new-project/instances/sample",
  });
  ({ WorkspaceSetupPage } = await import("./WorkspaceSetupPage"));
});

describe("WorkspaceSetupPage", () => {
  test("uses the shared step indicator for both setup steps", async () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();

    const initialIndicator = page.container.querySelector(
      "[data-slot='step-indicator']"
    );
    expect(initialIndicator).not.toBeNull();
    expect(
      [...initialIndicator!.querySelectorAll("li span")].map(
        (label) => label.textContent
      )
    ).toEqual(["Choose a goal", "Set up workspace"]);
    expect(initialIndicator!.querySelector("svg")).toBeNull();

    const queryOption = [...page.container.querySelectorAll("label")].find(
      (label) => label.textContent?.includes("Query data")
    )!;
    await act(async () => {
      fireEvent.click(queryOption);
      await Promise.resolve();
    });
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;
    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });

    const workspaceIndicator = page.container.querySelector(
      "[data-slot='step-indicator']"
    );
    expect(workspaceIndicator).not.toBeNull();
    expect(workspaceIndicator!.querySelector("svg")).not.toBeNull();
    page.unmount();
  });

  test("uses stable wizard footer actions for workspace setup", () => {
    const page = renderWorkspaceForm();
    const buttons = [...page.container.querySelectorAll("button")];
    const back = buttons.find((button) => button.textContent === "Back")!;
    const skip = buttons.find(
      (button) => button.textContent === "I'll do this later"
    )!;
    const submit = buttons.find(
      (button) => button.textContent === "Setup my workspace"
    )!;
    const footer = back.parentElement!;

    expect(footer.className).toContain("justify-between");
    expect(footer.className).toContain("gap-x-2");
    expect(footer.firstElementChild).toBe(back);
    expect(footer.lastElementChild).toContainElement(skip);
    expect(footer.lastElementChild).toContainElement(submit);
    expect(footer.lastElementChild?.textContent).toBe(
      "I'll do this laterSetup my workspace"
    );
    expect(footer.querySelector(".w-full")).toBeNull();
    page.unmount();
  });

  test("starts with two unanswered questionnaire groups", () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);

    page.render();

    expect(page.container.textContent).toContain(
      "What would you like to do with Bytebase?"
    );
    expect(page.container.querySelectorAll("[role='radio']")).toHaveLength(4);
    expect(page.container.textContent).toContain("Create a database change");
    expect(page.container.textContent).toContain("Who will use Bytebase with you?");
    expect(page.container.textContent).toContain("My team");
    expect(page.container.textContent).toContain("Just me");
    expect(page.container.textContent).not.toContain("Learn Bytebase Basics");
    expect(page.container.textContent).not.toContain("Setup 1st project");
    expect(page.container.textContent).not.toContain("Skip");

    page.unmount();
  });

  test("persists Create a database change when the questionnaire continues", async () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();
    const changeOption = [...page.container.querySelectorAll("label")].find(
      (label) => label.textContent?.includes("Create a database change")
    )!;

    await act(async () => {
      fireEvent.click(changeOption);
      await Promise.resolve();
    });
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;
    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });
    expect(page.container.textContent).toContain(
      "Enable sample databases as a safe change target"
    );
    expect(mocks.saveSelectedGuideScenarioId).toHaveBeenCalledWith(
      "create-database-change"
    );
    expect(mocks.captureMetric).not.toHaveBeenCalled();
    page.unmount();
  });

  test("persists both answers without recording before setup ends", async () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();
    const labels = [...page.container.querySelectorAll("label")];

    await act(async () => {
      fireEvent.click(
        labels.find((label) => label.textContent?.includes("Query data"))!
      );
      fireEvent.click(
        labels.find((label) => label.textContent?.includes("My team"))!
      );
      await Promise.resolve();
    });
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;
    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });

    expect(mocks.saveSelectedGuideScenarioId).toHaveBeenCalledWith(
      "query-data"
    );
    expect(mocks.saveGuideWorkspaceUsage).toHaveBeenCalledWith("team");
    expect(mocks.captureMetric).not.toHaveBeenCalled();
    page.unmount();
  });

  test("persists Query Data before optional workspace setup", async () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();
    const queryOption = [...page.container.querySelectorAll("label")].find(
      (label) => label.textContent?.includes("Query data")
    )!;

    await act(async () => {
      fireEvent.click(queryOption);
      await Promise.resolve();
    });
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;
    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });
    expect(mocks.saveSelectedGuideScenarioId).toHaveBeenCalledWith(
      "query-data"
    );
    expect(page.container.textContent).toContain(
      "Enable sample databases to start querying immediately"
    );

    const submit = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent?.includes("Setup my workspace")
    )!;
    await act(async () => {
      fireEvent.click(submit);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.saveSelectedGuideScenarioId).toHaveBeenCalledWith(
      "query-data"
    );
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "query-data",
        result: "finished",
        sample_enabled: true,
      },
    });
    page.unmount();
  });

  test("preserves an explicit selection when optional setup fails", async () => {
    mocks.updateUser.mockRejectedValueOnce(new Error("update failed"));
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();
    const changeOption = [...page.container.querySelectorAll("label")].find(
      (label) => label.textContent?.includes("Create a database change")
    )!;
    await act(async () => {
      fireEvent.click(changeOption);
      await Promise.resolve();
    });
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;
    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });
    const submit = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent?.includes("Setup my workspace")
    )!;

    await act(async () => {
      fireEvent.click(submit);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.saveSelectedGuideScenarioId).toHaveBeenCalledWith(
      "create-database-change"
    );
    expect(mocks.captureMetric).not.toHaveBeenCalled();
    page.unmount();
  });

  test("preserves the selected scenario when workspace setup is skipped", async () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();
    const queryOption = [...page.container.querySelectorAll("label")].find(
      (label) => label.textContent?.includes("Query data")
    )!;

    await act(async () => {
      fireEvent.click(queryOption);
      await Promise.resolve();
    });
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;
    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });
    const skipSetupButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "I'll do this later"
    )!;
    await act(async () => {
      fireEvent.click(skipSetupButton);
      await Promise.resolve();
    });

    expect(mocks.saveSelectedGuideScenarioId).toHaveBeenCalledWith(
      "query-data"
    );
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "query-data",
        result: "skipped",
        sample_enabled: false,
      },
    });
    expect(mocks.routerReplace).toHaveBeenCalled();
    page.unmount();
  });

  test("continuing without a scenario keeps scenario storage empty", async () => {
    const page = renderIntoContainer(<WorkspaceSetupPage />);
    page.render();
    const continueButton = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent === "Continue"
    )!;

    await act(async () => {
      fireEvent.click(continueButton);
      await Promise.resolve();
    });
    const submit = [...page.container.querySelectorAll("button")].find(
      (button) => button.textContent?.includes("Setup my workspace")
    )!;
    await act(async () => {
      fireEvent.click(submit);
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.clearSelectedGuideScenarioId).toHaveBeenCalledOnce();
    expect(mocks.clearGuideWorkspaceUsage).toHaveBeenCalledOnce();
    expect(mocks.saveSelectedGuideScenarioId).not.toHaveBeenCalled();
    expect(mocks.saveGuideWorkspaceUsage).not.toHaveBeenCalled();
    expect(mocks.captureMetric).toHaveBeenCalledTimes(2);
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "unselected",
        result: "finished",
        sample_enabled: true,
      },
    });
    page.unmount();
  });

  test("uses the shared product intro query key after creating a project", () => {
    const source = readFileSync(join(componentDir, "WorkspaceSetupPage.tsx"), {
      encoding: "utf8",
    });

    expect(source).toContain(
      "[PRODUCT_INTRO_QUERY_KEY]: CONNECT_DATABASE_PRODUCT_INTRO"
    );
    expect(source).toContain("PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO");
    expect(source).toContain(
      "query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO }"
    );
  });

  test("shows workspace name when the sole member can update the workspace", () => {
    const page = renderWorkspaceForm();

    expect(page.container.textContent).not.toContain(
      "Welcome! Set up your workspace"
    );
    expect(page.container.querySelector("[data-testid='user-avatar']")).toBeNull();
    const heading = page.container.querySelector("h1");
    expect(heading).toHaveTextContent("Set up workspace");
    expect(heading).toHaveClass("sr-only");
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
    const page = renderWorkspaceForm();

    expect(page.container.textContent).not.toContain(
      "settings.profile.workspace-name"
    );

    page.unmount();
  });

  test("can optionally create a project and continue to its databases page", async () => {
    const page = renderWorkspaceForm();

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

    const sampleCheckbox = page.container.querySelector(
      "[data-testid='enable-sample-databases']"
    ) as HTMLButtonElement;
    expect(sampleCheckbox).toBeChecked();
    expect(sampleCheckbox).toBeEnabled();
    expect(page.container.textContent).toContain("Enable sample databases");

    const save = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Setup my workspace")
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
    expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledWith(
      "projects/new-project"
    );
    expect(mocks.createProject.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.prepareSampleProjectInstance.mock.invocationCallOrder[0]
    );
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "sample instance requested",
      properties: { source: "workspace_setup" },
    });
    const sampleMetricCall = mocks.captureMetric.mock.calls.find(
      ([metric]) => metric.event === "sample instance requested"
    );
    expect(sampleMetricCall).toBeDefined();
    expect(sampleMetricCall![0]).toEqual({
      event: "sample instance requested",
      properties: { source: "workspace_setup" },
    });
    expect(mocks.captureMetric.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.prepareSampleProjectInstance.mock.invocationCallOrder[0]
    );
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "new-project" },
      query: {
        syncingInstance: "sample",
        intro: "project-instance-synced",
      },
    });

    page.unmount();
  });

  test("allows setup without creating a project when its title is cleared", async () => {
    const page = renderWorkspaceForm();

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
    const sampleCheckbox = page.container.querySelector(
      "[data-testid='enable-sample-databases']"
    ) as HTMLButtonElement;
    expect(sampleCheckbox).not.toBeChecked();
    expect(sampleCheckbox).toHaveAttribute("aria-disabled", "true");

    const save = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Setup my workspace")
    ) as HTMLButtonElement;
    await act(async () => {
      save.click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.createProject).not.toHaveBeenCalled();
    expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
    expect(mocks.captureMetric).not.toHaveBeenCalledWith({
      event: "sample instance requested",
      properties: expect.anything(),
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "unselected",
        result: "finished",
        sample_enabled: false,
      },
    });
    expect(mocks.setRecentProject).not.toHaveBeenCalled();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project",
      query: { intro: "create-project" },
    });

    page.unmount();
  });

  test("requires a project ID while the project title is present", async () => {
    const page = renderWorkspaceForm();

    const projectResourceIdInput = page.container.querySelector(
      "[data-testid='project-resource-id']"
    ) as HTMLInputElement;
    await act(async () => {
      fireEvent.change(projectResourceIdInput, { target: { value: "" } });
      await Promise.resolve();
    });

    const save = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Setup my workspace")
    ) as HTMLButtonElement;
    expect(save).toBeDisabled();

    page.unmount();
  });

  test("does not prepare sample databases when the user opts out", async () => {
    const page = renderWorkspaceForm();

    const sampleCheckbox = page.container.querySelector(
      "[data-testid='enable-sample-databases']"
    ) as HTMLButtonElement;
    await act(async () => {
      sampleCheckbox.click();
    });
    expect(sampleCheckbox).not.toBeChecked();

    const submit = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Setup my workspace")
    ) as HTMLButtonElement;
    await act(async () => {
      submit.click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.createProject).toHaveBeenCalled();
    expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "new-project" },
      query: { intro: "connect-database" },
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "unselected",
        result: "finished",
        sample_enabled: false,
      },
    });

    page.unmount();
  });

  test("finishes setup when sample database preparation fails", async () => {
    mocks.prepareSampleProjectInstance.mockRejectedValue(
      new Error("sample target unavailable")
    );
    const page = renderWorkspaceForm();

    const submit = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("Setup my workspace")
    ) as HTMLButtonElement;
    await act(async () => {
      submit.click();
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "CRITICAL",
        title: "Failed to prepare Sample Project Instance.",
      })
    );
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "new-project" },
      query: { intro: "connect-database" },
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "unselected",
        result: "finished",
        sample_enabled: true,
      },
    });

    page.unmount();
  });

  test("hides sample database setup when the target is unavailable", () => {
    mocks.sampleAvailable = false;
    const page = renderWorkspaceForm();

    expect(
      page.container.querySelector(
        "[data-testid='enable-sample-databases']"
      )
    ).toBeNull();

    page.unmount();
  });

  test("hides sample database setup when the workspace already provisioned a sample", () => {
    mocks.sampleInstances = [{ instance: "instances/sample" }];
    const page = renderWorkspaceForm();

    expect(
      page.container.querySelector(
        "[data-testid='enable-sample-databases']"
      )
    ).toBeNull();

    page.unmount();
  });

  test("skips workspace setup to the projects page with create project highlighted", async () => {
    const page = renderWorkspaceForm();

    const skip = Array.from(page.container.querySelectorAll("button")).find(
      (button) => button.textContent === "I'll do this later"
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
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup submitted",
      properties: {
        scenario: "unselected",
        result: "skipped",
        sample_enabled: false,
      },
    });

    page.unmount();
  });

  test.each([
    { isSaaSMode: false, deployment: "self-host" },
    { isSaaSMode: true, deployment: "cloud" },
  ])(
    "records a completed setup for the $deployment deployment",
    async ({ isSaaSMode }) => {
      mocks.isSaaSMode = isSaaSMode;
      const page = renderWorkspaceForm();

      const submit = Array.from(
        page.container.querySelectorAll("button")
      ).find((button) =>
        button.textContent?.includes("Setup my workspace")
      ) as HTMLButtonElement;
      await act(async () => {
        submit.click();
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(mocks.captureMetric).toHaveBeenCalledWith({
        event: "workspace setup submitted",
        properties: {
          scenario: "unselected",
          result: "finished",
          sample_enabled: true,
        },
      });

      page.unmount();
    }
  );
});
