import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { WorkloadIdentityDialog } from "./WorkloadIdentityDialog";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const store = {
    getProjectIamPolicy: vi.fn(),
    updateProjectIamPolicy: vi.fn(),
    patchWorkspaceIamPolicy: vi.fn(),
    createWorkloadIdentity: vi.fn(),
    updateWorkloadIdentity: vi.fn(),
    fetchWorkloadIdentity: vi.fn(),
    workspaceResourceName: () => "workspaces/default",
  };
  const useAppStore = vi.fn((selector: (state: typeof store) => unknown) =>
    selector(store)
  );
  return {
    store,
    canSetRoles: true,
    serverState: {
      externalUrl: "https://bytebase.example.com",
      needConfigureExternalUrl: false,
    },
    useAppStore: Object.assign(useAppStore, { getState: () => store }),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
}));

// Two buttons stand in for the role picker: pick another role, or clear it.
vi.mock("@/components/RoleSelect", () => ({
  RoleSelect: ({
    value,
    onChange,
  }: {
    value: string[];
    onChange: (roles: string[]) => void;
  }) => (
    <div data-testid="role-select" data-value={value.join(",")}>
      <button type="button" onClick={() => onChange(["roles/projectDeveloper"])}>
        pick-developer
      </button>
      <button type="button" onClick={() => onChange([])}>
        clear-role
      </button>
    </div>
  ),
}));

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({ children, open }: { children: React.ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2>{children}</h2>
  ),
  DialogClose: ({ children }: { children: React.ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children }: { children: React.ReactNode }) => (
    <a href="#settings">{children}</a>
  ),
}));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => mocks.serverState,
}));

vi.mock("@/utils/role", () => ({
  displayRoleTitle: (role: string) =>
    role === "roles/gitopsServiceAgent" ? "GitOps Service Agent" : role,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/stores/app", () => ({ useAppStore: mocks.useAppStore }));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/utils", async () => ({
  ...(await vi.importActual<typeof import("@/utils/workloadIdentity")>(
    "@/utils/workloadIdentity"
  )),
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: (permission: string) =>
    permission === "bb.workspaces.setIamPolicy" ? mocks.canSetRoles : true,
}));

let root: Root;
let container: HTMLElement;

const render = async (props: {
  workloadIdentity?: WorkloadIdentity;
  onClose?: () => void;
}) => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  await act(async () => {
    root.render(
      <WorkloadIdentityDialog
        open
        onClose={() => undefined}
        onCreated={() => undefined}
        {...props}
      />
    );
  });
};

const button = (text: string) =>
  Array.from(container.querySelectorAll("button")).find(
    (b) => b.textContent === text
  );

const click = async (text: string) => {
  const target = button(text);
  if (!target) throw new Error(`button ${text} not found`);
  await act(async () => target.click());
};

// The input a field's label points at.
const field = (labelKey: string) => {
  const label = Array.from(container.querySelectorAll("label[for]")).find(
    (l) => l.textContent?.startsWith(labelKey)
  );
  const input = label
    ? (document.getElementById(label.getAttribute("for") ?? "") as
        | HTMLInputElement
        | null)
    : null;
  if (!input) throw new Error(`field ${labelKey} not found`);
  return input;
};

const type = async (labelKey: string, value: string) => {
  const input = field(labelKey);
  await act(async () => {
    Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    )?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
};

const chooseProvider = async (index: number) => {
  const radios = container.querySelectorAll<HTMLElement>('[role="radio"]');
  await act(async () => radios[index].click());
};

const text = () => container.textContent ?? "";

const goToAccount = async (owner: string, ownerId = "") => {
  await click("workload-identity.continue");
  await type("workload-identity.repositories.owner", owner);
  await type("workload-identity.repositories.repository", "deploy");
  if (ownerId) await type("workload-identity.repositories.owner-id", ownerId);
  await click("workload-identity.continue");
};

const storedIdentity = (config: {
  providerType?: WorkloadIdentityConfig_ProviderType;
  subjectPattern?: string;
  allowedAudiences?: string[];
  jwksUrl?: string;
}) =>
  create(WorkloadIdentitySchema, {
    name: "workloadIdentities/deploy@workload.bytebase.com",
    email: "deploy@workload.bytebase.com",
    title: "deploy",
    workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
      providerType: WorkloadIdentityConfig_ProviderType.GITHUB,
      issuerUrl: "https://token.actions.githubusercontent.com",
      allowedAudiences: ["bytebase"],
      subjectPattern: "repo:acme-corp/deploy:*",
      ...config,
    }),
  });

describe("WorkloadIdentityDialog", () => {
  beforeEach(() => {
    mocks.canSetRoles = true;
    mocks.serverState = {
      externalUrl: "https://bytebase.example.com",
      needConfigureExternalUrl: false,
    };
    mocks.store.fetchWorkloadIdentity.mockResolvedValue(undefined);
    mocks.store.createWorkloadIdentity.mockImplementation(
      async (id: string) =>
        create(WorkloadIdentitySchema, {
          name: `workloadIdentities/${id}@workload.bytebase.com`,
          email: `${id}@workload.bytebase.com`,
        })
    );
    mocks.store.updateWorkloadIdentity.mockImplementation(async (wi) => wi);
  });

  afterEach(() => {
    act(() => root.unmount());
    document.body.innerHTML = "";
    vi.clearAllMocks();
  });

  test("creates a GitHub identity with the preset audience and the role", async () => {
    await render({});
    await goToAccount("acme-corp");

    expect(field("common.email").value).toBe("deploy-ci");
    await click("common.create");

    expect(mocks.store.createWorkloadIdentity).toHaveBeenCalledOnce();
    const [id, wi] = mocks.store.createWorkloadIdentity.mock.calls[0];
    expect(id).toBe("deploy-ci");
    expect(wi.workloadIdentityConfig.subjectPattern).toBe(
      "repo:acme-corp/deploy:*"
    );
    expect(wi.workloadIdentityConfig.allowedAudiences).toEqual(["bytebase"]);
    expect(mocks.store.patchWorkspaceIamPolicy).toHaveBeenCalledWith([
      {
        member: "workloadIdentity:deploy-ci@workload.bytebase.com",
        roles: ["roles/gitopsServiceAgent"],
      },
    ]);
    expect(text()).toContain("workload-identity.done.title");
    expect(text()).toContain(".github/workflows/bytebase-wif-test.yml");
    const header = container.querySelector('[data-slot="code-block-header"]');
    expect(header?.textContent).toContain(
      ".github/workflows/bytebase-wif-test.yml"
    );
    expect(header?.querySelector('[aria-label="common.copy"]')).toBeTruthy();
  });

  test("uses the External URL in the test file", async () => {
    await render({});
    await goToAccount("acme-corp");
    await click("common.create");

    const code = container.querySelector("pre")?.textContent ?? "";
    expect(code).toContain("https://bytebase.example.com/v1/auth:exchangeToken");
    expect(text()).not.toContain(
      "workload-identity.done.external-url-placeholder"
    );
  });

  test("shows the test file with a placeholder while the External URL is unset", async () => {
    mocks.serverState = { externalUrl: "", needConfigureExternalUrl: true };
    await render({});
    await goToAccount("acme-corp");
    await click("common.create");

    const code = container.querySelector("pre")?.textContent ?? "";
    expect(code).toContain("https://YOUR-BYTEBASE-URL/v1/auth:exchangeToken");
    expect(text()).toContain("workload-identity.done.external-url-placeholder");
    expect(text()).toContain("common.configure-now");
  });

  test("opens on GitHub Actions with its issuer and audience", async () => {
    await render({});
    const radios = container.querySelectorAll('[role="radio"]');
    expect(radios[0].getAttribute("aria-checked")).toBe("true");
    expect(field("workload-identity.token-source.issuer").value).toBe(
      "https://token.actions.githubusercontent.com"
    );
    expect(text()).toContain("workload-identity.token-source.github-note");
    expect(text()).toContain("bytebase");
  });

  test("prefills the email from the owner when Repository is blank", async () => {
    await render({});
    await click("workload-identity.continue");
    await type("workload-identity.repositories.owner", "acme-corp");
    await click("workload-identity.continue");
    expect(field("common.email").value).toBe("acme-corp-ci");
  });

  test("opens every step scrolled to the top", async () => {
    await render({});
    const body = container.querySelector<HTMLElement>(
      '[data-slot="workload-identity-step-body"]'
    );
    if (!body) throw new Error("step body not found");

    body.scrollTop = 200;
    await click("workload-identity.continue");
    expect(body.scrollTop).toBe(0);

    body.scrollTop = 200;
    await click("common.back");
    expect(body.scrollTop).toBe(0);
  });

  test("always shows the owner ID box for GitHub, styled by Owner ID", async () => {
    await render({});
    await click("workload-identity.continue");

    const box = () =>
      Array.from(container.querySelectorAll<HTMLElement>('[role="alert"]')).find(
        (alert) =>
          alert.textContent?.includes(
            "workload-identity.repositories.owner-id-note-title"
          )
      );
    const expectBox = (style: "warning" | "info") => {
      const alert = box();
      expect(alert).toBeTruthy();
      expect(alert?.className).toContain(`bg-${style}/10`);
      const link = Array.from(alert?.querySelectorAll("a") ?? []).find(
        (a) =>
          a.textContent === "workload-identity.repositories.owner-id-note-link"
      );
      expect(link?.getAttribute("href")).toBe(
        "https://github.blog/changelog/2026-04-23-immutable-subject-claims-for-github-actions-oidc-tokens/"
      );
      expect(link?.getAttribute("target")).toBe("_blank");
      expect(link?.getAttribute("rel")).toBe("noopener noreferrer");
    };

    expectBox("warning");
    await type("workload-identity.repositories.owner", "acme-corp");
    expectBox("warning");
    await type("workload-identity.repositories.owner-id", "111");
    expectBox("info");
  });

  test("shows no owner ID box for GitLab", async () => {
    await render({});
    await chooseProvider(1);
    await click("workload-identity.continue");
    expect(text()).not.toContain(
      "workload-identity.repositories.owner-id-note-title"
    );
  });

  test("puts the owner ID in the pattern", async () => {
    await render({});
    await goToAccount("acme-corp", "111");
    await click("common.create");

    const [, wi] = mocks.store.createWorkloadIdentity.mock.calls[0];
    expect(wi.workloadIdentityConfig.subjectPattern).toBe(
      "repo:acme-corp@111/deploy:*"
    );
  });

  test("shows the default role read-only until Edit", async () => {
    await render({});
    await goToAccount("acme-corp");

    expect(field("common.role.self").value).toBe("GitOps Service Agent");
    expect(field("common.role.self").readOnly).toBe(true);
    expect(container.querySelector('[data-testid="role-select"]')).toBeNull();

    await click("common.edit");
    expect(
      container
        .querySelector('[data-testid="role-select"]')
        ?.getAttribute("data-value")
    ).toBe("roles/gitopsServiceAgent");
  });

  test("grants the role picked after Edit", async () => {
    await render({});
    await goToAccount("acme-corp");
    await click("common.edit");
    await click("pick-developer");
    await click("common.create");

    expect(mocks.store.patchWorkspaceIamPolicy).toHaveBeenCalledWith([
      {
        member: "workloadIdentity:deploy-ci@workload.bytebase.com",
        roles: ["roles/projectDeveloper"],
      },
    ]);
  });

  test("restores the default role on Reset", async () => {
    await render({});
    await goToAccount("acme-corp");
    await click("common.edit");
    await click("pick-developer");
    await click("common.reset");

    expect(container.querySelector('[data-testid="role-select"]')).toBeNull();
    expect(field("common.role.self").value).toBe("GitOps Service Agent");
  });

  test("requires a role after it is cleared", async () => {
    await render({});
    await goToAccount("acme-corp");
    await click("common.edit");
    await click("clear-role");
    await click("common.create");

    expect(mocks.store.createWorkloadIdentity).not.toHaveBeenCalled();
    expect(text()).toContain("workload-identity.account.role-required");
  });

  test("creates without an IAM call when the user cannot grant roles", async () => {
    mocks.canSetRoles = false;
    await render({});
    await goToAccount("acme-corp");

    expect(container.querySelector('[data-testid="role-select"]')).toBeNull();
    expect(button("common.edit")).toBeUndefined();
    expect(text()).toContain("workload-identity.account.no-permission-workspace");
    await click("common.create");

    expect(mocks.store.createWorkloadIdentity).toHaveBeenCalledOnce();
    expect(mocks.store.patchWorkspaceIamPolicy).not.toHaveBeenCalled();
  });

  test("shows on Done that the role was not granted", async () => {
    mocks.store.patchWorkspaceIamPolicy.mockRejectedValue(
      new ConnectError("permission denied", Code.PermissionDenied)
    );
    await render({});
    await goToAccount("acme-corp");
    await click("common.create");

    expect(text()).toContain("workload-identity.done.title");
    expect(text()).toContain("workload-identity.done.role-not-granted");
  });

  test("blocks an email another identity uses", async () => {
    vi.useFakeTimers();
    try {
      mocks.store.fetchWorkloadIdentity.mockResolvedValue(storedIdentity({}));
      await render({});
      await goToAccount("acme-corp");
      await act(async () => {
        await vi.advanceTimersByTimeAsync(400);
      });

      expect(mocks.store.fetchWorkloadIdentity).toHaveBeenCalledWith(
        "deploy-ci@workload.bytebase.com",
        true
      );
      expect(text()).toContain("workload-identity.account.email-used");
      await click("common.create");
      expect(mocks.store.createWorkloadIdentity).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  test("sends the JWKS URL of another OIDC issuer", async () => {
    await render({});
    await chooseProvider(2);
    await type("workload-identity.token-source.issuer", "https://sts.example.com");
    await act(async () => {
      const input = field("workload-identity.token-source.audiences");
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(input, "sts");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", bubbles: true })
      );
    });
    await type(
      "workload-identity.token-source.jwks-url",
      "https://sts.example.com/keys"
    );
    await click("workload-identity.continue");
    await type(
      "workload-identity.repositories.subject-pattern",
      "system:serviceaccount:prod:*"
    );
    await click("workload-identity.continue");
    await type("common.email", "sts");
    await click("common.create");

    const [, wi] = mocks.store.createWorkloadIdentity.mock.calls[0];
    expect(wi.workloadIdentityConfig).toMatchObject({
      providerType: WorkloadIdentityConfig_ProviderType.OIDC,
      issuerUrl: "https://sts.example.com",
      jwksUrl: "https://sts.example.com/keys",
      allowedAudiences: ["bytebase", "sts"],
      subjectPattern: "system:serviceaccount:prod:*",
    });
  });

  test("keeps the stored JWKS URL and the pattern's remainder when only the owner changes", async () => {
    await render({
      workloadIdentity: storedIdentity({
        subjectPattern: "repo:acme-corp/deploy:environment:prod",
        jwksUrl: "https://keys.example.com/jwks",
      }),
    });
    await click("workload-identity.continue");
    expect(text()).toContain("workload-identity.repositories.environment-warning");
    await type("workload-identity.repositories.owner", "acme-labs");
    await click("workload-identity.continue");
    await click("common.update");

    const [wi, mask] = mocks.store.updateWorkloadIdentity.mock.calls[0];
    expect(wi.workloadIdentityConfig.subjectPattern).toBe(
      "repo:acme-labs/deploy:environment:prod"
    );
    expect(wi.workloadIdentityConfig.jwksUrl).toBe(
      "https://keys.example.com/jwks"
    );
    expect(mask.paths).toEqual(["workload_identity_config"]);
  });

  test("ends an edit on Done with the test file and no email row", async () => {
    const onClose = vi.fn();
    await render({ workloadIdentity: storedIdentity({}), onClose });
    await click("workload-identity.continue");
    await type("workload-identity.repositories.owner", "acme-labs");
    await click("workload-identity.continue");
    await click("common.update");

    expect(mocks.store.updateWorkloadIdentity).toHaveBeenCalledOnce();
    expect(onClose).not.toHaveBeenCalled();
    expect(text()).toContain("workload-identity.done.title-updated");
    expect(text()).toContain("workload-identity.done.updated-workspace");
    expect(text()).not.toContain("workload-identity.done.email-hint");
    const header = container.querySelector('[data-slot="code-block-header"]');
    expect(header?.textContent).toContain(
      ".github/workflows/bytebase-wif-test.yml"
    );
    expect(header?.querySelector('[aria-label="common.copy"]')).toBeTruthy();

    await click("common.done");
    expect(onClose).toHaveBeenCalledOnce();
  });

  test("keeps Update disabled until something changes", async () => {
    await render({ workloadIdentity: storedIdentity({}) });
    await click("workload-identity.continue");
    await click("workload-identity.continue");
    expect(button("common.update")?.disabled).toBe(true);
    await type("workload-identity.account.display-name", "Deploy bot");
    expect(button("common.update")?.disabled).toBe(false);
  });

  test("requires a provider for an identity stored without one", async () => {
    await render({
      workloadIdentity: storedIdentity({
        providerType: WorkloadIdentityConfig_ProviderType.PROVIDER_TYPE_UNSPECIFIED,
      }),
    });
    const radios = container.querySelectorAll('[role="radio"]');
    expect(
      Array.from(radios).some((r) => r.getAttribute("aria-checked") === "true")
    ).toBe(false);
    await click("workload-identity.continue");
    expect(text()).toContain("workload-identity.token-source.provider-required");

    await chooseProvider(0);
    await click("workload-identity.continue");
    // The stored pattern is kept, not reset by picking the provider.
    expect(field("workload-identity.repositories.subject-pattern").value).toBe(
      "repo:acme-corp/deploy:*"
    );
  });

  test("leaves an identity's empty audiences empty and required", async () => {
    await render({ workloadIdentity: storedIdentity({ allowedAudiences: [] }) });
    await click("workload-identity.continue");
    expect(text()).toContain("workload-identity.token-source.audiences-required");
  });

  test("moves to the step whose field the server rejected", async () => {
    mocks.store.createWorkloadIdentity.mockRejectedValue(
      new ConnectError(
        'subject pattern "repo:*" matches every subject from this issuer',
        Code.InvalidArgument
      )
    );
    await render({});
    await goToAccount("acme-corp");
    await click("common.create");

    expect(field("workload-identity.repositories.subject-pattern")).toBeTruthy();
    expect(text()).toContain("matches every subject from this issuer");
  });
});
