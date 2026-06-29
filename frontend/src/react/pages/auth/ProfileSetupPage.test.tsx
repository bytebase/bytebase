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
  routerReplace: vi.fn(),
  pushNotification: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => false),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
  useWorkspace: () => mocks.workspace,
  useWorkspacePermission: () => mocks.canUpdateWorkspace,
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

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/react/components/UserAvatar", () => ({
  UserAvatar: () => null,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
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
  mocks.workspacePolicy = {
    bindings: [
      {
        role: "roles/workspaceAdmin",
        members: ["users/ed@example.com"],
      },
    ],
  } as IamPolicy;
  ({ ProfileSetupPage } = await import("./ProfileSetupPage"));
});

describe("ProfileSetupPage", () => {
  test("shows workspace name when the sole member can update the workspace", () => {
    const page = renderIntoContainer(<ProfileSetupPage />);

    page.render();

    expect(page.container.textContent).toContain(
      "settings.profile.workspace-name"
    );
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
});
