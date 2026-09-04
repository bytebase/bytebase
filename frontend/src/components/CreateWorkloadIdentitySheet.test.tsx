import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
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

vi.mock("@/components/RoleSelect", () => ({
  RoleSelect: () => <div data-testid="role-select" />,
}));

vi.mock("@/components/ui/sheet", () => ({
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

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", () => ({
  getWorkloadIdentityProviderText: (provider: number) =>
    provider === 1 ? "GitLab" : "GitHub Actions",
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
  parseWorkloadIdentitySubjectPattern: (workloadIdentity: {
    workloadIdentityConfig?: { subjectPattern: string };
  }) => {
    if (
      workloadIdentity.workloadIdentityConfig?.subjectPattern ===
      "project_path:group/project:environment:production"
    ) {
      return { owner: "group", repo: "project", branch: "" };
    }
    return undefined;
  },
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

  test("places repository and branch helper text under the field title", () => {
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

    const fields = Array.from(
      container.querySelectorAll('[data-slot="form-field"]')
    );
    const repositoryField = fields.find((field) =>
      field.textContent?.includes("settings.members.workload-identity-repo")
    );
    const branchField = fields.find((field) =>
      field.textContent?.includes("settings.members.workload-identity-branch")
    );

    for (const field of [repositoryField, branchField]) {
      expect(field).toBeTruthy();
      const header = field?.querySelector('[data-slot="form-field-header"]');
      const input = field?.querySelector("input");

      expect(header).toBeTruthy();
      expect(header?.textContent).toContain("hint");
      expect(
        header && input
          ? Node.DOCUMENT_POSITION_FOLLOWING &
              header.compareDocumentPosition(input)
          : 0
      ).toBeTruthy();
    }

    act(() => {
      root.unmount();
    });
  });

  test("omits the blank preset audience when creating a GitHub identity", async () => {
    const created = create(WorkloadIdentitySchema, {
      name: "workloadIdentities/deploy@workload.bytebase.com",
      email: "deploy@workload.bytebase.com",
      title: "deploy",
    });
    mocks.store.createWorkloadIdentity.mockResolvedValue(created);

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <CreateWorkloadIdentitySheet
          open
          onClose={() => undefined}
          onCreated={() => undefined}
        />
      );
    });

    const fields = Array.from(
      container.querySelectorAll('[data-slot="form-field"]')
    );
    const fieldInput = (label: string) =>
      fields
        .find((field) => field.textContent?.includes(label))
        ?.querySelector("input");
    const setInputValue = (
      input: HTMLInputElement | null | undefined,
      value: string
    ) => {
      if (!input) return;
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    };

    await act(async () => {
      setInputValue(fieldInput("common.email"), "deploy");
      setInputValue(
        fieldInput("settings.members.workload-identity-owner"),
        "bytebase"
      );
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.create"
    );
    expect(createButton?.disabled).toBe(false);
    await act(async () => {
      createButton?.click();
    });

    expect(mocks.store.createWorkloadIdentity).toHaveBeenCalledOnce();
    const [, workloadIdentity] =
      mocks.store.createWorkloadIdentity.mock.calls[0];
    expect(
      workloadIdentity.workloadIdentityConfig?.allowedAudiences
    ).toEqual([]);

    act(() => {
      root.unmount();
    });
  });

  test("preserves a custom subject pattern when updating another field", async () => {
    const subjectPattern =
      "project_path:group/project:environment:production";
    const workloadIdentity = create(WorkloadIdentitySchema, {
      name: "workloadIdentities/atlantis@workload.bytebase.com",
      email: "atlantis@workload.bytebase.com",
      title: "Atlantis",
      workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
        providerType: WorkloadIdentityConfig_ProviderType.GITLAB,
        issuerUrl: "https://nomad.example.com",
        allowedAudiences: ["bytebase"],
        subjectPattern,
      }),
    });
    mocks.store.updateWorkloadIdentity.mockResolvedValue(workloadIdentity);

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <CreateWorkloadIdentitySheet
          open
          workloadIdentity={workloadIdentity}
          onClose={() => undefined}
          onCreated={() => undefined}
        />
      );
    });

    const advancedButton = Array.from(container.querySelectorAll("button")).find(
      (button) =>
        button.textContent?.includes(
          "settings.members.workload-identity-advanced"
        )
    );
    expect(advancedButton).toBeTruthy();
    await act(async () => {
      advancedButton?.click();
    });

    const subjectField = Array.from(
      container.querySelectorAll('[data-slot="form-field"]')
    ).find((field) =>
      field.textContent?.includes("settings.members.workload-identity-subject")
    );
    const subjectInput = subjectField?.querySelector("input");
    expect(subjectInput?.value).toBe(subjectPattern);

    const titleInput = container.querySelector("input");
    expect(titleInput).toBeTruthy();
    await act(async () => {
      if (titleInput) {
        Object.getOwnPropertyDescriptor(
          HTMLInputElement.prototype,
          "value"
        )?.set?.call(titleInput, "Updated Atlantis");
        titleInput.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });

    const updateButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.update"
    );
    expect(updateButton).toBeTruthy();
    await act(async () => {
      updateButton?.click();
    });

    expect(mocks.store.updateWorkloadIdentity).toHaveBeenCalledOnce();
    const [updatedWorkloadIdentity] =
      mocks.store.updateWorkloadIdentity.mock.calls[0];
    expect(
      updatedWorkloadIdentity.workloadIdentityConfig?.subjectPattern
    ).toBe(subjectPattern);

    act(() => {
      root.unmount();
    });
  });

  test("shows and submits generic OIDC fields without preset controls", async () => {
    const workloadIdentity = create(WorkloadIdentitySchema, {
      name: "workloadIdentities/atlantis@workload.bytebase.com",
      email: "atlantis@workload.bytebase.com",
      title: "Atlantis",
      workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
        providerType: WorkloadIdentityConfig_ProviderType.OIDC,
        issuerUrl: "https://nomad.example.com",
        jwksUrl: "https://nomad-verifier.example.com/jwks.json",
        allowedAudiences: ["bytebase", "terraform"],
        subjectPattern: "nomad_job:atlantis:namespace:production",
      }),
    });
    mocks.store.updateWorkloadIdentity.mockResolvedValue(workloadIdentity);

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <CreateWorkloadIdentitySheet
          open
          workloadIdentity={workloadIdentity}
          onClose={() => undefined}
          onCreated={() => undefined}
        />
      );
    });

    const fields = Array.from(
      container.querySelectorAll('[data-slot="form-field"]')
    );
    const fieldInput = (label: string) =>
      fields
        .find((field) => field.textContent?.includes(label))
        ?.querySelector("input");

    expect(
      fields.some((field) =>
        field.textContent?.includes("settings.members.workload-identity-owner")
      )
    ).toBe(false);
    expect(
      fields.some((field) =>
        field.textContent?.includes("settings.members.workload-identity-repo")
      )
    ).toBe(false);
    expect(fieldInput("settings.members.workload-identity-issuer")?.value).toBe(
      "https://nomad.example.com"
    );
    expect(fieldInput("settings.members.workload-identity-jwks-url")?.value).toBe(
      "https://nomad-verifier.example.com/jwks.json"
    );
    const audienceField = fields.find((field) =>
      field.textContent?.includes(
        "settings.members.workload-identity-audience"
      )
    );
    expect(
      Array.from(audienceField?.querySelectorAll("input") ?? []).map(
        (input) => input.value
      )
    ).toEqual(["bytebase", "terraform"]);
    expect(fieldInput("settings.members.workload-identity-subject")?.value).toBe(
      "nomad_job:atlantis:namespace:production"
    );

    const titleInput = container.querySelector("input");
    expect(titleInput).toBeTruthy();
    await act(async () => {
      if (titleInput) {
        Object.getOwnPropertyDescriptor(
          HTMLInputElement.prototype,
          "value"
        )?.set?.call(titleInput, "Updated Atlantis");
        titleInput.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });

    const updateButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.update"
    );
    expect(updateButton).toBeTruthy();
    await act(async () => {
      updateButton?.click();
    });

    expect(mocks.store.updateWorkloadIdentity).toHaveBeenCalledOnce();
    const [updatedWorkloadIdentity, updateMask] =
      mocks.store.updateWorkloadIdentity.mock.calls[0];
    expect(updatedWorkloadIdentity.workloadIdentityConfig).toMatchObject({
      providerType: WorkloadIdentityConfig_ProviderType.OIDC,
      issuerUrl: "https://nomad.example.com",
      jwksUrl: "https://nomad-verifier.example.com/jwks.json",
      allowedAudiences: ["bytebase", "terraform"],
      subjectPattern: "nomad_job:atlantis:namespace:production",
    });
    expect(updateMask.paths).toEqual(["title"]);

    act(() => {
      root.unmount();
    });
  });

  test("adds and removes generic OIDC audiences", async () => {
    const workloadIdentity = create(WorkloadIdentitySchema, {
      name: "workloadIdentities/atlantis@workload.bytebase.com",
      email: "atlantis@workload.bytebase.com",
      title: "Atlantis",
      workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
        providerType: WorkloadIdentityConfig_ProviderType.OIDC,
        issuerUrl: "https://nomad.example.com",
        allowedAudiences: ["bytebase", "terraform"],
        subjectPattern: "nomad_job:atlantis:namespace:production",
      }),
    });
    mocks.store.updateWorkloadIdentity.mockResolvedValue(workloadIdentity);

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <CreateWorkloadIdentitySheet
          open
          workloadIdentity={workloadIdentity}
          onClose={() => undefined}
          onCreated={() => undefined}
        />
      );
    });

    const audienceField = Array.from(
      container.querySelectorAll('[data-slot="form-field"]')
    ).find((field) =>
      field.textContent?.includes(
        "settings.members.workload-identity-audience"
      )
    );
    const addButton = audienceField?.querySelector<HTMLButtonElement>(
      'button[aria-label="common.add"]'
    );
    expect(addButton).toBeTruthy();

    await act(async () => {
      addButton?.click();
    });

    let audienceInputs = Array.from(
      audienceField?.querySelectorAll("input") ?? []
    );
    expect(audienceInputs.map((input) => input.value)).toEqual([
      "bytebase",
      "terraform",
      "",
    ]);

    const updateButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.update"
    );
    expect(updateButton?.disabled).toBe(true);

    await act(async () => {
      const newAudienceInput = audienceInputs[2];
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(newAudienceInput, "deployment");
      newAudienceInput.dispatchEvent(new Event("input", { bubbles: true }));
    });

    const removeButtons = audienceField?.querySelectorAll<HTMLButtonElement>(
      'button[aria-label="common.remove"]'
    );
    expect(removeButtons).toHaveLength(3);
    await act(async () => {
      removeButtons?.[1]?.click();
    });

    audienceInputs = Array.from(
      audienceField?.querySelectorAll("input") ?? []
    );
    expect(audienceInputs.map((input) => input.value)).toEqual([
      "bytebase",
      "deployment",
    ]);
    expect(updateButton?.disabled).toBe(false);

    await act(async () => {
      updateButton?.click();
    });

    expect(mocks.store.updateWorkloadIdentity).toHaveBeenCalledOnce();
    const [updatedWorkloadIdentity, updateMask] =
      mocks.store.updateWorkloadIdentity.mock.calls[0];
    expect(
      updatedWorkloadIdentity.workloadIdentityConfig?.allowedAudiences
    ).toEqual(["bytebase", "deployment"]);
    expect(updateMask.paths).toEqual(["workload_identity_config"]);

    act(() => {
      root.unmount();
    });
  });
});
