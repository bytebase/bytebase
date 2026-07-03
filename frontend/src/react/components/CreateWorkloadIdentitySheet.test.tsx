import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { CreateWorkloadIdentitySheet } from "./CreateWorkloadIdentitySheet";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const store = {
    projectsByName: {},
    getProjectIamPolicy: vi.fn(),
    updateProjectIamPolicy: vi.fn(),
    patchWorkspaceIamPolicy: vi.fn(),
    createWorkloadIdentity: vi.fn(),
    updateWorkloadIdentity: vi.fn(),
    workspaceResourceName: () => "workspaces/default",
  };
  const useAppStore = vi.fn((selector: (state: typeof store) => unknown) =>
    selector(store)
  );
  return {
    store,
    useAppStore: Object.assign(useAppStore, {
      getState: () => store,
    }),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/components/RoleSelect", () => ({
  RoleSelect: () => <div data-testid="role-select" />,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: React.ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  SheetBody: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetFooter: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetTitle: ({ children }: { children: React.ReactNode }) => (
    <h2>{children}</h2>
  ),
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", () => ({
  getWorkloadIdentityProviderText: (provider: number) =>
    provider === 1 ? "GitLab" : "GitHub Actions",
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
  parseWorkloadIdentitySubjectPattern: () => undefined,
}));

describe("CreateWorkloadIdentitySheet", () => {
  afterEach(() => {
    document.body.innerHTML = "";
    vi.clearAllMocks();
  });

  test("places role selection directly after email in create mode", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <CreateWorkloadIdentitySheet
          open
          onClose={() => undefined}
          onCreated={() => undefined}
        />
      );
    });

    const text = container.textContent ?? "";
    const emailIndex = text.indexOf("common.email");
    const rolesIndex = text.indexOf("settings.members.table.roles");
    const platformIndex = text.indexOf(
      "settings.members.workload-identity-platform"
    );

    expect(emailIndex).toBeGreaterThanOrEqual(0);
    expect(rolesIndex).toBeGreaterThan(emailIndex);
    expect(platformIndex).toBeGreaterThan(rolesIndex);

    act(() => {
      root.unmount();
    });
  });
});
