import { Code, ConnectError } from "@connectrpc/connect";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { skipErrorNotificationContextKey } from "@/api/context-key";
import { SplashPanelContext } from "@/app/layouts/SplashLayout";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
  WORKSPACE_ROUTE_LANDING,
} from "@/app/router/handles";
import "@/lib/i18n";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import {
  readGuideTeamSize,
  readSelectedGuideScenarioId,
} from "@/modules/workspace-setup-guide/selection";
import { readWorkspaceSetupFinished } from "@/modules/workspace-setup-guide/setup";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { Workspace } from "@/types/proto-es/v1/workspace_service_pb";
import { storageKeyWorkspaceSetupDraft } from "@/utils/storage-keys";

const mocks = vi.hoisted(() => ({
  currentUser: {} as User,
  workspace: {} as Workspace,
  workspacePolicy: {} as IamPolicy,
  canUpdateWorkspace: true,
  isSaaSMode: true,
  sample: { available: true, instances: [] as Array<{ instance: string }> },
  introState: {} as Record<string, boolean>,
  saveIntroStateByKey: vi.fn(),
  updateUser: vi.fn(),
  updateWorkspace: vi.fn(),
  createProject: vi.fn(),
  setRecentProject: vi.fn(),
  fetchProject: vi.fn(),
  getProject: vi.fn(),
  fetchWorkspaceIamPolicy: vi.fn(),
  prepareSampleProjectInstance: vi.fn(),
  getProjectIamPolicy: vi.fn(),
  updateProjectIamPolicy: vi.fn(),
  patchWorkspaceIamPolicy: vi.fn(),
  routerReplace: vi.fn(),
  captureMetric: vi.fn(),
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: { captureMetric: mocks.captureMetric },
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
  useWorkspace: () => mocks.workspace,
  useWorkspacePermission: (permission: string) =>
    permission === "bb.workspaces.update" && mocks.canUpdateWorkspace,
  useIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
}));

// The two reads Finish makes of the server itself, past the store's cache.
vi.mock("@/api", () => ({
  projectServiceClientConnect: {
    getProject: mocks.getProject,
    getIamPolicy: mocks.getProjectIamPolicy,
  },
}));

vi.mock("@/stores/app", async () => {
  const { useSyncExternalStore } = await import("react");
  const state = () => ({
    currentUser: mocks.currentUser,
    workspacePolicy: mocks.workspacePolicy,
    serverInfo: { sample: mocks.sample },
    isSaaSMode: () => mocks.isSaaSMode,
    updateUser: mocks.updateUser,
    updateWorkspace: mocks.updateWorkspace,
    createProject: mocks.createProject,
    setRecentProject: mocks.setRecentProject,
    fetchProject: mocks.fetchProject,
    prepareSampleProjectInstance: mocks.prepareSampleProjectInstance,
    updateProjectIamPolicy: mocks.updateProjectIamPolicy,
    fetchWorkspaceIamPolicy: mocks.fetchWorkspaceIamPolicy,
    patchWorkspaceIamPolicy: mocks.patchWorkspaceIamPolicy,
    // As the real store: the roles `member` holds in the workspace policy.
    findWorkspaceRolesByMember: (member: string) =>
      mocks.workspacePolicy.bindings
        .filter((binding) => binding.members.includes(member))
        .map((binding) => binding.role),
    saveIntroStateByKey: mocks.saveIntroStateByKey,
  });
  // Read as the real store's hook reads, so a selector that returns a new
  // value on every call fails here as it would crash the page.
  const useAppStore = Object.assign(
    (selector: (value: ReturnType<typeof state>) => unknown) =>
      useSyncExternalStore(
        () => () => {},
        () => selector(state())
      ),
    { getState: state }
  );
  return { useAppStore };
});

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: { replace: mocks.routerReplace },
}));

// The real project ID field, with its random suffix pinned: "Payments" gets
// the ID "payments-ab12".
vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  randomString: () => "ab12",
}));

import { WorkspaceSetupPage } from "./WorkspaceSetupPage";

// What the server holds. The stand-ins for its calls read and write this, so
// a project a create made is there for every later read, as on a real server.
const server = {
  projects: new Map<string, { name: string; title: string }>(),
  projectPolicy: {} as IamPolicy,
};
// Makes the next call of `write` take effect and then lose its answer.
const loseAnswerOf = (write: ReturnType<typeof vi.fn>) => {
  const run = write.getMockImplementation() as
    | ((...args: unknown[]) => unknown)
    | undefined;
  write.mockImplementationOnce(async (...args: unknown[]) => {
    await run?.(...args);
    throw unavailable();
  });
};
// Makes the next call of `write` lose its answer at once and take effect only
// when the returned function is called, as a request held up on its way.
const delayLostWrite = (write: ReturnType<typeof vi.fn>) => {
  const run = write.getMockImplementation() as (
    ...args: unknown[]
  ) => Promise<unknown>;
  let land = () => {};
  write.mockImplementationOnce(async (...args: unknown[]) => {
    land = () => void run(...args).catch(() => {});
    throw unavailable();
  });
  return () => land();
};
// Holds the next call of `call` until the returned function is called: the
// call then runs as usual, or fails with the error passed. Queue it while no
// one-time behavior of `call` waits, or the held call runs that behavior.
const holdNextCall = (call: ReturnType<typeof vi.fn>) => {
  const run = call.getMockImplementation() as
    | ((...args: unknown[]) => unknown)
    | undefined;
  let release: (error?: unknown) => void = () => {};
  const held = new Promise<unknown>((resolve) => {
    release = resolve;
  });
  call.mockImplementationOnce(async (...args: unknown[]) => {
    const error = await held;
    if (error) throw error;
    return run?.(...args);
  });
  return release;
};
// Lets every answer already given reach the page.
const settle = () =>
  act(() => new Promise((resolve) => setTimeout(resolve, 0)));

const CHANGE_GOAL = "Plan, review and roll out a database change";
const QUERY_GOAL = "Query data with guardrails";
const EXPLORING = "Not sure yet, just exploring";
const SAMPLE = "Start with the sample database";
const OWN_INSTANCE = "Connect my own instance after setup";
const ID_EXISTS = "The Project ID already exists. Please choose another one.";
const REPORT = { byCaller: true };

const continueButton = () => screen.getByRole("button", { name: "Continue" });
const backButton = () => screen.getByRole("button", { name: "Back" });
const finishButton = () => screen.getByRole("button", { name: /^Finish$/ });
const type = (field: HTMLElement, value: string) =>
  fireEvent.change(field, { target: { value } });

const choice = (title: string) => {
  const label = screen.getByText(title).closest("label");
  if (!label) throw new Error(`no choice titled ${title}`);
  return within(label).getByRole("radio");
};

// Finish's writes to the server, by name, in the order they were made.
const WRITES = {
  updateUser: mocks.updateUser,
  updateWorkspace: mocks.updateWorkspace,
  createProject: mocks.createProject,
  prepareSample: mocks.prepareSampleProjectInstance,
  projectMembers: mocks.updateProjectIamPolicy,
  workspaceMembers: mocks.patchWorkspaceIamPolicy,
};
const writes = () =>
  Object.entries(WRITES)
    .flatMap(([name, write]) =>
      write.mock.invocationCallOrder.map((at) => ({ name, at }))
    )
    .sort((a, b) => a.at - b.at)
    .map(({ name }) => name);

// The project policy Finish last wrote, binding by binding: a role listed
// twice would mean an earlier attempt's members were left in place.
const projectBindings = () => {
  const [, policy] = mocks.updateProjectIamPolicy.mock.calls.at(-1) as [
    string,
    IamPolicy,
  ];
  return policy.bindings.map((binding) => [binding.role, binding.members]);
};
const OWNER_BINDING = ["roles/projectOwner", ["user:ed@example.com"]];
const DRAFT_KEY = storageKeyWorkspaceSetupDraft(
  "workspaces/ws1",
  "ed@example.com"
);
// What the browser tells this tab when another tab saves the draft, or
// removes it.
const anotherTab = (newValue: string | null) =>
  act(() => {
    window.dispatchEvent(
      new StorageEvent("storage", { key: DRAFT_KEY, newValue })
    );
  });
const anotherTabSaved = () => anotherTab(localStorage.getItem(DRAFT_KEY));
// A role given on the project from somewhere other than the wizard.
const grantElsewhere = (role: string, member: string) => {
  server.projectPolicy = {
    ...server.projectPolicy,
    bindings: server.projectPolicy.bindings.map((binding) =>
      binding.role === role
        ? { ...binding, members: [...binding.members, member] }
        : binding
    ),
  } as IamPolicy;
};
// The project policy as the server holds it, whoever wrote it last.
const serverBindings = () =>
  server.projectPolicy.bindings.map((binding) => [
    binding.role,
    binding.members,
  ]);

const eventsNamed = (event: string) =>
  mocks.captureMetric.mock.calls
    .map(([metric]) => metric as { event: string; properties: unknown })
    .filter((metric) => metric.event === event)
    .map((metric) => metric.properties);

const nameProject = async (name = "Payments") => {
  type(screen.getByLabelText(/^Project name/), name);
  await waitFor(() => expect(continueButton()).toBeEnabled());
};
const editProjectId = (id: string) => {
  fireEvent.click(screen.getByRole("button", { name: "Edit" }));
  type(screen.getByPlaceholderText("Project ID"), id);
};

const goToGoal = () => fireEvent.click(continueButton());
const goToProject = (goal = QUERY_GOAL) => {
  goToGoal();
  fireEvent.click(screen.getByText(goal));
  fireEvent.click(continueButton());
};
const goToTeam = async ({
  goal,
  database,
}: {
  goal?: string;
  database?: string;
} = {}) => {
  goToProject(goal);
  await nameProject();
  if (database) fireEvent.click(screen.getByText(database));
  fireEvent.click(continueButton());
};
const startAtProject = (goal?: string) => {
  const view = render(<WorkspaceSetupPage />);
  goToProject(goal);
  return view;
};
const startAtTeam = async (options?: { goal?: string; database?: string }) => {
  const view = render(<WorkspaceSetupPage />);
  await goToTeam(options);
  return view;
};

const invitees = (field = "Invite teammates") =>
  screen.getByRole("textbox", { name: new RegExp(`^${field}`) });
const enterAddress = (address: string, field?: string) => {
  type(invitees(field), address);
  fireEvent.keyDown(invitees(field), { key: "Enter" });
};
const finishAlone = () => {
  fireEvent.click(screen.getByText("Just me"));
  fireEvent.click(finishButton());
};
const finishInviting = (...addresses: string[]) => {
  fireEvent.click(screen.getByText("2–10"));
  for (const address of addresses) enterAddress(address);
  fireEvent.click(screen.getByRole("button", { name: /and finish$/ }));
};
const storeDraft = (draft: Record<string, unknown>) =>
  localStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
const landed = () =>
  waitFor(() => expect(mocks.routerReplace).toHaveBeenCalled());
const failed = () => screen.findByRole("alert");
const tryAgain = () =>
  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
const finishAgain = () =>
  fireEvent.click(screen.getByRole("button", { name: /^Finish$|and finish$/ }));
function unavailable(message = "offline") {
  return new ConnectError(message, Code.Unavailable);
}
function alreadyExists() {
  return new ConnectError("already exists", Code.AlreadyExists);
}

beforeEach(() => {
  // Reset, not clear: a failure queued with `...Once` by a test that never
  // reached it must not fail the next test's first call.
  vi.resetAllMocks();
  localStorage.clear();
  mocks.currentUser = {
    name: "users/ed@example.com",
    email: "ed@example.com",
    title: "Ed",
    workspace: "workspaces/ws1",
  } as User;
  mocks.workspace = {
    name: "workspaces/ws1",
    title: "Workspace One",
  } as Workspace;
  mocks.workspacePolicy = {
    bindings: [
      { role: "roles/workspaceAdmin", members: ["user:ed@example.com"] },
    ],
  } as IamPolicy;
  mocks.canUpdateWorkspace = true;
  mocks.isSaaSMode = true;
  mocks.sample = { available: true, instances: [] };
  mocks.introState = {};
  mocks.saveIntroStateByKey.mockImplementation(({ key, newState }) => {
    mocks.introState[key] = newState;
  });
  mocks.updateUser.mockResolvedValue({});
  mocks.updateWorkspace.mockResolvedValue({});
  server.projects.clear();
  server.projectPolicy = {
    etag: "1",
    bindings: [
      { role: "roles/projectOwner", members: ["user:ed@example.com"] },
    ],
  } as IamPolicy;
  mocks.createProject.mockImplementation(async (title: string, id: string) => {
    const name = `projects/${id}`;
    if (server.projects.has(name)) {
      throw alreadyExists();
    }
    server.projects.set(name, { name, title });
    return { name, title };
  });
  // As the real store: undefined for a project that is not there.
  mocks.fetchProject.mockImplementation(async (name: string) =>
    server.projects.get(name)
  );
  mocks.getProject.mockImplementation(async ({ name }: { name: string }) => {
    const project = server.projects.get(name);
    if (!project) throw new ConnectError("not found", Code.NotFound);
    return project;
  });
  mocks.prepareSampleProjectInstance.mockImplementation(async (project: string) => {
    const name = `${project}/instances/sample`;
    mocks.sample.instances = [{ instance: name }];
    return { name };
  });
  mocks.getProjectIamPolicy.mockImplementation(
    async () => server.projectPolicy
  );
  // As the server: a write built on a policy it no longer holds is refused.
  mocks.updateProjectIamPolicy.mockImplementation(
    async (_project: string, policy: IamPolicy) => {
      if (policy.etag !== server.projectPolicy.etag) {
        throw new ConnectError("concurrent update", Code.Aborted);
      }
      server.projectPolicy = {
        ...policy,
        etag: String(Number(policy.etag) + 1),
      };
      return server.projectPolicy;
    }
  );
  mocks.fetchWorkspaceIamPolicy.mockImplementation(
    async () => mocks.workspacePolicy
  );
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("WorkspaceSetupPage", () => {
  describe("screen 1, workspace", () => {
    test("arrives with both names filled in, Continue ready, and the keyboard in the first field", () => {
      render(<WorkspaceSetupPage />);

      expect(
        screen.getByRole("heading", { name: "Set up your workspace" })
      ).toBeInTheDocument();
      expect(screen.getByLabelText(/^Your name/)).toHaveValue("Ed");
      expect(screen.getByLabelText(/^Your name/)).toHaveFocus();
      expect(screen.getByLabelText("Workspace name")).toHaveValue(
        "Workspace One"
      );
      expect(continueButton()).toBeEnabled();
    });

    test.each([
      {
        from: "a work address",
        email: "peter.zhu@acme.io",
        title: "peter.zhu@acme.io",
        name: "Peter Zhu",
        workspace: "Acme",
      },
      {
        from: "a tagged address at a hyphenated company",
        email: "peter_zhu+bytebase@blue-sky.io",
        title: "",
        name: "Peter Zhu",
        workspace: "Blue Sky",
      },
      {
        from: "a mail subdomain of the company",
        email: "peter@mail.acme.io",
        title: "",
        name: "Peter",
        workspace: "Acme",
      },
      {
        from: "a shared mailbox, which names nobody",
        email: "admin@acme.co.uk",
        title: "admin",
        name: "",
        workspace: "Acme",
      },
      {
        from: "a mailbox named for a team",
        email: "it-support@acme.com",
        title: "",
        name: "",
        workspace: "Acme",
      },
      {
        from: "a mailbox that takes no replies",
        email: "no-reply@acme.com",
        title: "",
        name: "",
        workspace: "Acme",
      },
      {
        from: "a public mail domain, which names no company",
        email: "peter@gmail.com",
        title: "",
        name: "Peter",
        workspace: "Peter's workspace",
      },
      {
        from: "a public mail provider's country domain",
        email: "taro@yahoo.co.jp",
        title: "",
        name: "Taro",
        workspace: "Taro's workspace",
      },
      {
        from: "a public mail provider's subdomain",
        email: "wang@vip.qq.com",
        title: "",
        name: "Wang",
        workspace: "Wang's workspace",
      },
      {
        from: "a number on a public mail domain, which names nobody",
        email: "13800138000@139.com",
        title: "",
        name: "",
        workspace: "My workspace",
      },
      {
        from: "a shared mailbox on a public mail domain",
        email: "info@outlook.com",
        title: "",
        name: "",
        workspace: "My workspace",
      },
    ])("guesses the names of a new account from $from", (account) => {
      mocks.currentUser = {
        ...mocks.currentUser,
        name: `users/${account.email}`,
        email: account.email,
        title: account.title,
      } as User;
      mocks.workspace = {
        ...mocks.workspace,
        title: "Default workspace",
      } as Workspace;
      mocks.workspacePolicy = {
        bindings: [
          { role: "roles/workspaceAdmin", members: [`user:${account.email}`] },
        ],
      } as IamPolicy;

      render(<WorkspaceSetupPage />);

      expect(screen.getByLabelText(/^Your name/)).toHaveValue(account.name);
      expect(screen.getByLabelText("Workspace name")).toHaveValue(
        account.workspace
      );
    });

    test("takes the title an identity-provider signup gets for an unnamed workspace too", () => {
      mocks.currentUser = {
        ...mocks.currentUser,
        name: "users/peter.zhu@acme.io",
        email: "peter.zhu@acme.io",
        title: "Peter Zhu",
      } as User;
      mocks.workspace = {
        ...mocks.workspace,
        title: "Default Workspace",
      } as Workspace;

      render(<WorkspaceSetupPage />);

      expect(screen.getByLabelText("Workspace name")).toHaveValue("Acme");
    });

    test.each([
      "",
      "  ",
    ])("explains under the field why a workspace name of %j blocks Continue", (emptied) => {
      render(<WorkspaceSetupPage />);
      const workspaceName = screen.getByLabelText("Workspace name");

      type(workspaceName, emptied);

      expect(workspaceName).toHaveValue(emptied);
      expect(continueButton()).toBeDisabled();
      expect(workspaceName).toHaveAccessibleDescription(
        "Enter a workspace name"
      );
    });

    test("lets the name be emptied without the guess coming back", () => {
      render(<WorkspaceSetupPage />);

      type(screen.getByLabelText(/^Your name/), "");

      expect(screen.getByLabelText(/^Your name/)).toHaveValue("");
      expect(continueButton()).toBeEnabled();
    });

    test("shows nothing until the account is known, and records no entry before the workspace is", () => {
      mocks.currentUser = { ...mocks.currentUser, email: "" } as User;
      const view = render(<WorkspaceSetupPage />);
      expect(view.container).toBeEmptyDOMElement();

      mocks.currentUser = {
        ...mocks.currentUser,
        email: "ed@example.com",
      } as User;
      mocks.workspace = { name: "", title: "" } as Workspace;
      view.rerender(<WorkspaceSetupPage />);

      expect(screen.getByLabelText(/^Your name/)).toHaveValue("Ed");
      expect(eventsNamed("workspace setup page entered")).toEqual([]);
    });
  });

  describe("every screen", () => {
    const steps = () =>
      within(screen.getByRole("list"))
        .getAllByRole("listitem")
        .map((step) => step.textContent);

    test("shows the same four steps, with the ones behind it checked off", () => {
      startAtProject();

      expect(steps()).toEqual(["Workspace", "Goal", "3Project", "4Team"]);
    });

    test("moves the keyboard to the new screen's first answer", async () => {
      render(<WorkspaceSetupPage />);
      goToGoal();
      expect(choice(CHANGE_GOAL)).toHaveFocus();

      fireEvent.click(screen.getByText(QUERY_GOAL));
      fireEvent.click(continueButton());
      expect(screen.getByLabelText(/^Project name/)).toHaveFocus();

      await nameProject();
      fireEvent.click(continueButton());
      expect(choice("Just me")).toHaveFocus();

      fireEvent.click(screen.getByText("11–50"));
      fireEvent.click(backButton());
      fireEvent.click(backButton());
      expect(choice(QUERY_GOAL)).toHaveFocus();
    });

    test("continues on Enter without the browser submitting the form", () => {
      render(<WorkspaceSetupPage />);
      const form = screen.getByLabelText(/^Your name/).closest("form");

      // False: the submit's default action was prevented.
      expect(fireEvent.submit(form as HTMLFormElement)).toBe(false);

      expect(
        screen.getByRole("heading", { name: "What do you want to do first?" })
      ).toBeInTheDocument();
    });

    test("keeps the browser from offering saved values in any of its fields", async () => {
      render(<WorkspaceSetupPage />);
      const offersNothing = () => {
        const fields = screen.getAllByRole("textbox");
        for (const field of fields) {
          expect(field).toHaveAttribute("autocomplete", "off");
        }
        return fields.length;
      };
      expect(offersNothing()).toBe(2);

      goToProject();
      await nameProject();
      editProjectId("pay");
      expect(offersNothing()).toBe(2);

      fireEvent.click(continueButton());
      fireEvent.click(screen.getByText("2–10"));
      expect(offersNothing()).toBe(1);
    });
  });

  describe.each([
    {
      account: "an account that joined someone else's workspace",
      arrange: () => {
        mocks.workspacePolicy = {
          bindings: [
            {
              role: "roles/workspaceAdmin",
              members: ["user:owner@example.com", "user:ed@example.com"],
            },
          ],
        } as IamPolicy;
      },
    },
    {
      account: "an account that may not update the workspace",
      arrange: () => {
        mocks.canUpdateWorkspace = false;
      },
    },
  ])("$account", ({ arrange }) => {
    test("is not asked to name it, and Finish leaves its name alone", async () => {
      arrange();
      render(<WorkspaceSetupPage />);

      expect(screen.queryByLabelText("Workspace name")).not.toBeInTheDocument();
      expect(continueButton()).toBeEnabled();

      await goToTeam();
      finishAlone();

      await landed();
      expect(writes()).toEqual(["updateUser", "createProject", "prepareSample"]);
    });
  });

  describe("screen 2, goal", () => {
    const goalChoices = () =>
      screen.getByRole("radiogroup", { name: "What do you want to do first?" });

    test("blocks Continue until a goal is picked, with the reason under the choices", () => {
      render(<WorkspaceSetupPage />);
      goToGoal();

      expect(within(goalChoices()).getAllByRole("radio")).toHaveLength(3);
      expect(continueButton()).toBeDisabled();
      expect(goalChoices()).toHaveAccessibleDescription("Pick one to continue");

      fireEvent.click(screen.getByText(QUERY_GOAL));

      expect(continueButton()).toBeEnabled();
      expect(goalChoices()).not.toHaveAccessibleDescription();
    });
  });

  describe("screen 3, project", () => {
    test("asks for a project name before Continue, with the sample database preselected", async () => {
      startAtProject();
      const projectName = screen.getByLabelText(/^Project name/);

      expect(projectName).toHaveValue("");
      expect(continueButton()).toBeDisabled();
      expect(projectName).toHaveAccessibleDescription(
        expect.stringContaining("Enter a project name")
      );
      expect(choice(SAMPLE)).toBeChecked();
      expect(choice(OWN_INSTANCE)).not.toBeChecked();

      await nameProject();

      expect(screen.getByText("payments-ab12")).toBeInTheDocument();
      expect(projectName).not.toHaveAccessibleDescription(
        expect.stringContaining("Enter a project name")
      );
      expect(
        screen.getByRole("radiogroup", { name: "Database for Payments" })
      ).toBeInTheDocument();
    });

    test("lets self-hosted leave the project for later, offering no database and making no project", async () => {
      mocks.isSaaSMode = false;
      startAtProject(CHANGE_GOAL);
      const projectName = screen.getByLabelText(/^Project name/);

      expect(projectName).toHaveAccessibleName(
        expect.stringContaining("optional")
      );
      expect(projectName).toHaveAccessibleDescription(
        expect.stringContaining("Leave it empty to create a project later.")
      );
      expect(screen.queryByText(SAMPLE)).not.toBeInTheDocument();
      expect(screen.queryByText(OWN_INSTANCE)).not.toBeInTheDocument();

      fireEvent.click(continueButton());
      fireEvent.click(screen.getByText("2–10"));
      fireEvent.click(finishButton());

      await landed();
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: WORKSPACE_ROUTE_LANDING,
      });
      expect(writes()).toEqual(["updateUser", "updateWorkspace"]);
      expect(readSelectedGuideScenarioId()).toBe("create-database-change");
      expect(eventsNamed("workspace setup submitted")).toEqual([
        expect.objectContaining({
          result: "finished",
          source: "none",
          sample_enabled: false,
        }),
      ]);
    });

    test("offers the database again once a self-hosted project is named", async () => {
      mocks.isSaaSMode = false;
      startAtProject();

      await nameProject();

      expect(choice(SAMPLE)).toBeChecked();
      expect(choice(OWN_INSTANCE)).not.toBeChecked();
    });

    test("needs the project name on self-hosted once a create has been sent", () => {
      mocks.isSaaSMode = false;
      storeDraft({
        screen: "project",
        goal: "query-data",
        projectTitle: "",
        projectId: "",
        attemptedProjectId: "payments-ab12",
      });
      render(<WorkspaceSetupPage />);

      expect(screen.getByLabelText(/^Project name/)).toHaveAccessibleDescription(
        expect.stringContaining("Enter a project name")
      );
      expect(continueButton()).toBeDisabled();
    });

    test("blocks Continue on an emptied project ID, with the field's reason", async () => {
      startAtProject();
      await nameProject();

      editProjectId("");

      expect(
        await screen.findByText("The Project ID cannot be empty")
      ).toBeInTheDocument();
      expect(continueButton()).toBeDisabled();
    });

    test("blocks Continue on a project ID another project holds, with the field's reason", async () => {
      server.projects.set("projects/billing", {
        name: "projects/billing",
        title: "Someone else's project",
      });
      startAtProject();
      await nameProject();

      editProjectId("billing");

      expect(await screen.findByText(ID_EXISTS)).toBeInTheDocument();
      expect(continueButton()).toBeDisabled();
    });

    test("does not hold Continue back while the ID check is still out", async () => {
      mocks.fetchProject.mockReturnValue(new Promise(() => {}));
      startAtProject();

      type(screen.getByLabelText(/^Project name/), "Payments");

      await waitFor(() => expect(continueButton()).toBeEnabled());
      expect(mocks.fetchProject).toHaveBeenCalledWith(
        "projects/payments-ab12",
        true
      );
    });

    test("derives a new ID for a name entered after the first was cleared", async () => {
      startAtProject();
      await nameProject("Payments");

      type(screen.getByLabelText(/^Project name/), "");
      expect(continueButton()).toBeDisabled();
      expect(screen.getByText("Enter a project name")).toBeInTheDocument();

      await nameProject("Orders");

      expect(screen.getByText("orders-ab12")).toBeInTheDocument();
      expect(screen.queryByText("payments-ab12")).not.toBeInTheDocument();
    });

    test.each([
      {
        when: "the server offers no sample",
        available: false,
        instances: [],
        reason: "The sample database is not available on this server.",
      },
      {
        when: "the workspace already used its sample",
        available: true,
        instances: [{ instance: "projects/earlier/instances/sample" }],
        reason: "This workspace has already used its sample database.",
      },
    ])("offers only the own-instance choice, with the reason, when $when", async (sample) => {
      mocks.sample = sample;
      startAtProject();

      expect(screen.queryByText(SAMPLE)).not.toBeInTheDocument();
      expect(choice(OWN_INSTANCE)).toBeChecked();
      expect(screen.getByText(sample.reason)).toBeInTheDocument();

      await nameProject();
      fireEvent.click(continueButton());
      finishAlone();

      await landed();
      expect(writes()).toEqual(["updateUser", "updateWorkspace", "createProject"]);
      expect(eventsNamed("workspace setup submitted")).toEqual([
        expect.objectContaining({ source: "own", sample_enabled: false }),
      ]);
    });

    test.each([
      {
        deployment: "Bytebase Cloud, for seven days",
        isSaaSMode: true,
        says: /hosted by Bytebase Cloud.*available for 7 days/,
        omits: /on this server/,
      },
      {
        deployment: "this server, with no expiry",
        isSaaSMode: false,
        says: /running on this server/,
        omits: /7 days/,
      },
    ])("says the sample runs on $deployment", async ({
      isSaaSMode,
      says,
      omits,
    }) => {
      mocks.isSaaSMode = isSaaSMode;
      startAtProject();
      await nameProject();

      expect(screen.getByText(says)).toBeInTheDocument();
      expect(screen.queryByText(omits)).not.toBeInTheDocument();
    });
  });

  describe("screen 3, after a Finish attempt created the project", () => {
    // One invitee, so every step of the commit has work to do.
    const failFinish = async () => {
      await startAtTeam();
      finishInviting("ann@example.com");
      await failed();
    };

    test("shows the project and its prepared sample read-only", async () => {
      mocks.updateProjectIamPolicy.mockRejectedValueOnce(unavailable());
      await failFinish();
      // Whatever the server offers now, the prepared sample is the project's.
      mocks.sample = { available: false, instances: [] };
      fireEvent.click(backButton());

      expect(screen.getByLabelText(/^Project name/)).toBeDisabled();
      expect(screen.getByLabelText(/^Project name/)).toHaveValue("Payments");
      expect(screen.getByText("payments-ab12")).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Edit" })
      ).not.toBeInTheDocument();
      expect(
        screen.getByText(
          "This project already exists. Rename it in its settings."
        )
      ).toBeInTheDocument();
      expect(choice(SAMPLE)).toBeChecked();
      expect(choice(SAMPLE)).toHaveAttribute("aria-disabled", "true");
      expect(
        screen.getByText("The sample database is ready in this project.")
      ).toBeInTheDocument();
      expect(continueButton()).toBeEnabled();
    });

    test("still lets the database choice change while the sample is not prepared", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await failFinish();
      fireEvent.click(backButton());

      expect(screen.getByLabelText(/^Project name/)).toBeDisabled();
      fireEvent.click(screen.getByText(OWN_INSTANCE));

      expect(choice(OWN_INSTANCE)).toBeChecked();
      expect(continueButton()).toBeEnabled();
    });
  });

  describe("I'll do this later", () => {
    const skipButton = () =>
      screen.getByRole("button", { name: "I'll do this later" });
    const skipOffered = () =>
      screen.queryByRole("button", { name: "I'll do this later" }) !== null;

    test("is not offered on Cloud, where setup is mandatory", () => {
      startAtProject();

      expect(skipOffered()).toBe(false);
    });

    test("saves the two names on self-hosted and leaves without creating a project", async () => {
      mocks.isSaaSMode = false;
      const view = startAtProject(CHANGE_GOAL);

      fireEvent.click(skipButton());

      await landed();
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: WORKSPACE_ROUTE_LANDING,
      });
      expect(writes()).toEqual(["updateUser", "updateWorkspace"]);
      expect(readSelectedGuideScenarioId()).toBe("create-database-change");
      expect(eventsNamed("workspace setup submitted")).toEqual([
        {
          setup_version: "v2",
          result: "skipped",
          scenario: "create-database-change",
          sample_enabled: false,
        },
      ]);

      view.unmount();
      render(<WorkspaceSetupPage />);
      expect(
        screen.getByRole("heading", { name: "Set up your workspace" })
      ).toBeInTheDocument();
    });

    test("stays on the screen with the error when the names cannot be saved, and can be pressed again", async () => {
      mocks.isSaaSMode = false;
      mocks.updateUser.mockRejectedValueOnce(unavailable("database is down"));
      startAtProject();

      fireEvent.click(skipButton());

      const alert = await failed();
      expect(alert).toHaveTextContent(
        "Could not save your name and the workspace name."
      );
      expect(alert).toHaveTextContent("database is down");
      expect(screen.getByLabelText(/^Project name/)).toBeInTheDocument();
      expect(mocks.routerReplace).not.toHaveBeenCalled();

      fireEvent.click(skipButton());

      await landed();
    });

    test("drops its failure's report once the screen is left", async () => {
      mocks.isSaaSMode = false;
      mocks.updateUser.mockRejectedValueOnce(unavailable());
      startAtProject();
      fireEvent.click(skipButton());
      await failed();

      fireEvent.click(backButton());
      fireEvent.click(continueButton());

      expect(screen.getByLabelText(/^Project name/)).toBeInTheDocument();
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    test("is withheld while a create got no answer and the server cannot be read", async () => {
      mocks.isSaaSMode = false;
      mocks.createProject.mockRejectedValueOnce(unavailable());
      mocks.getProject.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishAlone();
      await failed();

      fireEvent.click(backButton());

      await waitFor(() => expect(mocks.getProject).toHaveBeenCalledTimes(1));
      expect(skipOffered()).toBe(false);
      expect(screen.getByLabelText(/^Project name/)).toBeEnabled();
    });

    test("is withdrawn once a Finish attempt has created the project", async () => {
      mocks.isSaaSMode = false;
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishAlone();
      await failed();

      fireEvent.click(backButton());

      expect(screen.getByLabelText(/^Project name/)).toBeDisabled();
      expect(skipOffered()).toBe(false);
    });
  });

  describe("screen 4, team", () => {
    const teamSizes = () =>
      screen.getByRole("radiogroup", {
        name: "How many people will use Bytebase?",
      });

    test("asks only for a team size, and blocks Finish until one is picked", async () => {
      await startAtTeam();

      expect(
        within(teamSizes())
          .getAllByRole("radio")
          .map((radio) => radio.closest("label")?.textContent)
      ).toEqual(["Just me", "2–10", "11–50", "51–200", "200+"]);
      expect(finishButton()).toBeDisabled();
      expect(teamSizes()).toHaveAccessibleDescription(
        "Pick a team size to finish"
      );
      expect(backButton()).toBeEnabled();
      expect(screen.queryByText("Invite teammates")).not.toBeInTheDocument();

      fireEvent.click(screen.getByText("Just me"));

      expect(finishButton()).toBeEnabled();
      expect(teamSizes()).not.toHaveAccessibleDescription();
      expect(screen.queryByText("Invite teammates")).not.toBeInTheDocument();
    });

    test("invites teammates on Cloud, and the button counts them", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      expect(invitees()).toHaveAccessibleDescription(
        /^Press Enter after each address\./
      );

      expect(
        screen.getByText(
          /They join as members who can query and propose changes in Payments\./
        )
      ).toBeInTheDocument();
      expect(finishButton()).toBeEnabled();

      enterAddress("ann@example.com");
      expect(
        screen.getByRole("button", { name: "Send 1 invite and finish" })
      ).toBeEnabled();

      enterAddress("bob@example.com");
      expect(
        screen.getByRole("button", { name: "Send 2 invites and finish" })
      ).toBeEnabled();
    });

    test.each([
      {
        where: "in place of the text selected in the field",
        typed: "bob@acme.cmo",
        selection: [0, 12],
        entries: ["ann@acme.com", "cy@acme.com"],
      },
      {
        where: "before the text typed after the cursor, which stays an entry",
        typed: "bob@acme.com",
        selection: [0, 0],
        entries: ["ann@acme.com", "cy@acme.com", "bob@acme.com"],
      },
    ])("puts a pasted list $where", async ({ typed, selection, entries }) => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      type(invitees(), typed);
      (invitees() as HTMLInputElement).setSelectionRange(
        selection[0],
        selection[1]
      );

      fireEvent.paste(invitees(), {
        clipboardData: { getData: () => "ann@acme.com, cy@acme.com" },
      });

      expect(
        screen
          .getAllByRole("button", { name: /^Remove / })
          .map((remove) => remove.getAttribute("aria-label"))
      ).toEqual(entries.map((entry) => `Remove ${entry}`));
    });

    test("takes the addresses out of a pasted list, and shows but does not count repeats, bad entries and the owner", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("11–50"));
      type(invitees(), "cy@example.com");

      // False: the paste's default action, inserting the text, was prevented.
      expect(
        fireEvent.paste(invitees(), {
          clipboardData: {
            getData: () =>
              'Ann Lee <ann@example.com>, "bob@example.com";ANN@example.com\nmailto:dee@example.com. (fay@example.com) a@b@c.com gil@localhost Ed@Example.com',
          },
        })
      ).toBe(false);

      expect(invitees()).toHaveValue("");
      expect(
        screen
          .getAllByRole("button", { name: /^Remove / })
          .map((remove) => remove.getAttribute("aria-label"))
      ).toEqual([
        "Remove cy@example.com",
        "Remove ann@example.com",
        "Remove bob@example.com",
        "Remove dee@example.com",
        "Remove fay@example.com",
        "Remove a@b@c.com",
        "Remove gil@localhost",
        "Remove Ed@Example.com",
      ]);
      for (const bad of ["a@b@c.com", "gil@localhost"]) {
        expect(screen.getByText(bad).parentElement).toHaveTextContent(
          "Not an email address"
        );
      }
      expect(
        screen.getByText("Ed@Example.com").parentElement
      ).toHaveTextContent("Your own address");
      expect(
        screen.getByRole("button", { name: "Send 5 invites and finish" })
      ).toBeEnabled();

      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );

      expect(screen.queryByText("bob@example.com")).not.toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Send 4 invites and finish" })
      ).toBeEnabled();
    });

    test.each([
      { typed: "ann@example", entered: "not an address", finishes: false },
      { typed: "ann@example.com", entered: "an address", finishes: true },
    ])("enters text left in the field when the button is pressed, and stops there when it is $entered", async ({
      typed,
      finishes,
    }) => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      type(invitees(), typed);
      const button = screen.getByRole("button", { name: /finish$/i });

      // A press moves the focus out of the field before its click.
      fireEvent.pointerDown(button);
      fireEvent.blur(invitees());
      fireEvent.click(button);

      expect(screen.getByText(typed)).toBeInTheDocument();
      if (finishes) {
        await landed();
        expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalledWith(
          [{ member: "user:ann@example.com", roles: ["roles/workspaceMember"] }],
          REPORT
        );
        return;
      }
      expect(screen.getByText(typed).parentElement).toHaveTextContent(
        "Not an email address"
      );
      expect(writes()).toEqual([]);

      fireEvent.click(finishButton());

      await landed();
      expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
    });

    test("stops a press on Finish without the sample that enters text that is not an address", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      fireEvent.click(finishButton());
      await failed();
      type(invitees(), "ann@example");
      const button = screen.getByRole("button", {
        name: "Finish without the sample",
      });
      const writesBefore = writes();

      fireEvent.pointerDown(button);
      fireEvent.blur(invitees());
      fireEvent.click(button);

      expect(screen.getByText("ann@example").parentElement).toHaveTextContent(
        "Not an email address"
      );
      // No attempt started: it would have saved the names first.
      expect(writes()).toEqual(writesBefore);

      fireEvent.click(button);

      await landed();
      expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
    });

    test("does not stop a press that enters the account's own address", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      type(invitees(), "ed@example.com");
      const button = finishButton();

      fireEvent.pointerDown(button);
      fireEvent.blur(invitees());
      fireEvent.click(button);

      await landed();
      expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
    });

    test.each([
      { separator: "，", how: "a full-width comma" },
      { separator: "；", how: "a full-width semicolon" },
      { separator: "、", how: "an ideographic comma" },
    ])("splits addresses at $how, as an input method types it", async ({
      separator,
    }) => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));

      type(invitees(), `ann@example.com${separator}`);
      fireEvent.paste(invitees(), {
        clipboardData: {
          getData: () => `bob@example.com${separator}cy@example.com`,
        },
      });

      expect(
        screen.getByRole("button", { name: "Send 3 invites and finish" })
      ).toBeEnabled();
    });

    test("leaves the Enter that confirms an input method's text to the input method", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      type(invitees(), "tanaka");

      fireEvent.keyDown(invitees(), { key: "Enter", isComposing: true });

      expect(invitees()).toHaveValue("tanaka");
      expect(
        screen.queryByRole("button", { name: /^Remove / })
      ).not.toBeInTheDocument();
    });

    test("leaves the Backspace that edits an input method's text to the input method", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      enterAddress("ann@example.com");

      fireEvent.keyDown(invitees(), { key: "Backspace", isComposing: true });

      expect(
        screen.getByRole("button", { name: "Remove ann@example.com" })
      ).toBeInTheDocument();
    });

    test("leaves a single pasted address to the browser, as typed text", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));

      expect(
        fireEvent.paste(invitees(), {
          clipboardData: { getData: () => "ann@example.com" },
        })
      ).toBe(true);
    });

    test("adds an address on a separator, on Enter and on leaving the field, and removes the last on Backspace", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      const chips = () =>
        screen
          .getAllByRole("button", { name: /^Remove / })
          .map((remove) => remove.getAttribute("aria-label"));

      type(invitees(), "ann@example.com,");
      type(invitees(), "bob@example.com");
      // False: Enter adds the address; it must not submit the wizard.
      expect(fireEvent.keyDown(invitees(), { key: "Enter" })).toBe(false);
      type(invitees(), "cy@example.com");
      fireEvent.blur(invitees());

      expect(chips()).toEqual([
        "Remove ann@example.com",
        "Remove bob@example.com",
        "Remove cy@example.com",
      ]);
      expect(writes()).toEqual([]);

      fireEvent.keyDown(invitees(), { key: "Backspace" });

      expect(chips()).toEqual([
        "Remove ann@example.com",
        "Remove bob@example.com",
      ]);
    });

    test("leaves the invitees out, on screen and at Finish, when the answer goes back to Just me", async () => {
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));
      enterAddress("ann@example.com");

      fireEvent.click(screen.getByText("Just me"));

      expect(screen.queryByText("ann@example.com")).not.toBeInTheDocument();
      fireEvent.click(finishButton());
      await landed();
      expect(writes()).toEqual([
        "updateUser",
        "updateWorkspace",
        "createProject",
        "prepareSample",
      ]);
    });

    test("asks only the team size on self-hosted, where a member added would have no password", async () => {
      mocks.isSaaSMode = false;
      await startAtTeam();
      fireEvent.click(screen.getByText("2–10"));

      expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
      fireEvent.click(finishButton());
      await landed();
      expect(mocks.updateProjectIamPolicy).not.toHaveBeenCalled();
      expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
    });

    test("adds no one on self-hosted from addresses a draft holds", async () => {
      mocks.isSaaSMode = false;
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "own",
        teamSize: "2-10",
        emails: ["ann@example.com"],
      });
      render(<WorkspaceSetupPage />);

      expect(screen.queryByText("ann@example.com")).not.toBeInTheDocument();
      fireEvent.click(finishButton());
      await landed();
      expect(mocks.updateProjectIamPolicy).not.toHaveBeenCalled();
      expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
      expect(eventsNamed("workspace setup submitted")).toEqual([
        expect.objectContaining({ result: "finished", invite_count: 0 }),
      ]);
    });

    test.each([
      { database: SAMPLE, arriveFirst: false },
      { database: OWN_INSTANCE, arriveFirst: true },
    ])("tells the account whether its invitees arrive before the databases, with $database", async ({
      database,
      arriveFirst,
    }) => {
      await startAtTeam({ database });
      fireEvent.click(screen.getByText("2–10"));

      expect(
        screen.queryAllByText(
          /Its databases appear once your instance is connected\./
        )
      ).toHaveLength(arriveFirst ? 1 : 0);
    });
  });

  describe("Back", () => {
    const answerEverything = async () => {
      type(screen.getByLabelText(/^Your name/), "Edward");
      type(screen.getByLabelText("Workspace name"), "Acme");
      goToProject(CHANGE_GOAL);
      await nameProject();
      editProjectId("pay");
      await waitFor(() => expect(continueButton()).toBeEnabled());
      fireEvent.click(screen.getByText(OWN_INSTANCE));
      fireEvent.click(continueButton());
      fireEvent.click(screen.getByText("2–10"));
      enterAddress("ann@example.com");
    };
    const expectEveryAnswer = async () => {
      expect(choice("2–10")).toBeChecked();
      expect(screen.getByText("ann@example.com")).toBeInTheDocument();

      fireEvent.click(backButton());
      expect(screen.getByLabelText(/^Project name/)).toHaveValue("Payments");
      expect(screen.getByText("pay")).toBeInTheDocument();
      expect(choice(OWN_INSTANCE)).toBeChecked();
      await waitFor(() => expect(continueButton()).toBeEnabled());

      fireEvent.click(backButton());
      expect(choice(CHANGE_GOAL)).toBeChecked();

      fireEvent.click(backButton());
      expect(screen.getByLabelText(/^Your name/)).toHaveValue("Edward");
      expect(screen.getByLabelText("Workspace name")).toHaveValue("Acme");
    };

    test("returns through every screen with each answer kept, a hand-set project ID included", async () => {
      render(<WorkspaceSetupPage />);
      await answerEverything();

      await expectEveryAnswer();

      fireEvent.click(continueButton());
      fireEvent.click(continueButton());
      fireEvent.click(continueButton());
      expect(choice("2–10")).toBeChecked();
      expect(screen.getByText("ann@example.com")).toBeInTheDocument();
    });

    test("finds every answer on the same screen after a reload", async () => {
      const first = render(<WorkspaceSetupPage />);
      await answerEverything();
      first.unmount();

      render(<WorkspaceSetupPage />);

      await expectEveryAnswer();
    });
  });

  describe("left panel", () => {
    // The layout's left column, which it hands to the wizard on this route.
    function WithLeftColumn() {
      const [column, setColumn] = useState<HTMLElement | null>(null);
      return (
        <>
          <aside ref={setColumn} data-testid="left-column" />
          <SplashPanelContext.Provider value={column}>
            <WorkspaceSetupPage />
          </SplashPanelContext.Provider>
        </>
      );
    }
    const panel = () => within(screen.getByTestId("left-column"));
    const still = () => panel().getByRole("presentation").dataset.still;

    test("previews the workspace as it is named, then shows each goal's own picture", async () => {
      render(<WithLeftColumn />);
      expect(panel().getAllByText("Workspace One")).not.toHaveLength(0);
      expect(panel().getByText("Ed")).toBeInTheDocument();
      expect(panel().getByText("Workspace Admin")).toBeInTheDocument();
      expect(panel().getAllByText("Workspace Member")).toHaveLength(2);
      type(screen.getByLabelText(/^Your name/), "Ada Park");
      type(screen.getByLabelText("Workspace name"), "Acme");
      expect(panel().getAllByText("Acme")).not.toHaveLength(0);
      expect(panel().getByText("Ada Park")).toBeInTheDocument();

      goToGoal();
      expect(panel().getAllByText("Acme")).not.toHaveLength(0);
      fireEvent.click(screen.getByText(CHANGE_GOAL));
      expect(still()).toBe("hero");
      // Its size is known before the file loads, so the panel does not start
      // out collapsed.
      expect(panel().getByRole("presentation")).toHaveAttribute(
        "width",
        "1638"
      );
      expect(panel().getByRole("presentation")).toHaveAttribute(
        "height",
        "1111"
      );
      fireEvent.click(screen.getByText(QUERY_GOAL));
      expect(still()).toBe("query");
      fireEvent.click(screen.getByText(EXPLORING));
      expect(still()).toBe("explore");
    });

    test("draws the workspace, the project and its instance as they are answered", async () => {
      render(<WithLeftColumn />);
      goToProject();

      expect(panel().getByText("Workspace One")).toBeInTheDocument();
      expect(panel().getByText("your project")).toBeInTheDocument();
      expect(panel().getByText("Sample Project Instance")).toBeInTheDocument();
      expect(panel().getByText("Sample database")).toBeInTheDocument();

      await nameProject();
      expect(panel().getByText("Payments")).toBeInTheDocument();

      fireEvent.click(screen.getByText(OWN_INSTANCE));
      expect(panel().getByText("Your instance")).toBeInTheDocument();
      expect(panel().getByText("Connected after setup")).toBeInTheDocument();
      expect(panel().getByText("Its databases")).toBeInTheDocument();
      expect(
        panel().queryByText("Sample Project Instance")
      ).not.toBeInTheDocument();
    });

    test("draws no instances for a project self-hosted leaves for later", async () => {
      mocks.isSaaSMode = false;
      render(<WithLeftColumn />);
      goToProject();

      expect(panel().getByText("your project")).toBeInTheDocument();
      expect(
        panel().queryByText("Sample Project Instance")
      ).not.toBeInTheDocument();

      await nameProject();
      expect(panel().getByText("Sample Project Instance")).toBeInTheDocument();
    });

    test("lists the team in the workspace as it is answered, five rows at most", async () => {
      render(<WithLeftColumn />);
      await goToTeam();

      expect(panel().getAllByText("Workspace One")).not.toHaveLength(0);
      expect(panel().getByText("Ed")).toBeInTheDocument();
      expect(panel().getByText("Workspace Admin")).toBeInTheDocument();
      expect(panel().queryByText("Workspace Member")).not.toBeInTheDocument();

      fireEvent.click(screen.getByText("2–10"));
      enterAddress("ann@example.com");
      enterAddress("not-an-address");
      expect(panel().getByText("ann@example.com")).toBeInTheDocument();
      expect(panel().getAllByText("Workspace Member")).toHaveLength(4);
      expect(panel().queryByText("not-an-address")).not.toBeInTheDocument();

      for (const name of ["bob", "cy", "dee", "eve", "fay"]) {
        enterAddress(`${name}@example.com`);
      }
      expect(panel().getByText("dee@example.com")).toBeInTheDocument();
      expect(panel().queryByText("eve@example.com")).not.toBeInTheDocument();
      expect(panel().getByText("+2 more")).toBeInTheDocument();

      fireEvent.click(screen.getByText("Just me"));
      expect(panel().queryByText("ann@example.com")).not.toBeInTheDocument();
      expect(panel().getByText("Ed")).toBeInTheDocument();
    });

    test("draws a wider column's picture larger, as one", () => {
      const width = vi
        .spyOn(HTMLElement.prototype, "clientWidth", "get")
        .mockReturnValue(1024);
      render(<WithLeftColumn />);

      const picture = screen.getByTestId("left-column").firstElementChild;
      expect(picture?.firstElementChild).toHaveStyle({
        transform: "scale(2)",
      });
      width.mockRestore();
    });
  });

  describe("events", () => {
    test("records entering setup once per account", () => {
      const first = render(<WorkspaceSetupPage />);
      first.rerender(<WorkspaceSetupPage />);
      first.unmount();
      render(<WorkspaceSetupPage />);

      expect(eventsNamed("workspace setup page entered")).toEqual([
        { setup_version: "v2" },
      ]);
    });

    test("records entering setup even when its marker cannot be saved", () => {
      mocks.saveIntroStateByKey.mockImplementation(() => {
        throw new Error("localStorage unavailable");
      });

      expect(() => render(<WorkspaceSetupPage />)).not.toThrow();
      expect(eventsNamed("workspace setup page entered")).toHaveLength(1);
    });

    test("records each Continue with the answers so far and no names", async () => {
      await startAtTeam({ goal: CHANGE_GOAL, database: OWN_INSTANCE });

      expect(eventsNamed("workspace setup screen continued")).toEqual([
        {
          setup_version: "v2",
          screen: 1,
          goal: "unselected",
          project_named: false,
        },
        {
          setup_version: "v2",
          screen: 2,
          goal: "create-database-change",
          project_named: false,
        },
        {
          setup_version: "v2",
          screen: 3,
          goal: "create-database-change",
          project_named: true,
          source: "own",
        },
      ]);
      expect(JSON.stringify(mocks.captureMetric.mock.calls)).not.toMatch(
        /Payments|ed@example\.com|Workspace One/
      );
    });

    test("records the sample request, then a finished setup with its goal, source, team size and invite count", async () => {
      await startAtTeam({ goal: QUERY_GOAL });
      fireEvent.click(screen.getByText("11–50"));
      enterAddress("ann@example.com");
      enterAddress("bob@example.com");
      enterAddress("not-an-address");
      fireEvent.click(
        screen.getByRole("button", { name: "Send 2 invites and finish" })
      );
      await landed();

      expect(
        mocks.captureMetric.mock.calls
          .map(([metric]) => metric as { event: string; properties: unknown })
          .filter(({ event }) => event !== "workspace setup screen continued")
          .slice(1)
      ).toEqual([
        {
          event: "sample instance requested",
          properties: { source: "workspace_setup" },
        },
        {
          event: "workspace setup submitted",
          properties: {
            setup_version: "v2",
            result: "finished",
            scenario: "query-data",
            source: "sample",
            team_size: "11-50",
            invite_count: 2,
            sample_enabled: true,
            collaboration_type: "team",
          },
        },
      ]);
      expect(JSON.stringify(mocks.captureMetric.mock.calls)).not.toMatch(
        /ann@example\.com/
      );
    });

    test("records a finished own-instance setup as having no sample, and requests none", async () => {
      await startAtTeam({ goal: CHANGE_GOAL, database: OWN_INSTANCE });
      finishAlone();
      await landed();

      expect(eventsNamed("sample instance requested")).toEqual([]);
      expect(eventsNamed("workspace setup submitted")).toEqual([
        {
          setup_version: "v2",
          result: "finished",
          scenario: "create-database-change",
          source: "own",
          team_size: "just-me",
          invite_count: 0,
          sample_enabled: false,
          collaboration_type: "solo",
        },
      ]);
    });

    test("records a failed Finish with the stage that failed, then the finish without the sample", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await startAtTeam({ goal: EXPLORING });
      finishAlone();
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Finish without the sample" })
      );
      await landed();

      const answers = {
        setup_version: "v2",
        scenario: "just-exploring",
        team_size: "just-me",
        invite_count: 0,
        collaboration_type: "solo",
      };
      expect(eventsNamed("workspace setup submitted")).toEqual([
        {
          ...answers,
          result: "failed",
          failure_stage: "sample",
          source: "sample",
          sample_enabled: true,
        },
        { ...answers, result: "finished", source: "own", sample_enabled: false },
      ]);
    });

    test("records a project ID the server refused as a failed project stage", async () => {
      mocks.createProject.mockRejectedValueOnce(alreadyExists());
      await startAtTeam({ database: OWN_INSTANCE });
      finishAlone();
      await screen.findByText(ID_EXISTS);

      expect(eventsNamed("workspace setup submitted")).toEqual([
        expect.objectContaining({
          result: "failed",
          failure_stage: "project",
          source: "own",
          sample_enabled: false,
        }),
      ]);
    });
  });

  describe("draft", () => {
    test.each([
      { deployment: "self-hosted", isSaaSMode: false, heading: "Your team" },
      {
        deployment: "Cloud",
        isSaaSMode: true,
        heading: "Create a project and choose its database",
      },
    ])("resumes a draft with no project name where $deployment allows it", ({
      isSaaSMode,
      heading,
    }) => {
      mocks.isSaaSMode = isSaaSMode;
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "",
        projectId: "",
        teamSize: "2-10",
      });

      render(<WorkspaceSetupPage />);

      expect(screen.getByRole("heading", { name: heading })).toBeInTheDocument();
    });

    test("resumes on screen 3 a self-hosted draft whose create was sent before its name was cleared", () => {
      mocks.isSaaSMode = false;
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "",
        projectId: "",
        teamSize: "2-10",
        attemptedProjectId: "payments-ab12",
      });

      render(<WorkspaceSetupPage />);

      expect(
        screen.getByRole("heading", {
          name: "Create a project and choose its database",
        })
      ).toBeInTheDocument();
    });

    test.each([
      {
        who: "another user of the browser",
        user: { email: "zoe@example.com", workspace: "workspaces/ws1" },
      },
      {
        who: "the same user in another workspace",
        user: { email: "ed@example.com", workspace: "workspaces/ws2" },
      },
    ])("does not show one account's draft to $who", async ({ user }) => {
      const first = startAtProject();
      await nameProject();
      first.unmount();

      mocks.currentUser = {
        ...mocks.currentUser,
        name: `users/${user.email}`,
        email: user.email,
        workspace: user.workspace,
      } as User;
      mocks.workspace = { ...mocks.workspace, name: user.workspace } as Workspace;
      render(<WorkspaceSetupPage />);

      expect(
        screen.getByRole("heading", { name: "Set up your workspace" })
      ).toBeInTheDocument();
    });

    test.each([
      {
        missing: "a goal",
        stored: { screen: "team", goal: "not-a-goal", emails: "x" },
        opensOn: "What do you want to do first?",
      },
      {
        missing: "a project name",
        stored: { screen: "team", goal: "query-data", projectId: "pay" },
        opensOn: "Create a project and choose its database",
      },
    ])("opens on the first unanswered screen when the stored draft is ahead of its answers, here $missing", ({
      stored,
      opensOn,
    }) => {
      storeDraft(stored);

      render(<WorkspaceSetupPage />);

      expect(
        screen.getByRole("heading", { name: opensOn })
      ).toBeInTheDocument();
    });

    test("is gone once Finish succeeds", async () => {
      const first = await startAtTeam();
      finishAlone();
      await landed();
      first.unmount();

      render(<WorkspaceSetupPage />);

      expect(
        screen.getByRole("heading", { name: "Set up your workspace" })
      ).toBeInTheDocument();
    });

    test("remembers across a reload that the project was created", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      const first = await startAtTeam();
      finishAlone();
      await failed();
      first.unmount();

      render(<WorkspaceSetupPage />);
      fireEvent.click(finishButton());

      await landed();
      expect(mocks.createProject).toHaveBeenCalledTimes(1);
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledTimes(2);
      expect(mocks.prepareSampleProjectInstance).toHaveBeenLastCalledWith(
        "projects/payments-ab12"
      );
    });

    test("remembers across a reload that the sample was prepared", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      const first = await startAtTeam();
      finishInviting("ann@example.com");
      await failed();
      first.unmount();

      render(<WorkspaceSetupPage />);
      fireEvent.click(screen.getByRole("button", { name: /and finish$/ }));

      await landed();
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledTimes(1);
    });

    test("keeps the sample as the project's database after a reload that cut its request short, and asks for it again", async () => {
      // The draft holds the project and no word on the sample; the server
      // lists the instance, which on self-hosted it does before it is ready.
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "sample",
        teamSize: "2-10",
        createdProject: "projects/payments-ab12",
      });
      mocks.sample = {
        available: true,
        instances: [{ instance: "projects/payments-ab12/instances/sample" }],
      };

      render(<WorkspaceSetupPage />);

      expect(
        screen.queryByText(/Its databases appear once your instance is connected/)
      ).not.toBeInTheDocument();
      fireEvent.click(finishButton());
      await landed();
      expect(writes()).toEqual(["updateUser", "updateWorkspace", "prepareSample"]);
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledWith(
        "projects/payments-ab12"
      );
      expect(eventsNamed("workspace setup submitted")).toEqual([
        expect.objectContaining({ source: "sample", sample_enabled: true }),
      ]);
    });

    test("keeps the choice to finish without the sample after a reload, though the server lists it under the project", async () => {
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "own",
        teamSize: "just-me",
        createdProject: "projects/payments-ab12",
      });
      mocks.sample = {
        available: true,
        instances: [{ instance: "projects/payments-ab12/instances/sample" }],
      };

      render(<WorkspaceSetupPage />);
      fireEvent.click(finishButton());

      await landed();
      expect(writes()).toEqual(["updateUser", "updateWorkspace"]);
      expect(eventsNamed("workspace setup submitted")).toEqual([
        expect.objectContaining({ source: "own", sample_enabled: false }),
      ]);
    });

    test("does not take another project's sample for this one's", () => {
      storeDraft({
        screen: "project",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "own",
        createdProject: "projects/payments-ab12",
      });
      mocks.sample = {
        available: true,
        instances: [{ instance: "projects/earlier/instances/sample" }],
      };

      render(<WorkspaceSetupPage />);

      expect(screen.getByLabelText(/^Project name/)).toBeDisabled();
      expect(choice(OWN_INSTANCE)).toBeChecked();
      expect(screen.queryByText(SAMPLE)).not.toBeInTheDocument();
      expect(
        screen.getByText("This workspace has already used its sample database.")
      ).toBeInTheDocument();
    });

    test("does not stop setup when the browser will not store anything", async () => {
      vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new DOMException("quota");
      });
      mocks.setRecentProject.mockImplementation(() => {
        throw new DOMException("quota");
      });
      await startAtTeam();

      finishAlone();

      await landed();
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: PROJECT_V1_ROUTE_DATABASES,
        params: { projectId: "payments-ab12" },
        query: {
          syncingInstance: "sample",
          [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
        },
      });
    });
  });

  describe("Finish", () => {
    test("creates nothing before Finish, then everything in order, and opens project databases", async () => {
      await startAtTeam({ goal: CHANGE_GOAL });
      fireEvent.click(screen.getByText("2–10"));
      enterAddress("Ann@Example.com");

      expect(writes()).toEqual([]);

      fireEvent.click(
        screen.getByRole("button", { name: "Send 1 invite and finish" })
      );

      await landed();
      expect(writes()).toEqual([
        "updateUser",
        "updateWorkspace",
        "createProject",
        "prepareSample",
        "projectMembers",
        "workspaceMembers",
      ]);
      expect(mocks.updateUser).toHaveBeenCalledWith(
        expect.objectContaining({
          user: expect.objectContaining({
            name: "users/ed@example.com",
            title: "Ed",
          }),
        }),
        REPORT
      );
      expect(mocks.updateWorkspace).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "workspaces/ws1",
          title: "Workspace One",
        }),
        ["title"],
        REPORT
      );
      expect(mocks.createProject).toHaveBeenCalledWith(
        "Payments",
        "payments-ab12",
        REPORT
      );
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledWith(
        "projects/payments-ab12"
      );
      const [policyRead, policyReadOptions] =
        mocks.getProjectIamPolicy.mock.calls[0];
      expect(policyRead).toMatchObject({ resource: "projects/payments-ab12" });
      expect(
        policyReadOptions.contextValues.get(skipErrorNotificationContextKey)
      ).toBe(true);
      expect(mocks.updateProjectIamPolicy).toHaveBeenCalledWith(
        "projects/payments-ab12",
        expect.anything(),
        REPORT
      );
      expect(projectBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:ann@example.com"]],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
      expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalledWith(
        [{ member: "user:ann@example.com", roles: ["roles/workspaceMember"] }],
        REPORT
      );
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: PROJECT_V1_ROUTE_DATABASES,
        params: { projectId: "payments-ab12" },
        query: {
          syncingInstance: "sample",
          [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
        },
      });
    });

    test("keeps the sample instance sync hint when server info is not refreshed", async () => {
      mocks.prepareSampleProjectInstance.mockResolvedValueOnce({
        name: "projects/payments-ab12/instances/sample",
      });
      await startAtTeam();

      finishAlone();

      await landed();
      expect(mocks.sample.instances).toEqual([]);
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: PROJECT_V1_ROUTE_DATABASES,
        params: { projectId: "payments-ab12" },
        query: {
          syncingInstance: "sample",
          [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
        },
      });
    });

    test("adds the workspace role to an invitee without taking away one they already hold", async () => {
      mocks.workspacePolicy = {
        bindings: [
          { role: "roles/workspaceAdmin", members: ["user:ed@example.com"] },
          { role: "roles/workspaceDBA", members: ["user:ann@example.com"] },
        ],
      } as IamPolicy;
      await startAtTeam();

      finishInviting("ann@example.com");

      await landed();
      expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalledWith(
        [
          {
            member: "user:ann@example.com",
            roles: ["roles/workspaceDBA", "roles/workspaceMember"],
          },
        ],
        REPORT
      );
    });

    test("builds the workspace write on the policy the server holds then, not the one the page loaded", async () => {
      await startAtTeam();
      // Since the page loaded, ann was made a DBA from somewhere else.
      mocks.fetchWorkspaceIamPolicy.mockImplementation(async () => {
        mocks.workspacePolicy = {
          bindings: [
            { role: "roles/workspaceAdmin", members: ["user:ed@example.com"] },
            { role: "roles/workspaceDBA", members: ["user:ann@example.com"] },
          ],
        } as IamPolicy;
        return mocks.workspacePolicy;
      });

      finishInviting("ann@example.com");

      await landed();
      expect(mocks.fetchWorkspaceIamPolicy).toHaveBeenCalledWith(false, REPORT);
      expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalledWith(
        [
          {
            member: "user:ann@example.com",
            roles: ["roles/workspaceDBA", "roles/workspaceMember"],
          },
        ],
        REPORT
      );
    });

    test("hands the goal and team size to the guide, and ends mandatory setup on Cloud", async () => {
      await startAtTeam({ goal: EXPLORING });
      fireEvent.click(screen.getByText("Just me"));

      expect(readSelectedGuideScenarioId()).toBeUndefined();
      expect(readGuideTeamSize()).toBeUndefined();

      fireEvent.click(finishButton());

      await landed();
      expect(readSelectedGuideScenarioId()).toBe("just-exploring");
      expect(readGuideTeamSize()).toBe("just-me");
      expect(readWorkspaceSetupFinished("workspaces/ws1")).toBe(true);
      expect(mocks.setRecentProject).toHaveBeenCalledWith(
        "projects/payments-ab12"
      );
    });

    test("leaves the setup flag alone on self-hosted, where setup is not mandatory", async () => {
      mocks.isSaaSMode = false;
      await startAtTeam();
      finishAlone();

      await landed();
      expect(readWorkspaceSetupFinished("workspaces/ws1")).toBeUndefined();
    });

    test("does not save an emptied name over the account's", async () => {
      render(<WorkspaceSetupPage />);
      type(screen.getByLabelText(/^Your name/), " ");
      await goToTeam();
      finishAlone();

      await landed();
      expect(writes()).toEqual([
        "updateWorkspace",
        "createProject",
        "prepareSample",
      ]);
    });

    test("finishes the own-instance path on project instances", async () => {
      await startAtTeam({ database: OWN_INSTANCE });
      finishAlone();

      await landed();
      expect(writes()).toEqual(["updateUser", "updateWorkspace", "createProject"]);
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: PROJECT_V1_ROUTE_INSTANCES,
        params: { projectId: "payments-ab12" },
        query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO },
      });
    });

    test("holds the screen, with what it is creating, while Finish runs", async () => {
      const release = holdNextCall(mocks.prepareSampleProjectInstance);
      await startAtTeam();
      finishInviting("ann@example.com");

      expect(
        await screen.findByRole("button", { name: "Setting up…" })
      ).toBeDisabled();
      expect(backButton()).toBeDisabled();
      expect(
        screen.getByText("Creating Payments and its sample database…")
      ).toBeInTheDocument();
      for (const radio of screen.getAllByRole("radio")) {
        expect(radio).toHaveAttribute("aria-disabled", "true");
      }
      expect(invitees()).toBeDisabled();
      expect(
        screen.getByRole("button", { name: "Remove ann@example.com" })
      ).toBeDisabled();

      release();
      await landed();
    });

    test("shows only the progress of a retry while it runs, not the failure before it", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishAlone();
      await failed();
      const release = holdNextCall(mocks.prepareSampleProjectInstance);

      tryAgain();

      expect(
        await screen.findByRole("button", { name: "Setting up…" })
      ).toBeDisabled();
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Finish without the sample" })
      ).not.toBeInTheDocument();

      release();
      await landed();
    });

    test("drops a failure's report once its screen is left", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishAlone();
      await failed();

      fireEvent.click(backButton());
      fireEvent.click(screen.getByText(OWN_INSTANCE));
      fireEvent.click(continueButton());

      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Finish without the sample" })
      ).not.toBeInTheDocument();
      expect(finishButton()).toBeEnabled();
    });

    test("takes the invitees off the project when the write that added them lost its answer", async () => {
      loseAnswerOf(mocks.updateProjectIamPolicy);
      await startAtTeam();
      finishInviting("ann@example.com");
      await failed();
      expect(server.projectPolicy.bindings).toHaveLength(3);

      fireEvent.click(screen.getByText("Just me"));
      finishAgain();

      await landed();
      expect(projectBindings()).toEqual([OWNER_BINDING]);
    });

    test.each([
      {
        step: "saving the names",
        fail: () => mocks.updateWorkspace.mockRejectedValueOnce(unavailable()),
        message: "Could not save your name and the workspace name.",
      },
      {
        step: "creating the project",
        fail: () => mocks.createProject.mockRejectedValueOnce(unavailable()),
        message: "Could not create Payments.",
      },
      {
        step: "preparing the sample",
        fail: () =>
          mocks.prepareSampleProjectInstance.mockRejectedValueOnce(
            unavailable()
          ),
        message: "Could not prepare the sample database.",
      },
      {
        step: "adding the invitees to the project",
        fail: () =>
          mocks.updateProjectIamPolicy.mockRejectedValueOnce(unavailable()),
        message: "Could not add the people you listed.",
      },
      {
        step: "adding the invitees to the workspace",
        fail: () =>
          mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable()),
        message: "Could not add the people you listed.",
      },
    ])("stays on the team screen and names the step when $step fails", async ({
      fail,
      message,
    }) => {
      fail();
      await startAtTeam();
      finishInviting("ann@example.com");

      const alert = await failed();
      expect(alert).toHaveTextContent(message);
      expect(alert).toHaveTextContent("offline");
      expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
      expect(backButton()).toBeEnabled();
      expect(mocks.routerReplace).not.toHaveBeenCalled();
      expect(readWorkspaceSetupFinished("workspaces/ws1")).toBeUndefined();
      expect(readSelectedGuideScenarioId()).toBeUndefined();
    });

    test.each([
      {
        answer: "the team size",
        change: () => fireEvent.click(screen.getByText("Just me")),
        button: "Finish",
      },
      {
        answer: "the addresses",
        change: () =>
          fireEvent.click(
            screen.getByRole("button", { name: "Remove bob@example.com" })
          ),
        button: "Send 1 invite and finish",
      },
    ])("drops a failed attempt's error once $answer change", async ({
      change,
      button,
    }) => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();

      change();

      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: button })).toBeEnabled();
    });

    test("keeps another step's failure, and Finish without the sample, when the team answers change", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();

      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );

      expect(screen.getByRole("alert")).toHaveTextContent(
        "Could not prepare the sample database."
      );
      expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
      expect(
        screen.getByRole("button", { name: "Finish without the sample" })
      ).toBeEnabled();
    });

    test("has no members step for an account finishing alone on its first attempt", async () => {
      mocks.getProjectIamPolicy.mockRejectedValue(unavailable());
      await startAtTeam();

      finishAlone();

      await landed();
      expect(mocks.getProjectIamPolicy).not.toHaveBeenCalled();
    });

    test("writes no members when the project's access policy cannot be read, and keeps the owner on the retry", async () => {
      mocks.getProjectIamPolicy.mockRejectedValueOnce(
        unavailable("policy unavailable")
      );
      await startAtTeam();
      finishInviting("ann@example.com");

      const alert = await failed();
      expect(alert).toHaveTextContent("Could not add the people you listed.");
      expect(alert).toHaveTextContent("policy unavailable");
      expect(mocks.updateProjectIamPolicy).not.toHaveBeenCalled();
      expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();

      tryAgain();

      await landed();
      expect(projectBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:ann@example.com"]],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
    });

    test("tries again without creating the project or requesting the sample twice", async () => {
      mocks.prepareSampleProjectInstance.mockResolvedValueOnce({
        name: "projects/payments-ab12/instances/sample",
      });
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com");
      await failed();

      tryAgain();

      await landed();
      expect(writes()).toEqual([
        "updateUser",
        "updateWorkspace",
        "createProject",
        "prepareSample",
        "projectMembers",
        "workspaceMembers",
        "updateUser",
        "updateWorkspace",
        "projectMembers",
        "workspaceMembers",
      ]);
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      expect(mocks.sample.instances).toEqual([]);
      expect(mocks.routerReplace).toHaveBeenCalledWith({
        name: PROJECT_V1_ROUTE_DATABASES,
        params: { projectId: "payments-ab12" },
        query: {
          syncingInstance: "sample",
          [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
        },
      });
    });

    test("drops an invitee removed after a failed attempt from the project too", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();
      expect(projectBindings()).toContainEqual([
        "roles/projectDeveloper",
        ["user:ann@example.com", "user:bob@example.com"],
      ]);

      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );
      finishAgain();

      await landed();
      expect(projectBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:ann@example.com"]],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
      expect(mocks.patchWorkspaceIamPolicy).toHaveBeenLastCalledWith(
        [{ member: "user:ann@example.com", roles: ["roles/workspaceMember"] }],
        REPORT
      );
    });

    test("takes every invitee off the project when the answer becomes Just me after a failed attempt", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com");
      await failed();

      fireEvent.click(screen.getByText("Just me"));
      finishAgain();

      await landed();
      expect(projectBindings()).toEqual([OWNER_BINDING]);
      expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalledTimes(1);
    });

    test("takes off the project only the addresses it added, and leaves what anyone else was given", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();
      // From somewhere else, before the retry: zoe is made a developer, and
      // bob is given a developer role that expires.
      const expiring = {
        role: "roles/projectDeveloper",
        members: ["user:bob@example.com"],
        condition: { expression: 'request.time < timestamp("2027-01-01T00:00:00Z")' },
      };
      server.projectPolicy = {
        ...server.projectPolicy,
        bindings: [
          ...server.projectPolicy.bindings.map((binding) =>
            binding.role === "roles/projectDeveloper"
              ? { ...binding, members: [...binding.members, "user:zoe@example.com"] }
              : binding
          ),
          expiring,
        ],
      } as IamPolicy;

      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );
      finishAgain();

      await landed();
      expect(
        server.projectPolicy.bindings.map(({ role, members, condition }) => [
          role,
          members,
          condition?.expression ?? "",
        ])
      ).toEqual([
        ["roles/projectOwner", ["user:ed@example.com"], ""],
        [expiring.role, expiring.members, expiring.condition.expression],
        [
          "roles/projectDeveloper",
          ["user:ann@example.com", "user:zoe@example.com"],
          "",
        ],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"], ""],
      ]);
    });

    test("still knows after a reload which addresses it added to the project", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      const first = await startAtTeam();
      finishInviting("ann@example.com");
      await failed();
      first.unmount();

      render(<WorkspaceSetupPage />);
      fireEvent.click(screen.getByText("Just me"));
      fireEvent.click(finishButton());

      await landed();
      expect(projectBindings()).toEqual([OWNER_BINDING]);
    });

    test("does not take away a role an address held before the wizard listed it", async () => {
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "own",
        teamSize: "2-10",
        emails: ["ann@example.com", "bob@example.com"],
        createdProject: "projects/payments-ab12",
      });
      // From another browser, ann was given both roles already.
      server.projectPolicy = {
        ...server.projectPolicy,
        bindings: [
          ...server.projectPolicy.bindings,
          { role: "roles/projectDeveloper", members: ["user:ann@example.com"] },
          { role: "roles/sqlEditorReadUser", members: ["user:ann@example.com"] },
        ],
      } as IamPolicy;
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      render(<WorkspaceSetupPage />);
      fireEvent.click(
        screen.getByRole("button", { name: "Send 2 invites and finish" })
      );
      await failed();

      fireEvent.click(screen.getByText("Just me"));
      finishAgain();

      await landed();
      expect(serverBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:ann@example.com"]],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
    });

    test("keeps counting an address it added while the write that took it away has not been made", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );
      mocks.updateProjectIamPolicy.mockRejectedValueOnce(unavailable());
      finishAgain();
      await failed();

      tryAgain();

      await landed();
      expect(serverBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:ann@example.com"]],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
    });

    test("stops counting an address once a write has taken it away, so a later grant of it stays", async () => {
      mocks.patchWorkspaceIamPolicy
        .mockRejectedValueOnce(unavailable())
        .mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );
      finishAgain();
      await failed();
      // From somewhere else, bob is made a developer again.
      grantElsewhere("roles/projectDeveloper", "user:bob@example.com");

      tryAgain();

      await landed();
      expect(serverBindings()).toEqual([
        OWNER_BINDING,
        [
          "roles/projectDeveloper",
          ["user:ann@example.com", "user:bob@example.com"],
        ],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
    });

    test("forgets what a refused write would have added, so another writer's grant stays", async () => {
      // Another browser gives ann both roles between the wizard's read and
      // its write, so the server refuses the write as made on a stale policy.
      mocks.getProjectIamPolicy.mockImplementationOnce(async () => {
        const read = server.projectPolicy;
        server.projectPolicy = {
          etag: "9",
          bindings: [
            ...read.bindings,
            { role: "roles/projectDeveloper", members: ["user:ann@example.com"] },
            { role: "roles/sqlEditorReadUser", members: ["user:ann@example.com"] },
          ],
        } as IamPolicy;
        return read;
      });
      await startAtTeam();
      finishInviting("ann@example.com");
      expect(await failed()).toHaveTextContent("concurrent update");

      fireEvent.click(screen.getByText("Just me"));
      finishAgain();

      await landed();
      expect(serverBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:ann@example.com"]],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
    });

    test("stops counting a removed address once the next write lands, though the removal's answer was lost", async () => {
      mocks.patchWorkspaceIamPolicy
        .mockRejectedValueOnce(unavailable())
        .mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com", "bob@example.com");
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );
      // The write that takes bob away is made, and its answer lost.
      loseAnswerOf(mocks.updateProjectIamPolicy);
      finishAgain();
      await failed();
      // The next attempt writes the policy again, as it already is, then fails
      // at the workspace.
      tryAgain();
      await waitFor(() =>
        expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalledTimes(2)
      );
      await failed();
      // From somewhere else, bob is made a developer again.
      grantElsewhere("roles/projectDeveloper", "user:bob@example.com");

      tryAgain();

      await landed();
      expect(serverBindings()).toEqual([
        OWNER_BINDING,
        [
          "roles/projectDeveloper",
          ["user:ann@example.com", "user:bob@example.com"],
        ],
        ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
      ]);
    });

    test("lets no lost write of an invitee's roles land after the next attempt took the invitee off", async () => {
      await startAtTeam({ database: OWN_INSTANCE });
      const land = delayLostWrite(mocks.updateProjectIamPolicy);
      finishInviting("bob@example.com");
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );

      finishAgain();
      await landed();
      land();
      await settle();

      expect(serverBindings()).toEqual([OWNER_BINDING]);
    });

    test("lets no lost write that took an invitee off land after the next attempt listed them again", async () => {
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam({ database: OWN_INSTANCE });
      finishInviting("bob@example.com");
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Remove bob@example.com" })
      );
      const land = delayLostWrite(mocks.updateProjectIamPolicy);
      finishAgain();
      await waitFor(() =>
        expect(mocks.updateProjectIamPolicy).toHaveBeenCalledTimes(2)
      );
      await failed();
      enterAddress("bob@example.com");

      finishAgain();
      await landed();
      land();
      await settle();

      expect(serverBindings()).toEqual([
        OWNER_BINDING,
        ["roles/projectDeveloper", ["user:bob@example.com"]],
        ["roles/sqlEditorReadUser", ["user:bob@example.com"]],
      ]);
    });

    describe("with two tabs on the wizard", () => {
      const SAVED_ELSEWHERE = "This setup continued in another tab";
      const continueHere = (tab: HTMLElement) =>
        fireEvent.click(
          within(tab).getByRole("button", { name: "Continue here" })
        );
      // Two tabs on one draft. In a browser only the other tab hears of a
      // save, so the tab that saved is closed before the notice is sent.
      const twoTabs = () => {
        const a = render(<WorkspaceSetupPage />);
        const b = render(<WorkspaceSetupPage />);
        return { a: a.container, closeA: a.unmount, b: b.container };
      };

      test("stops a tab when the other saves, until it continues there with the saved answers", async () => {
        storeDraft({
          screen: "team",
          goal: "query-data",
          projectTitle: "Payments",
          projectId: "payments-ab12",
          source: "sample",
          teamSize: "just-me",
        });
        const tab = twoTabs();
        mocks.prepareSampleProjectInstance.mockRejectedValueOnce(
          unavailable()
        );
        fireEvent.click(within(tab.a).getByRole("button", { name: /^Finish$/ }));
        await within(tab.a).findByRole("alert");
        tab.closeA();

        anotherTabSaved();

        expect(within(tab.b).getByText(SAVED_ELSEWHERE)).toBeInTheDocument();
        expect(within(tab.b).queryByRole("radio")).not.toBeInTheDocument();
        continueHere(tab.b);
        fireEvent.click(within(tab.b).getByRole("button", { name: /^Finish$/ }));

        await landed();
        expect(mocks.createProject).toHaveBeenCalledTimes(1);
        expect(mocks.prepareSampleProjectInstance).toHaveBeenLastCalledWith(
          "projects/payments-ab12"
        );
      });

      test("continues with the answers the other tab saved, not its own older ones", async () => {
        storeDraft({
          screen: "team",
          goal: "query-data",
          projectTitle: "Orders",
          projectId: "orders-ab12",
          source: "own",
          teamSize: "2-10",
          emails: ["ann@example.com"],
        });
        const tabB = render(<WorkspaceSetupPage />).container;
        // The other tab saves a Finish that created Payments.
        storeDraft({
          screen: "team",
          goal: "query-data",
          projectTitle: "Payments",
          projectId: "payments-ab12",
          source: "own",
          teamSize: "2-10",
          emails: ["ann@example.com"],
          attemptedProjectId: "payments-ab12",
          createdProject: "projects/payments-ab12",
        });
        server.projects.set("projects/payments-ab12", {
          name: "projects/payments-ab12",
          title: "Payments",
        });
        anotherTabSaved();

        continueHere(tabB);
        expect(
          within(tabB).getByText(/propose changes in Payments\./)
        ).toBeInTheDocument();
        fireEvent.click(
          within(tabB).getByRole("button", { name: "Send 1 invite and finish" })
        );

        await landed();
        expect(mocks.createProject).not.toHaveBeenCalled();
        expect(mocks.routerReplace).toHaveBeenCalledWith({
          name: PROJECT_V1_ROUTE_INSTANCES,
          params: { projectId: "payments-ab12" },
          query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO },
        });
      });

      test("takes off the project, from the other tab, an invitee that tab's attempt added", async () => {
        storeDraft({
          screen: "team",
          goal: "query-data",
          projectTitle: "Payments",
          projectId: "payments-ab12",
          source: "own",
          teamSize: "2-10",
          emails: ["ann@example.com", "bob@example.com"],
        });
        const tab = twoTabs();
        mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
        fireEvent.click(
          within(tab.a).getByRole("button", { name: "Send 2 invites and finish" })
        );
        await within(tab.a).findByRole("alert");
        tab.closeA();
        anotherTabSaved();

        continueHere(tab.b);
        fireEvent.click(
          within(tab.b).getByRole("button", { name: "Remove bob@example.com" })
        );
        fireEvent.click(
          within(tab.b).getByRole("button", { name: "Send 1 invite and finish" })
        );

        await landed();
        expect(serverBindings()).toEqual([
          OWNER_BINDING,
          ["roles/projectDeveloper", ["user:ann@example.com"]],
          ["roles/sqlEditorReadUser", ["user:ann@example.com"]],
        ]);
      });

      test("stops a running Finish once the wizard is left, so the answers of a later Finish stand", async () => {
        const release = holdNextCall(mocks.prepareSampleProjectInstance);
        const first = await startAtTeam();
        finishInviting("bob@example.com");
        await waitFor(() =>
          expect(mocks.prepareSampleProjectInstance).toHaveBeenCalled()
        );

        first.unmount();
        render(<WorkspaceSetupPage />);
        finishAlone();
        await landed();
        release();
        await settle();

        expect(serverBindings()).toEqual([OWNER_BINDING]);
        expect(mocks.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
      });

      test("lets no roles write of a stopped tab land after the other tab finished without its invitee", async () => {
        storeDraft({
          screen: "team",
          goal: "query-data",
          projectTitle: "Payments",
          projectId: "payments-ab12",
          source: "own",
          teamSize: "2-10",
          emails: ["bob@example.com"],
        });
        const tab = twoTabs();
        const release = holdNextCall(mocks.updateProjectIamPolicy);
        fireEvent.click(
          within(tab.a).getByRole("button", { name: "Send 1 invite and finish" })
        );
        await waitFor(() =>
          expect(mocks.updateProjectIamPolicy).toHaveBeenCalled()
        );

        // Both tabs hear of the save, as tab A would once tab B continues.
        anotherTabSaved();
        continueHere(tab.b);
        fireEvent.click(within(tab.b).getByText("Just me"));
        fireEvent.click(
          within(tab.b).getByRole("button", { name: /^Finish$/ })
        );
        await landed();
        release();
        await settle();

        expect(serverBindings()).toEqual([OWNER_BINDING]);
      });

      describe("when the other tab takes setup up while this one works", () => {
        const stored = () =>
          JSON.parse(localStorage.getItem(DRAFT_KEY) ?? "null");
        const otherTabSaves = (draft: Record<string, unknown>) => {
          storeDraft(draft);
          anotherTabSaved();
        };
        const otherTabEnds = () => {
          localStorage.removeItem(DRAFT_KEY);
          anotherTab(null);
        };
        // The draft as the other tab saves it after taking it up: its own
        // instance, and nobody invited.
        const takenUpAlone = () => ({
          ...stored(),
          source: "own",
          teamSize: "just-me",
          emails: [],
        });

        test("gives the invitee nothing once the other tab finished setup alone during the sample", async () => {
          const release = holdNextCall(mocks.prepareSampleProjectInstance);
          await startAtTeam();
          finishInviting("bob@example.com");
          await waitFor(() =>
            expect(mocks.prepareSampleProjectInstance).toHaveBeenCalled()
          );

          otherTabSaves(takenUpAlone());
          otherTabEnds();
          expect(
            screen.getByRole("button", { name: "Setting up…" })
          ).toBeDisabled();
          release();

          expect(
            await screen.findByText("This setup ended in another tab")
          ).toBeInTheDocument();
          expect(writes()).toEqual([
            "updateUser",
            "updateWorkspace",
            "createProject",
            "prepareSample",
          ]);
          expect(serverBindings()).toEqual([OWNER_BINDING]);
          expect(localStorage.getItem(DRAFT_KEY)).toBeNull();
          expect(mocks.routerReplace).not.toHaveBeenCalled();
        });

        test("creates nothing once the other tab took setup up while the names were saved", async () => {
          const release = holdNextCall(mocks.updateUser);
          await startAtTeam();
          finishAlone();
          await waitFor(() => expect(mocks.updateUser).toHaveBeenCalled());

          otherTabSaves(takenUpAlone());
          release();

          expect(await screen.findByText(SAVED_ELSEWHERE)).toBeInTheDocument();
          expect(writes()).toEqual(["updateUser"]);
        });

        test("asks for no sample and saves over nothing once the other tab took setup up during the create, which that tab then finds", async () => {
          const release = holdNextCall(mocks.createProject);
          await startAtTeam();
          finishAlone();
          await waitFor(() => expect(mocks.createProject).toHaveBeenCalled());

          const theirs = takenUpAlone();
          otherTabSaves(theirs);
          release();

          expect(await screen.findByText(SAVED_ELSEWHERE)).toBeInTheDocument();
          expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
          expect(stored()).toEqual(theirs);

          continueHere(document.body);
          fireEvent.click(finishButton());
          await landed();
          expect(mocks.createProject).toHaveBeenCalledTimes(2);
          expect(server.projects.size).toBe(1);
          expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
        });

        test.each([
          {
            read: "the project's roles",
            call: mocks.getProjectIamPolicy,
            sent: ["updateUser", "updateWorkspace", "createProject"],
          },
          {
            read: "the workspace's members",
            call: mocks.fetchWorkspaceIamPolicy,
            sent: [
              "updateUser",
              "updateWorkspace",
              "createProject",
              "projectMembers",
            ],
          },
        ])("gives the invitee nothing more once the other tab took setup up while $read were read", async ({
          call,
          sent,
        }) => {
          const release = holdNextCall(call);
          await startAtTeam({ database: OWN_INSTANCE });
          finishInviting("bob@example.com");
          await waitFor(() => expect(call).toHaveBeenCalled());

          otherTabSaves(takenUpAlone());
          release();

          expect(await screen.findByText(SAVED_ELSEWHERE)).toBeInTheDocument();
          expect(writes()).toEqual(sent);
        });

        test.each([
          { outcome: "succeeds", error: undefined },
          { outcome: "fails", error: unavailable() },
        ])("neither leaves nor reports when its last write $outcome after the other tab took setup up", async ({
          error,
        }) => {
          const answer = holdNextCall(mocks.patchWorkspaceIamPolicy);
          await startAtTeam({ database: OWN_INSTANCE });
          finishInviting("bob@example.com");
          await waitFor(() =>
            expect(mocks.patchWorkspaceIamPolicy).toHaveBeenCalled()
          );

          const theirs = takenUpAlone();
          otherTabSaves(theirs);
          answer(error);

          expect(await screen.findByText(SAVED_ELSEWHERE)).toBeInTheDocument();
          expect(screen.queryByRole("alert")).not.toBeInTheDocument();
          expect(mocks.routerReplace).not.toHaveBeenCalled();
          expect(stored()).toEqual(theirs);
          expect(readWorkspaceSetupFinished("workspaces/ws1")).toBeUndefined();
          expect(eventsNamed("workspace setup submitted")).toEqual([]);
        });

        test("saves nothing it finds about a lost create once the other tab took setup up", async () => {
          storeDraft({
            screen: "project",
            goal: "query-data",
            projectTitle: "Payments",
            projectId: "payments-ab12",
            source: "sample",
            attemptedProjectId: "payments-ab12",
          });
          const answer = holdNextCall(mocks.getProject);
          render(<WorkspaceSetupPage />);
          await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());

          const theirs = {
            screen: "project",
            goal: "query-data",
            projectTitle: "Orders",
            projectId: "orders-ab12",
            source: "own",
          };
          otherTabSaves(theirs);
          answer();
          await settle();

          expect(stored()).toEqual(theirs);
        });

        test("neither clears the draft nor leaves when the other tab took setup up during a skip", async () => {
          mocks.isSaaSMode = false;
          const release = holdNextCall(mocks.updateUser);
          startAtProject();
          await nameProject();
          fireEvent.click(
            screen.getByRole("button", { name: "I'll do this later" })
          );
          await waitFor(() => expect(mocks.updateUser).toHaveBeenCalled());

          const theirs = { ...stored(), projectTitle: "Orders" };
          otherTabSaves(theirs);
          release();
          await settle();

          expect(writes()).toEqual(["updateUser"]);
          expect(mocks.routerReplace).not.toHaveBeenCalled();
          expect(stored()).toEqual(theirs);
        });

        test("reports no failure of a skip the other tab took setup up from", async () => {
          mocks.isSaaSMode = false;
          const fail = holdNextCall(mocks.updateUser);
          startAtProject();
          await nameProject();
          fireEvent.click(
            screen.getByRole("button", { name: "I'll do this later" })
          );
          await waitFor(() => expect(mocks.updateUser).toHaveBeenCalled());

          otherTabSaves({ ...stored(), projectTitle: "Orders" });
          continueHere(document.body);
          fail(unavailable());
          await settle();

          expect(screen.getByLabelText(/^Project name/)).toHaveValue("Orders");
          expect(screen.queryByRole("alert")).not.toBeInTheDocument();
        });

        test("saves the draft as soon as it continues here, so a tab still working on it stops", () => {
          storeDraft({ screen: "goal", goal: "query-data" });
          render(<WorkspaceSetupPage />);
          anotherTabSaved();
          const before = localStorage.getItem(DRAFT_KEY);

          continueHere(document.body);

          expect(localStorage.getItem(DRAFT_KEY)).not.toBe(before);
        });

        test("saves the draft before Finish writes anything, even with no answer changed", async () => {
          await startAtTeam();
          fireEvent.click(screen.getByText("Just me"));
          const before = localStorage.getItem(DRAFT_KEY);
          holdNextCall(mocks.updateUser);

          fireEvent.click(finishButton());

          await waitFor(() => expect(mocks.updateUser).toHaveBeenCalled());
          expect(localStorage.getItem(DRAFT_KEY)).not.toBe(before);
        });
      });

      test.each([
        { change: "another key", key: "bb.recent-visit", newValue: "[]" },
        {
          change: "another account's draft",
          key: storageKeyWorkspaceSetupDraft(
            "workspaces/ws1",
            "ann@example.com"
          ),
          newValue: "{}",
        },
        {
          change: "another workspace's draft",
          key: storageKeyWorkspaceSetupDraft(
            "workspaces/ws2",
            "ed@example.com"
          ),
          newValue: "{}",
        },
        { change: "all of storage, by clearing it", key: null, newValue: null },
      ])("keeps working when another tab changes $change", ({
        key,
        newValue,
      }) => {
        render(<WorkspaceSetupPage />);

        act(() => {
          window.dispatchEvent(new StorageEvent("storage", { key, newValue }));
        });

        expect(screen.queryByText(SAVED_ELSEWHERE)).not.toBeInTheDocument();
        expect(
          screen.queryByText("This setup ended in another tab")
        ).not.toBeInTheDocument();
        expect(continueButton()).toBeEnabled();
      });

      test.each([
        {
          report: "a failed Finish",
          arrange: async () => {
            mocks.prepareSampleProjectInstance.mockRejectedValueOnce(
              unavailable()
            );
            await startAtTeam();
            finishAlone();
          },
        },
        {
          report: "a failed skip",
          arrange: async () => {
            mocks.isSaaSMode = false;
            mocks.updateUser.mockRejectedValueOnce(unavailable());
            startAtProject();
            fireEvent.click(
              screen.getByRole("button", { name: "I'll do this later" })
            );
          },
        },
      ])("drops the report of $report when it continues after the other tab saved", async ({
        arrange,
      }) => {
        await arrange();
        await failed();

        anotherTabSaved();
        fireEvent.click(screen.getByRole("button", { name: "Continue here" }));

        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      });

      test("sends a tab to the workspace when the other tab ends setup", () => {
        render(<WorkspaceSetupPage />);

        anotherTab(null);
        fireEvent.click(
          screen.getByRole("button", { name: "Go to the workspace" })
        );

        expect(screen.getByText("This setup ended in another tab")).toBeInTheDocument();
        expect(mocks.routerReplace).toHaveBeenCalledWith({
          name: WORKSPACE_ROUTE_LANDING,
        });
      });
    });

    test("keeps its own progress when the browser stops storing partway", async () => {
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "sample",
        teamSize: "2-10",
        emails: ["ann@example.com"],
      });
      render(<WorkspaceSetupPage />);
      vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
        throw new DOMException("quota");
      });
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      fireEvent.click(
        screen.getByRole("button", { name: "Send 1 invite and finish" })
      );
      await failed();

      tryAgain();

      await landed();
      expect(mocks.createProject).toHaveBeenCalledTimes(1);
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledTimes(1);
    });

    test("writes nothing to a project it added nobody to, whoever holds roles there by now", async () => {
      // A Finish that created the project and stopped; it is resumed after
      // teammates were given roles on the project from another browser.
      storeDraft({
        screen: "team",
        goal: "query-data",
        projectTitle: "Payments",
        projectId: "payments-ab12",
        source: "own",
        teamSize: "just-me",
        createdProject: "projects/payments-ab12",
      });
      server.projectPolicy = {
        ...server.projectPolicy,
        bindings: [
          ...server.projectPolicy.bindings,
          { role: "roles/projectDeveloper", members: ["user:zoe@example.com"] },
          { role: "roles/sqlEditorReadUser", members: ["user:yan@example.com"] },
        ],
      } as IamPolicy;
      render(<WorkspaceSetupPage />);

      fireEvent.click(finishButton());

      await landed();
      expect(writes()).toEqual(["updateUser", "updateWorkspace"]);
    });

    test("offers to finish without the sample only when the sample is what failed", async () => {
      mocks.createProject.mockRejectedValueOnce(unavailable());
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(
        unavailable("no capacity")
      );
      await startAtTeam();
      finishAlone();
      await failed();
      expect(
        screen.queryByRole("button", { name: "Finish without the sample" })
      ).not.toBeInTheDocument();

      tryAgain();
      fireEvent.click(
        await screen.findByRole("button", { name: "Finish without the sample" })
      );

      await landed();
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledTimes(1);
      expect(mocks.createProject).toHaveBeenCalledTimes(2);
    });

    test("does not ask for the sample again on a retry after Finish without the sample", async () => {
      mocks.prepareSampleProjectInstance.mockRejectedValueOnce(unavailable());
      mocks.patchWorkspaceIamPolicy.mockRejectedValueOnce(unavailable());
      await startAtTeam();
      finishInviting("ann@example.com");
      await failed();
      fireEvent.click(
        screen.getByRole("button", { name: "Finish without the sample" })
      );
      expect(await failed()).toHaveTextContent(
        "Could not add the people you listed."
      );

      tryAgain();

      await landed();
      expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledTimes(1);
      expect(eventsNamed("workspace setup submitted").at(-1)).toMatchObject({
        result: "finished",
        source: "own",
        sample_enabled: false,
      });
    });

    describe("when the project ID already exists", () => {
      const hold = (title: string) =>
        server.projects.set("projects/payments-ab12", {
          name: "projects/payments-ab12",
          title,
        });
      // The create reaches the server and its answer is lost.
      const loseTheFirstAnswer = async () => {
        loseAnswerOf(mocks.createProject);
        const view = await startAtTeam();
        finishAlone();
        await failed();
        return view;
      };
      // The create never reaches the server.
      const failTheFirstCreate = async () => {
        mocks.createProject.mockRejectedValueOnce(unavailable());
        await startAtTeam();
        finishAlone();
        await failed();
      };
      const adopted = async () => {
        await landed();
        expect(mocks.createProject).toHaveBeenCalledTimes(2);
        expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledWith(
          "projects/payments-ab12"
        );
        expect(server.projects.size).toBe(1);
      };

      test("adopts the project its own lost attempt created", async () => {
        await loseTheFirstAnswer();

        tryAgain();

        await adopted();
        for (const read of [mocks.getProject, mocks.getProjectIamPolicy]) {
          expect(
            read.mock.calls[0][1].contextValues.get(
              skipErrorNotificationContextKey
            )
          ).toBe(true);
        }
      });

      test("adopts it after a reload too", async () => {
        const first = await loseTheFirstAnswer();
        first.unmount();

        render(<WorkspaceSetupPage />);
        fireEvent.click(finishButton());

        await adopted();
      });

      test("finds it on Back, and shows the project as existing from then on", async () => {
        await loseTheFirstAnswer();

        fireEvent.click(backButton());

        await waitFor(() =>
          expect(screen.getByLabelText(/^Project name/)).toBeDisabled()
        );
        expect(
          screen.getByText(
            "This project already exists. Rename it in its settings."
          )
        ).toBeInTheDocument();
        expect(screen.queryByText(ID_EXISTS)).not.toBeInTheDocument();
        fireEvent.click(continueButton());
        fireEvent.click(finishButton());

        await landed();
        expect(mocks.createProject).toHaveBeenCalledTimes(1);
        expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledWith(
          "projects/payments-ab12"
        );
      });

      test("calls it a clash on Back when the project that holds the ID is not its own", async () => {
        await failTheFirstCreate();
        hold("Billing");

        fireEvent.click(backButton());

        expect(await screen.findByText(ID_EXISTS)).toBeInTheDocument();
        expect(screen.getByLabelText(/^Project name/)).toBeEnabled();
        expect(continueButton()).toBeDisabled();
      });

      test("does not give the project it finds the name and ID typed while it looked", async () => {
        await loseTheFirstAnswer();
        const answer = holdNextCall(mocks.getProject);

        fireEvent.click(backButton());
        await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());
        await nameProject("Orders");
        answer();

        await waitFor(() =>
          expect(mocks.getProjectIamPolicy).toHaveBeenCalled()
        );
        expect(screen.getByLabelText(/^Project name/)).toBeEnabled();
        expect(screen.getByLabelText(/^Project name/)).toHaveValue("Orders");
        fireEvent.click(continueButton());
        fireEvent.click(finishButton());
        await landed();
        expect(mocks.routerReplace).toHaveBeenCalledWith({
          name: PROJECT_V1_ROUTE_DATABASES,
          params: { projectId: "orders-ab12" },
          query: {
            syncingInstance: "sample",
            [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
          },
        });
      });

      test.each([
        {
          edit: "a new name with the ID kept",
          change: async () => {
            fireEvent.click(screen.getByRole("button", { name: "Edit" }));
            type(screen.getByLabelText(/^Project name/), "Orders");
          },
        },
        {
          edit: "a new ID with the name kept",
          change: async () => editProjectId("payments-2"),
        },
      ])("drops what it finds when $edit was typed while it looked", async ({
        change,
      }) => {
        await loseTheFirstAnswer();
        const answer = holdNextCall(mocks.getProject);
        fireEvent.click(backButton());
        await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());

        await change();
        answer();

        await waitFor(() =>
          expect(mocks.getProjectIamPolicy).toHaveBeenCalled()
        );
        expect(screen.getByLabelText(/^Project name/)).toBeEnabled();
        expect(
          screen.queryByText(
            "This project already exists. Rename it in its settings."
          )
        ).not.toBeInTheDocument();
      });

      test("leaves a create it finds missing unsettled, so the retry takes the project as its own once the create lands", async () => {
        mocks.isSaaSMode = false;
        const land = delayLostWrite(mocks.createProject);
        await startAtTeam();
        finishAlone();
        await failed();

        fireEvent.click(backButton());
        await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());
        await settle();
        expect(
          screen.queryByRole("button", { name: "I'll do this later" })
        ).not.toBeInTheDocument();
        land();
        await settle();
        fireEvent.click(continueButton());
        fireEvent.click(finishButton());

        await landed();
        expect(mocks.createProject).toHaveBeenCalledTimes(2);
        expect(server.projects.size).toBe(1);
        expect(mocks.routerReplace).toHaveBeenCalledWith({
          name: PROJECT_V1_ROUTE_DATABASES,
          params: { projectId: "payments-ab12" },
          query: {
            syncingInstance: "sample",
            [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
          },
        });
      });

      test("saves nothing it finds once Finish has ended setup while it looked", async () => {
        await loseTheFirstAnswer();
        const answer = holdNextCall(mocks.getProject);
        fireEvent.click(backButton());
        await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());

        fireEvent.click(continueButton());
        fireEvent.click(finishButton());
        await landed();
        answer();
        await settle();

        expect(localStorage.getItem(DRAFT_KEY)).toBeNull();
      });

      test("leaves the ID usable on Back when the server cannot be read", async () => {
        await loseTheFirstAnswer();
        mocks.getProject.mockRejectedValueOnce(unavailable());

        fireEvent.click(backButton());

        await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());
        expect(screen.getByLabelText(/^Project name/)).toBeEnabled();
        expect(screen.queryByText(ID_EXISTS)).not.toBeInTheDocument();
        expect(continueButton()).toBeEnabled();
      });

      test("leaves the project editable on Back when the create never reached the server", async () => {
        await failTheFirstCreate();

        fireEvent.click(backButton());

        await waitFor(() => expect(mocks.getProject).toHaveBeenCalled());
        expect(screen.getByLabelText(/^Project name/)).toBeEnabled();
        expect(screen.queryByText(ID_EXISTS)).not.toBeInTheDocument();
        expect(continueButton()).toBeEnabled();
      });

      test.each([
        { what: "the project", read: mocks.getProject },
        { what: "its access policy", read: mocks.getProjectIamPolicy },
      ])("stays retryable when $what cannot be read", async ({ read }) => {
        await loseTheFirstAnswer();
        read.mockRejectedValueOnce(unavailable("read failed"));

        tryAgain();

        const alert = await failed();
        expect(alert).toHaveTextContent("Could not create Payments.");
        expect(alert).toHaveTextContent("read failed");
        expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();

        tryAgain();

        await landed();
        expect(server.projects.size).toBe(1);
      });

      test.each([
        {
          whose: "someone else has been given access to",
          bindings: [
            { role: "roles/projectOwner", members: ["user:ed@example.com"] },
            { role: "roles/projectDeveloper", members: ["user:zoe@example.com"] },
          ],
        },
        {
          whose: "someone else owns",
          bindings: [
            { role: "roles/projectOwner", members: ["user:zoe@example.com"] },
          ],
        },
        {
          whose: "this account holds without owning",
          bindings: [
            { role: "roles/projectDeveloper", members: ["user:ed@example.com"] },
          ],
        },
        {
          whose: "has a second owner",
          bindings: [
            {
              role: "roles/projectOwner",
              members: ["user:ed@example.com", "user:zoe@example.com"],
            },
          ],
        },
      ])("does not take over a project of the same ID and name that $whose", async ({
        bindings,
      }) => {
        await failTheFirstCreate();
        hold("Payments");
        server.projectPolicy = { etag: "7", bindings } as IamPolicy;

        tryAgain();

        await screen.findByText(ID_EXISTS, { selector: "li" });
        expect(continueButton()).toBeDisabled();
        expect(writes()).not.toContain("projectMembers");
        expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
        expect(server.projectPolicy.bindings).toEqual(bindings);
      });

      test("asks for another ID when the project that holds it has another name", async () => {
        await failTheFirstCreate();
        hold("Billing");

        tryAgain();

        // Once the field's own check has answered, it is the one place that
        // says so.
        await screen.findByText(ID_EXISTS, { selector: "li" });
        expect(screen.getAllByText(ID_EXISTS)).toHaveLength(1);
        expect(continueButton()).toBeDisabled();
        expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
      });

      test("asks for another ID when the ID is held outside this workspace, after a reload too, until the ID changes", async () => {
        const first = await startAtTeam();
        // Another workspace's project: the create is refused, and this
        // workspace cannot read it.
        mocks.createProject.mockRejectedValue(alreadyExists());
        finishAlone();

        expect(await failed()).toHaveTextContent(ID_EXISTS);
        expect(continueButton()).toBeDisabled();

        first.unmount();
        render(<WorkspaceSetupPage />);

        expect(await failed()).toHaveTextContent(ID_EXISTS);
        expect(continueButton()).toBeDisabled();
        expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();

        editProjectId("payments-2");

        await waitFor(() => expect(continueButton()).toBeEnabled());
        expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      });

      test("asks for another ID when a retry finds the ID held outside this workspace", async () => {
        await failTheFirstCreate();
        mocks.createProject.mockRejectedValueOnce(alreadyExists());

        tryAgain();

        expect(await failed()).toHaveTextContent(ID_EXISTS);
        expect(continueButton()).toBeDisabled();
        expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
      });

      test("asks for another ID on a first attempt, even when a project of the same name holds it", async () => {
        const first = await startAtTeam();
        hold("Payments");

        finishAlone();

        await screen.findByText(ID_EXISTS, { selector: "li" });
        expect(screen.getAllByText(ID_EXISTS)).toHaveLength(1);
        expect(continueButton()).toBeDisabled();
        expect(mocks.getProject).not.toHaveBeenCalled();

        // The refusal settled that create: a reload does not turn it into one
        // whose answer was lost.
        first.unmount();
        render(<WorkspaceSetupPage />);

        await screen.findByText(ID_EXISTS, { selector: "li" });
        expect(continueButton()).toBeDisabled();
        expect(mocks.prepareSampleProjectInstance).not.toHaveBeenCalled();
        expect(mocks.routerReplace).not.toHaveBeenCalled();
      });

    });

    test("returns to the project screen with the server's reason when it refuses the ID", async () => {
      mocks.createProject.mockRejectedValueOnce(
        new ConnectError(
          'project ID "payments-ab12" is reserved',
          Code.InvalidArgument
        )
      );
      const first = await startAtTeam();

      finishAlone();

      expect(await failed()).toHaveTextContent(
        'project ID "payments-ab12" is reserved'
      );
      expect(screen.getByLabelText(/^Project name/)).toHaveValue("Payments");
      expect(continueButton()).toBeDisabled();
      expect(mocks.routerReplace).not.toHaveBeenCalled();

      first.unmount();
      render(<WorkspaceSetupPage />);

      expect(await failed()).toHaveTextContent(
        'project ID "payments-ab12" is reserved'
      );
      expect(continueButton()).toBeDisabled();

      editProjectId("main");

      await waitFor(() => expect(continueButton()).toBeEnabled());
    });
  });
});
