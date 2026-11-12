import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import githubIcon from "@/assets/git/github.svg";
import gitlabIcon from "@/assets/git/gitlab.svg";
import { BindingSchema, IamPolicySchema } from "@/types/proto-es/v1/iam_policy_pb";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { DatabaseGroupSchema } from "@/types/proto-es/v1/database_group_service_pb";
import {
  type WorkloadIdentity,
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { FILE_PREVIEW_LINES } from "./gitops-setup/WorkflowFilesStep";
import { ProjectGitOpsPage } from "./ProjectGitOpsPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const identityOf = (
  id: string,
  providerType: WorkloadIdentityConfig_ProviderType,
  subjectPattern: string
): WorkloadIdentity => {
  const email = `${id}@shop.workload.bytebase.com`;
  return create(WorkloadIdentitySchema, {
    name: `workloadIdentities/${email}`,
    email,
    title: id,
    workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
      providerType,
      subjectPattern,
    }),
  });
};

const DEPLOY_CI = identityOf(
  "deploy-ci",
  WorkloadIdentityConfig_ProviderType.GITHUB,
  "repo:acme-corp/deploy:ref:refs/heads/main"
);
const ORG_CI = identityOf(
  "org-ci",
  WorkloadIdentityConfig_ProviderType.GITHUB,
  "repo:acme-corp/*"
);
const OTHER_CI = identityOf(
  "other-ci",
  WorkloadIdentityConfig_ProviderType.OIDC,
  "system:serviceaccount:ci"
);
const NEW_CI = identityOf(
  "new-ci",
  WorkloadIdentityConfig_ProviderType.GITLAB,
  "project_path:acme-corp/deploy:ref_type:branch:ref:main"
);

const databaseOf = (name: string, environment: string): Database =>
  create(DatabaseSchema$, {
    name: `instances/pg-prod/databases/${name}`,
    effectiveEnvironment: environment,
  });

const mocks = vi.hoisted(() => {
  const state = {
    groups: [] as unknown[],
    databases: [] as unknown[],
    identities: [] as WorkloadIdentity[],
    identitiesByName: {} as Record<string, WorkloadIdentity>,
    // Members that hold a role in the project.
    members: [] as string[],
  };
  const store = {
    fetchDBGroupListByProjectName: vi.fn(async () => state.groups),
    fetchDatabases: vi.fn(async () => ({
      databases: state.databases,
      nextPageToken: "",
    })),
    listWorkloadIdentities: vi.fn(async () => ({
      workloadIdentities: state.identities,
      nextPageToken: "",
    })),
    loadProjectIamPolicy: vi.fn(async () => undefined),
    fetchWorkloadIdentity: vi.fn(async () => undefined),
    getWorkloadIdentity: (name: string) => state.identitiesByName[name],
    projectPoliciesByName: {} as Record<string, unknown>,
    workspacePolicy: undefined,
  };
  const useAppStore = vi.fn((selector: (s: typeof store) => unknown) =>
    selector(store)
  );
  return {
    state,
    store,
    canCreate: true,
    serverState: {
      serverInfo: { version: "3.14.0" },
      externalUrl: "https://bytebase.example.com",
      needConfigureExternalUrl: false,
    },
    useAppStore: Object.assign(useAppStore, { getState: () => store }),
  };
});

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key} ${JSON.stringify(values)}` : key,
  }),
  Trans: ({
    i18nKey,
    values,
  }: {
    i18nKey: string;
    values?: Record<string, unknown>;
  }) => `${i18nKey} ${JSON.stringify(values ?? {})}`,
}));

vi.mock("@/stores/app", () => ({ useAppStore: mocks.useAppStore }));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => mocks.serverState,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/shop" }),
}));

vi.mock("@/utils", async () => ({
  ...(await vi.importActual<typeof import("@/utils/workloadIdentity")>(
    "@/utils/workloadIdentity"
  )),
  getDefaultPagination: () => 50,
  hasProjectPermissionV2: () => mocks.canCreate,
  isBindingPolicyExpired: () => false,
}));

vi.mock("@/utils/role", () => ({
  displayRoleTitle: (role: string) => role,
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    children,
    to,
  }: {
    children: React.ReactNode;
    to: { name: string };
  }) => <a data-route={to.name}>{children}</a>,
}));

vi.mock("@/components/ExternalUrlAlert", () => ({
  ExternalUrlAlert: () =>
    mocks.serverState.needConfigureExternalUrl ? (
      <div data-testid="external-url-alert" />
    ) : null,
}));

vi.mock("@/components/ui/step-indicator", () => ({
  StepIndicator: ({
    steps,
    currentIndex,
  }: {
    steps: { title: string }[];
    currentIndex: number;
  }) => <div data-testid="step">{steps[currentIndex].title}</div>,
}));

vi.mock("@/components/ui/sticky-action-footer", () => ({
  StickyActionFooter: ({
    left,
    right,
  }: {
    left: React.ReactNode;
    right: React.ReactNode;
  }) => (
    <div data-testid="footer">
      {left}
      {right}
    </div>
  ),
}));

vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: ({
    content,
    disabled,
    label,
  }: {
    content: string;
    disabled?: boolean;
    label?: string;
  }) => (
    <button
      type="button"
      data-copy=""
      data-content={content}
      disabled={disabled}
    >
      {label ?? "copy"}
    </button>
  ),
}));

vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({ value, portal }: { value: string; portal?: boolean }) => (
    <div
      data-testid="group-select"
      data-value={value}
      data-portal={String(!!portal)}
    />
  ),
}));

vi.mock("@/components/DatabaseSelect", () => ({
  DatabaseSelect: ({
    value,
    onChange,
    portal,
  }: {
    value: string[];
    onChange: (names: string[]) => void;
    portal?: boolean;
  }) => (
    <div
      data-testid="database-select"
      data-value={value.join(",")}
      data-portal={String(!!portal)}
    >
      <button
        type="button"
        onClick={() => onChange(["instances/pg-prod/databases/shop_a"])}
      >
        pick-database
      </button>
    </div>
  ),
}));

vi.mock("./gitops-setup/DeployOrder", () => ({
  DeployOrder: () => <div data-testid="deploy-order" />,
}));

// Every pick lands in one environment, "prod".
vi.mock("./gitops-setup/useDeployTargets", () => ({
  useDeployTargets: (
    pick: { kind: "group"; name: string } | { kind: "databases"; names: string[] }
  ) => {
    const picked = pick.kind === "group" ? !!pick.name : pick.names.length > 0;
    return picked
      ? {
          stages: [{ environment: "environments/prod", id: "prod", databases: [] }],
          unassigned: [],
          total: 1,
          loading: false,
        }
      : { stages: [], unassigned: [], total: 0, loading: false };
  },
}));

// Each identity is a button; one that cannot be picked shows the reason.
vi.mock("./WorkloadIdentitySelect", () => ({
  WorkloadIdentitySelect: ({
    value,
    onChange,
    disabledReason,
    portal,
  }: {
    value: string;
    onChange: (name: string) => void;
    disabledReason: (wi: WorkloadIdentity) => string | undefined;
    portal?: boolean;
  }) => (
    <div
      data-testid="identity-select"
      data-value={value}
      data-portal={String(!!portal)}
    >
      {mocks.state.identities.map((wi) => {
        const reason = disabledReason(wi);
        return (
          <button
            key={wi.name}
            type="button"
            data-option={wi.email}
            disabled={reason !== undefined}
            onClick={() => onChange(wi.name)}
          >
            {reason ?? wi.email}
          </button>
        );
      })}
    </div>
  ),
}));

// Create hands back NEW_CI, as the real dialog does with closeOnCreate.
vi.mock("@/components/WorkloadIdentityFlow/WorkloadIdentityDialog", () => ({
  WorkloadIdentityDialog: ({
    open,
    closeOnCreate,
    gitProvidersOnly,
    onCreated,
    onClose,
  }: {
    open: boolean;
    closeOnCreate?: boolean;
    gitProvidersOnly?: boolean;
    onCreated: (wi: WorkloadIdentity) => void;
    onClose: () => void;
  }) =>
    open ? (
      <button
        type="button"
        data-close-on-create={String(closeOnCreate)}
        data-git-providers-only={String(gitProvidersOnly)}
        onClick={() => {
          const created = NEW_CI;
          mocks.state.identities = [...mocks.state.identities, created];
          mocks.state.identitiesByName[created.name] = created;
          mocks.state.members.push(`workloadIdentity:${created.email}`);
          setPolicy();
          onCreated(created);
          onClose();
        }}
      >
        dialog-create
      </button>
    ) : null,
}));

const setPolicy = () => {
  mocks.store.projectPoliciesByName = {
    "projects/shop": create(IamPolicySchema, {
      bindings: [
        create(BindingSchema, {
          role: "roles/gitopsServiceAgent",
          members: [...mocks.state.members],
        }),
      ],
    }),
  };
};

const setProject = ({
  groups = 0,
  databases = [],
  identities = [],
  members = [],
}: {
  groups?: number;
  databases?: Database[];
  identities?: WorkloadIdentity[];
  members?: WorkloadIdentity[];
}) => {
  mocks.state.groups = Array.from({ length: groups }, (_, i) =>
    create(DatabaseGroupSchema, {
      name: `projects/shop/databaseGroups/group-${i}`,
      title: `group-${i}`,
    })
  );
  mocks.state.databases = databases;
  mocks.state.identities = identities;
  mocks.state.identitiesByName = Object.fromEntries(
    identities.map((wi) => [wi.name, wi])
  );
  mocks.state.members = members.map((wi) => `workloadIdentity:${wi.email}`);
  setPolicy();
};

let root: Root;
let container: HTMLElement;

const render = async () => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  await act(async () => {
    root.render(<ProjectGitOpsPage projectId="shop" />);
  });
};

const button = (text: string) =>
  Array.from(container.querySelectorAll("button")).find((b) =>
    b.textContent?.includes(text)
  );
const click = async (element: Element | undefined | null) => {
  if (!element) throw new Error("element not found");
  await act(async () => {
    (element as HTMLElement).click();
  });
};
const footerButtons = () =>
  Array.from(
    container.querySelectorAll('[data-testid="footer"] button')
  ).map((b) => b.textContent);
const continueButton = () =>
  button("gitops.setup.continue") as HTMLButtonElement | undefined;
const currentStep = () =>
  container.querySelector('[data-testid="step"]')?.textContent;
const fileText = () =>
  Array.from(container.querySelectorAll("pre"))
    .map((pre) => pre.textContent)
    .join("\n");
const typeInto = async (input: HTMLInputElement, value: string) => {
  await act(async () => {
    const setter = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    )?.set;
    setter?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
};

// Walks a project with one group and the given identity to step 3.
const openStep3 = async (identity: WorkloadIdentity) => {
  setProject({
    groups: 1,
    databases: [databaseOf("shop", "environments/prod")],
    identities: [identity],
    members: [identity],
  });
  await render();
  await click(continueButton());
  await click(continueButton());
  expect(currentStep()).toBe("gitops.setup.steps.workflow-files");
};

beforeEach(() => {
  mocks.canCreate = true;
  mocks.serverState.externalUrl = "https://bytebase.example.com";
  mocks.serverState.needConfigureExternalUrl = false;
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

describe("ProjectGitOpsPage", () => {
  test("a project with one database and no group opens on Databases with it picked", async () => {
    setProject({ databases: [databaseOf("shop", "environments/prod")] });
    await render();

    expect(currentStep()).toBe("gitops.setup.steps.databases");
    expect(
      container
        .querySelector('[data-testid="database-select"]')
        ?.getAttribute("data-value")
    ).toBe("instances/pg-prod/databases/shop");
    expect(continueButton()?.disabled).toBe(false);
  });

  test("Continue is disabled with nothing picked", async () => {
    setProject({
      databases: [
        databaseOf("shop_a", "environments/prod"),
        databaseOf("shop_b", "environments/prod"),
      ],
    });
    await render();

    expect(continueButton()?.disabled).toBe(true);
    await click(button("pick-database"));
    expect(continueButton()?.disabled).toBe(false);
  });

  test("Continue is disabled for an identity with no role", async () => {
    setProject({
      groups: 1,
      databases: [databaseOf("shop", "environments/prod")],
      identities: [DEPLOY_CI],
    });
    await render();
    await click(continueButton());

    expect(currentStep()).toBe("gitops.setup.steps.workload-identity");
    expect(container.textContent).toContain(
      "gitops.setup.identity.no-role-title"
    );
    expect(continueButton()?.disabled).toBe(true);
  });

  test("an identity for another OIDC provider is listed but cannot be picked", async () => {
    setProject({
      groups: 1,
      databases: [databaseOf("shop", "environments/prod")],
      identities: [DEPLOY_CI, OTHER_CI],
      members: [DEPLOY_CI, OTHER_CI],
    });
    await render();
    await click(continueButton());

    const other = container.querySelector(
      `[data-option="${OTHER_CI.email}"]`
    ) as HTMLButtonElement;
    expect(other.disabled).toBe(true);
    expect(other.textContent).toBe("gitops.setup.identity.other-provider");
    const deploy = container.querySelector(
      `[data-option="${DEPLOY_CI.email}"]`
    ) as HTMLButtonElement;
    expect(deploy.disabled).toBe(false);
  });

  test("creating an identity from step 2 returns to step 2 with it picked", async () => {
    setProject({
      groups: 1,
      databases: [databaseOf("shop", "environments/prod")],
    });
    await render();
    await click(continueButton());
    expect(container.textContent).toContain("gitops.setup.identity.none");

    await click(button("gitops.setup.identity.create"));
    const create = button("dialog-create");
    expect(create?.getAttribute("data-close-on-create")).toBe("true");
    // Setup has workflow files for GitHub Actions and GitLab CI only.
    expect(create?.getAttribute("data-git-providers-only")).toBe("true");
    await click(create);

    expect(button("dialog-create")).toBeUndefined();
    expect(currentStep()).toBe("gitops.setup.steps.workload-identity");
    expect(
      container
        .querySelector('[data-testid="identity-select"]')
        ?.getAttribute("data-value")
    ).toBe(NEW_CI.name);
    expect(continueButton()?.disabled).toBe(false);
  });

  test("the repository link shows only when the identity names a repository", async () => {
    await openStep3(DEPLOY_CI);
    const link = Array.from(container.querySelectorAll("a")).find(
      (a) => a.textContent === "acme-corp/deploy"
    );
    expect(link?.getAttribute("href")).toBe(
      "https://github.com/acme-corp/deploy"
    );
    act(() => root.unmount());
    container.remove();

    await openStep3(ORG_CI);
    expect(container.textContent).not.toContain("gitops.setup.files.repository");
  });

  test("the git provider's logo shows before the provider name and the repository link", async () => {
    const cases = [
      { identity: DEPLOY_CI, logo: githubIcon },
      { identity: NEW_CI, logo: gitlabIcon },
    ];
    for (const [index, { identity, logo }] of cases.entries()) {
      if (index > 0) {
        act(() => root.unmount());
        container.remove();
      }
      setProject({
        groups: 1,
        databases: [databaseOf("shop", "environments/prod")],
        identities: [identity],
        members: [identity],
      });
      await render();
      await click(continueButton());
      expect(currentStep()).toBe("gitops.setup.steps.workload-identity");
      // The first fact is the git provider.
      expect(
        container.querySelector("dd")?.querySelector("img")?.getAttribute("src")
      ).toBe(logo);
      expect(container.querySelectorAll("img")).toHaveLength(1);

      await click(continueButton());
      const link = Array.from(container.querySelectorAll("a")).find(
        (a) => a.textContent === "acme-corp/deploy"
      );
      expect(link?.querySelector("img")?.getAttribute("src")).toBe(logo);
      expect(container.querySelectorAll("img")).toHaveLength(1);
    }
  });

  test("editing the folder changes the files and the developers' line", async () => {
    await openStep3(DEPLOY_CI);
    expect(fileText()).toContain('"migrations/*.sql"');

    await click(button("common.edit"));
    const input = container.querySelector(
      "#gitops-migration-folder"
    ) as HTMLInputElement;
    await typeInto(input, "db/migrations/");

    expect(fileText()).toContain('"db/migrations/*.sql"');
    expect(fileText()).not.toContain('"migrations/*.sql"');
    expect(container.textContent).toContain(
      "db/migrations/0001_create_orders.sql"
    );

    await typeInto(input, "../outside");
    expect(container.textContent).toContain(
      "gitops.setup.files.folder-invalid"
    );
    expect(fileText()).toContain('"db/migrations/*.sql"');
  });

  test("an unset External URL shows the task on every step, a placeholder and no copy", async () => {
    mocks.serverState.externalUrl = "";
    mocks.serverState.needConfigureExternalUrl = true;
    setProject({
      groups: 1,
      databases: [databaseOf("shop", "environments/prod")],
      identities: [DEPLOY_CI],
      members: [DEPLOY_CI],
    });
    await render();
    const alert = () =>
      container.querySelector('[data-testid="external-url-alert"]');

    expect(alert()).not.toBeNull();
    await click(continueButton());
    expect(alert()).not.toBeNull();
    await click(continueButton());
    expect(alert()).not.toBeNull();

    expect(fileText()).toContain("{EXTERNAL_URL}");
    const copyButtons = Array.from(
      container.querySelectorAll<HTMLButtonElement>("[data-copy]")
    );
    expect(copyButtons.length).toBeGreaterThan(0);
    expect(copyButtons.every((b) => b.disabled)).toBe(true);
    expect(container.textContent).toContain("gitops.setup.files.url-missing");
  });

  test("every picker opens its menu above the sticky footer", async () => {
    const portal = (testId: string) =>
      container
        .querySelector(`[data-testid="${testId}"]`)
        ?.getAttribute("data-portal");
    setProject({
      groups: 1,
      databases: [databaseOf("shop", "environments/prod")],
      identities: [DEPLOY_CI],
      members: [DEPLOY_CI],
    });
    await render();
    expect(portal("group-select")).toBe("true");
    act(() => root.unmount());
    container.remove();

    // With no group the Databases tab opens and its one database is picked.
    setProject({
      databases: [databaseOf("shop", "environments/prod")],
      identities: [DEPLOY_CI],
      members: [DEPLOY_CI],
    });
    await render();
    expect(portal("database-select")).toBe("true");

    await click(continueButton());
    expect(portal("identity-select")).toBe("true");
  });

  test("a long file shows its first lines until Show all, and Copy takes the whole file", async () => {
    await openStep3(DEPLOY_CI);
    const shownLines = () =>
      Array.from(container.querySelectorAll("pre")).map(
        (pre) => (pre.textContent ?? "").split("\n").length
      );
    const copied = () =>
      Array.from(container.querySelectorAll("[data-copy]")).map(
        (b) => b.getAttribute("data-content") ?? ""
      );
    expect(shownLines()).toEqual([FILE_PREVIEW_LINES, FILE_PREVIEW_LINES]);
    // The whole review file ends with its file pattern, which the cut hides.
    expect(fileText()).not.toContain("--file-pattern=");
    expect(copied().every((c) => c.includes("--file-pattern="))).toBe(true);

    const showAll = button("gitops.setup.files.show-all");
    expect(showAll?.getAttribute("aria-expanded")).toBe("false");
    await click(showAll);
    const [review, deploy] = shownLines();
    expect(review).toBeGreaterThan(FILE_PREVIEW_LINES);
    expect(deploy).toBe(FILE_PREVIEW_LINES);
    expect(fileText()).toContain("--file-pattern=");

    await click(button("common.show-less"));
    expect(shownLines()).toEqual([FILE_PREVIEW_LINES, FILE_PREVIEW_LINES]);
  });

  test("a file within the limit shows in full with no action", async () => {
    await openStep3(NEW_CI);
    // GitLab: the workflow file, then the two include lines.
    expect(container.querySelectorAll("pre")).toHaveLength(2);
    expect(
      container.querySelectorAll('[data-slot="code-block-footer"]')
    ).toHaveLength(1);
    expect(container.querySelectorAll("pre")[1].textContent).toBe(
      "include:\n  - local: bytebase.gitlab-ci.yml"
    );
  });

  test("a file's header bar names its path and nothing else", async () => {
    await openStep3(NEW_CI);
    const headers = [
      ...container.querySelectorAll('[data-slot="code-block-header"]'),
    ].map((header) => header.firstElementChild?.textContent);
    expect(headers).toEqual(["bytebase.gitlab-ci.yml", ".gitlab-ci.yml"]);
    for (const header of container.querySelectorAll(
      '[data-slot="code-block-header"]'
    )) {
      expect(header.querySelectorAll("span")).toHaveLength(1);
    }
  });

  test("the scope line shows on every step", async () => {
    setProject({
      groups: 1,
      databases: [databaseOf("shop", "environments/prod")],
      identities: [DEPLOY_CI],
      members: [DEPLOY_CI],
    });
    await render();

    expect(container.textContent).toContain("gitops.setup.scope");
    await click(continueButton());
    expect(container.textContent).toContain("gitops.setup.scope");
    await click(continueButton());
    expect(currentStep()).toBe("gitops.setup.steps.workflow-files");
    expect(container.textContent).toContain("gitops.setup.scope");
  });

  test("the footer holds Continue on steps 1 and 2 and only Back on step 3", async () => {
    await openStep3(DEPLOY_CI);
    expect(footerButtons()).toEqual(["common.back"]);

    await click(button("common.back"));
    expect(footerButtons()).toEqual(["common.back", "gitops.setup.continue"]);
    await click(button("common.back"));
    expect(footerButtons()).toEqual(["gitops.setup.continue"]);
  });
});
