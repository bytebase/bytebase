import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
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

// The real parser and resolver, so the tests see exactly what the form does
// with a stored subject. Only the permission checks are stubbed.
vi.mock("@/utils", async () => ({
  ...(await vi.importActual<typeof import("@/utils/workloadIdentity")>(
    "@/utils/workloadIdentity"
  )),
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
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

  // Both bugs below reproduce on main. #21309 skips the first run of the
  // fields-to-pattern effect, which covers opening the sheet; it does not
  // cover a subject typed after any derived field is edited.
  const editableIdentity = (
    providerType: WorkloadIdentityConfig_ProviderType,
    subjectPattern: string
  ) =>
    create(WorkloadIdentitySchema, {
      name: "workloadIdentities/ci@workload.bytebase.com",
      email: "ci@workload.bytebase.com",
      title: "CI deploy",
      workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
        providerType,
        issuerUrl: "https://token.actions.githubusercontent.com",
        allowedAudiences: ["bytebase"],
        subjectPattern,
      }),
    });

  const renderSheet = (workloadIdentity?: WorkloadIdentity) => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() => {
      root.render(
        <CreateWorkloadIdentitySheet
          open
          workloadIdentity={workloadIdentity}
          onClose={() => undefined}
          onCreated={() => undefined}
          onUpdated={() => undefined}
        />
      );
    });
    return { container, root };
  };

  const inputFor = (container: HTMLElement, label: string) =>
    Array.from(container.querySelectorAll('[data-slot="form-field"]'))
      .find((field) => field.textContent?.includes(label))
      ?.querySelector("input") as HTMLInputElement;

  const type = (input: HTMLInputElement, value: string) =>
    act(() => {
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });

  const openAdvanced = (container: HTMLElement) =>
    act(() => {
      Array.from(container.querySelectorAll("button"))
        .find((button) =>
          button.textContent?.includes(
            "settings.members.workload-identity-advanced"
          )
        )
        ?.click();
    });

  test("keeps a subject pattern typed after a field edit", async () => {
    const identity = editableIdentity(
      WorkloadIdentityConfig_ProviderType.GITHUB,
      "repo:acme-corp/deploy:ref:refs/heads/main"
    );
    mocks.store.updateWorkloadIdentity.mockResolvedValue(identity);
    const { container, root } = renderSheet(identity);

    type(inputFor(container, "settings.members.workload-identity-owner"), "acme-two");
    openAdvanced(container);
    const pinned = "repo:acme-two/deploy:environment:production";
    type(inputFor(container, "settings.members.workload-identity-subject"), pinned);

    expect(
      inputFor(container, "settings.members.workload-identity-subject").value
    ).toBe(pinned);

    const update = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.update"
    );
    await act(async () => {
      update?.click();
    });
    expect(
      mocks.store.updateWorkloadIdentity.mock.calls[0]?.[0]
        ?.workloadIdentityConfig?.subjectPattern
    ).toBe(pinned);

    act(() => root.unmount());
  });

  test("opens an identity that never declared a provider", async () => {
    const identity = editableIdentity(
      WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED,
      "repo:acme-corp/deploy:ref:refs/heads/main"
    );
    mocks.store.updateWorkloadIdentity.mockResolvedValue(identity);
    const { container, root } = renderSheet(identity);

    // The subject names the provider, so the fields parse and the form is
    // usable; reading the stored enum leaves owner empty and Update dead.
    expect(
      inputFor(container, "settings.members.workload-identity-owner").value
    ).toBe("acme-corp");

    type(inputFor(container, "common.name"), "CI deploy renamed");
    const update = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.update"
    );
    expect(update?.hasAttribute("disabled")).toBe(false);
    await act(async () => {
      update?.click();
    });

    const sent =
      mocks.store.updateWorkloadIdentity.mock.calls[0]?.[0]
        ?.workloadIdentityConfig;
    expect(sent?.subjectPattern).toBe("repo:acme-corp/deploy:ref:refs/heads/main");
    // provider_type is required on write now, so the save must name one.
    expect(sent?.providerType).toBe(WorkloadIdentityConfig_ProviderType.GITHUB);

    act(() => root.unmount());
  });

  test("recomputes the subject pattern from every derived control", () => {
    const { container, root } = renderSheet(
      editableIdentity(
        WorkloadIdentityConfig_ProviderType.GITHUB,
        "repo:acme-corp/deploy:ref:refs/heads/main"
      )
    );
    openAdvanced(container);
    const subject = () =>
      inputFor(container, "settings.members.workload-identity-subject").value;

    type(inputFor(container, "settings.members.workload-identity-owner"), "acme-two");
    expect(subject()).toBe("repo:acme-two/deploy:ref:refs/heads/main");

    type(inputFor(container, "settings.members.workload-identity-repo"), "release");
    expect(subject()).toBe("repo:acme-two/release:ref:refs/heads/main");

    type(inputFor(container, "settings.members.workload-identity-branch"), "dev");
    expect(subject()).toBe("repo:acme-two/release:ref:refs/heads/dev");

    act(() => root.unmount());
  });
});
