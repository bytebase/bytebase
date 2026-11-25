import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import githubIcon from "@/assets/git/github.svg";
import gitlabIcon from "@/assets/git/gitlab.svg";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { DatabaseGroupSchema } from "@/types/proto-es/v1/database_group_service_pb";
import { BindingSchema, IamPolicySchema } from "@/types/proto-es/v1/iam_policy_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  type WorkloadIdentity,
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { FILE_PREVIEW_LINES } from "./gitops-setup/FilesSection";
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

const databaseOf = (
  name: string,
  environment: string,
  engine = Engine.POSTGRES
): Database =>
  create(DatabaseSchema$, {
    name: `instances/pg-prod/databases/${name}`,
    effectiveEnvironment: environment,
    instanceResource: create(InstanceResourceSchema, { engine }),
  });

const SHOP = databaseOf("shop", "environments/prod");

const mocks = vi.hoisted(() => {
  const state = {
    groups: [] as unknown[],
    databases: [] as unknown[],
    // What every pick deploys to, all in the environment "prod".
    deployed: [] as unknown[],
    identities: [] as WorkloadIdentity[],
    identitiesByName: {} as Record<string, WorkloadIdentity>,
    // Members that hold a role in the project.
    members: [] as string[],
    // Bytebase Cloud.
    saas: false,
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
    workspacePolicy: undefined as unknown,
    isSaaSMode: () => state.saas,
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
  useEnvironmentList: () => [],
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/shop" }),
}));

vi.mock("@/utils", async () => ({
  ...(await vi.importActual<typeof import("@/utils/workloadIdentity")>(
    "@/utils/workloadIdentity"
  )),
  ...(await vi.importActual<typeof import("@/utils/v1/database")>(
    "@/utils/v1/database"
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

vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: ({
    content,
    disabled,
    label,
    children,
  }: {
    content: string;
    disabled?: boolean;
    label?: string;
    children?: React.ReactNode;
  }) => (
    // The button with text copies a file's content, the icon its path.
    <button
      type="button"
      data-copy={children ? "content" : "path"}
      data-content={content}
      disabled={disabled}
      aria-label={label}
    >
      {children}
    </button>
  ),
}));

// A database option shows the reason it cannot be picked, else its name.
vi.mock("@/components/DatabaseSelectOption", () => ({
  DatabaseSelectOption: ({
    database,
    disabledReason,
  }: {
    database: Database;
    disabledReason?: string;
  }) => disabledReason ?? database.name,
}));

// Each option is a button, drawn by its own render when it has one.
vi.mock("@/components/ui/combobox", () => ({
  Combobox: ({
    value,
    options,
    onChange,
  }: {
    value: string;
    options: {
      value: string;
      label: string;
      render?: (keyword: string) => React.ReactNode;
      disabled?: boolean;
    }[];
    onChange: (value: string) => void;
  }) => (
    <div data-value={value}>
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          data-option={option.value}
          disabled={option.disabled}
          onClick={() => onChange(option.value)}
        >
          {option.render ? option.render("") : option.label}
        </button>
      ))}
    </div>
  ),
}));

vi.mock("@/components/DatabaseSelect", () => ({
  DatabaseSelect: ({
    value,
    onChange,
  }: {
    value: string[];
    onChange: (names: string[]) => void;
  }) => (
    <div data-testid="database-select" data-value={value.join(",")}>
      <button
        type="button"
        onClick={() => onChange(["instances/pg-prod/databases/shop"])}
      >
        pick-database
      </button>
    </div>
  ),
}));

vi.mock("./gitops-setup/DeployOrder", () => ({
  DeployOrder: ({ unsupported }: { unsupported: Database[] }) => (
    <div
      data-testid="deploy-order"
      data-unsupported={unsupported.map((database) => database.name).join(",")}
    />
  ),
}));

// The export is the server's; the page only passes it on.
vi.mock("./gitops-setup/useSchemaExport", () => ({
  useSchemaExport: (databaseName: string | undefined) =>
    databaseName ? { status: "loading" } : undefined,
}));

vi.mock("./gitops-setup/SchemaFiles", () => ({
  SchemaFiles: ({
    database,
    folder,
  }: {
    database: Database;
    folder: string;
  }) => (
    <div
      data-testid="schema-files"
      data-database={database.name}
      data-folder={folder}
    />
  ),
}));

// Every pick deploys to mocks.state.deployed, in one environment, "prod".
vi.mock("./gitops-setup/useDeployTargets", () => ({
  useDeployTargets: (
    pick: { kind: "group"; name: string } | { kind: "databases"; names: string[] }
  ) => {
    const picked = pick.kind === "group" ? !!pick.name : pick.names.length > 0;
    return picked
      ? {
          stages: [
            {
              environment: "environments/prod",
              id: "prod",
              databases: mocks.state.deployed,
            },
          ],
          unassigned: [],
          total: mocks.state.deployed.length,
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
  }: {
    value: string;
    onChange: (name: string) => void;
    disabledReason: (wi: WorkloadIdentity) => string | undefined;
  }) => (
    <div
      data-testid="identity-select"
      data-slot="combobox-trigger"
      data-value={value}
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
vi.mock("@/components/WorkloadIdentityFlow/WorkloadIdentitySheet", () => ({
  WorkloadIdentitySheet: ({
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

// A policy whose one binding grants the GitOps role to every account.
const allUsersPolicy = () =>
  create(IamPolicySchema, {
    bindings: [
      create(BindingSchema, {
        role: "roles/gitopsServiceAgent",
        members: ["allUsers"],
      }),
    ],
  });

const setProject = ({
  groups = 0,
  databases = [SHOP],
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
  mocks.state.deployed = databases.filter((d) => d.effectiveEnvironment);
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
const rerender = async () => {
  act(() => root.unmount());
  container.remove();
  await render();
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
const option = (value: string) =>
  container.querySelector(`[data-option="${value}"]`) as HTMLButtonElement;
const pickGroup = () => click(option("projects/shop/databaseGroups/group-0"));
const pickIdentity = (identity: WorkloadIdentity) =>
  click(option(identity.email));
const pickStateBased = () =>
  click(container.querySelectorAll('[role="radio"]')[1]);
const fileText = () =>
  Array.from(container.querySelectorAll("pre"))
    .map((pre) => pre.textContent)
    .join("\n");
const stepTitles = () =>
  Array.from(container.querySelectorAll("ol:not([data-locked]) > li h3")).map(
    (h) => h.textContent
  );
const pending = () => container.textContent?.includes("files.waiting-for");
const folderInput = () =>
  container.querySelector("#gitops-folder") as HTMLInputElement;
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

// A project with one group and the given identity, both picked.
const openFiles = async (identity: WorkloadIdentity) => {
  setProject({ groups: 1, identities: [identity], members: [identity] });
  await render();
  await pickGroup();
  await pickIdentity(identity);
  expect(pending()).toBe(false);
};

beforeEach(() => {
  mocks.canCreate = true;
  mocks.serverState.externalUrl = "https://bytebase.example.com";
  mocks.serverState.needConfigureExternalUrl = false;
  mocks.store.workspacePolicy = undefined;
  mocks.state.saas = false;
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

describe("ProjectGitOpsPage", () => {
  test("only the workflow type starts picked, even with one database and one identity", async () => {
    setProject({ identities: [DEPLOY_CI], members: [DEPLOY_CI] });
    await render();

    const radios = container.querySelectorAll('[role="radio"]');
    expect(radios[0].getAttribute("aria-checked")).toBe("true");
    expect(radios[1].getAttribute("aria-checked")).toBe("false");
    expect(
      container
        .querySelector('[data-testid="database-select"]')
        ?.getAttribute("data-value")
    ).toBe("");
    expect(
      container
        .querySelector('[data-testid="identity-select"]')
        ?.getAttribute("data-value")
    ).toBe("");
    expect(pending()).toBe(true);
    expect(container.querySelectorAll("pre")).toHaveLength(0);
  });

  test("an open setup shows the locked steps, and an open choice takes the owner to its field", async () => {
    setProject({ identities: [DEPLOY_CI], members: [DEPLOY_CI] });
    await render();
    const scrolled: string[] = [];
    Element.prototype.scrollIntoView = function (this: Element) {
      scrolled.push(this.id);
    };
    vi.stubGlobal("matchMedia", () => ({ matches: false }));

    const waiting = () =>
      Array.from(container.querySelectorAll("[data-choice]")).map((button) =>
        button.getAttribute("data-choice")
      );
    const locked = () =>
      Array.from(container.querySelectorAll("ol[data-locked] h3")).map(
        (h) => h.textContent
      );
    expect(waiting()).toEqual(["deploy-to", "sign-in"]);
    // The git provider is not known yet.
    expect(locked()).toEqual([
      "gitops.setup.files.workflow-step-title",
      "gitops.setup.files.request-step-title",
    ]);

    // The field's picker gets the click that opens it.
    const opened = vi.fn();
    container
      .querySelector("#gitops-field-sign-in [data-slot=combobox-trigger]")
      ?.addEventListener("click", opened);
    await click(container.querySelector('[data-choice="sign-in"]'));
    expect(scrolled).toEqual(["gitops-field-sign-in"]);
    expect(opened).toHaveBeenCalledTimes(1);

    await pickStateBased();
    await pickIdentity(DEPLOY_CI);
    expect(waiting()).toEqual(["deploy-to"]);
    expect(locked()).toEqual([
      "gitops.setup.files.workflow-step-title",
      "gitops.setup.files.schema-step-title",
      "gitops.setup.files.request-step-title-github",
    ]);
    vi.unstubAllGlobals();
  });

  test("the files show once the databases and the identity are picked", async () => {
    setProject({ identities: [DEPLOY_CI], members: [DEPLOY_CI] });
    await render();

    await click(button("pick-database"));
    expect(pending()).toBe(true);
    await pickIdentity(DEPLOY_CI);
    expect(pending()).toBe(false);
    expect(stepTitles()).toEqual([
      "gitops.setup.files.workflow-step-title",
      "gitops.setup.files.request-step-title-github",
    ]);
    expect(fileText()).toContain("name: Bytebase SQL review");
  });

  test("an identity with no role shows no file and says what it needs", async () => {
    setProject({ groups: 1, identities: [DEPLOY_CI] });
    await render();
    await pickGroup();
    await pickIdentity(DEPLOY_CI);

    expect(container.textContent).toContain(
      "gitops.setup.identity.no-role-title"
    );
    expect(pending()).toBe(true);
    expect(container.textContent).toContain(
      "gitops.setup.files.choice-needs-role"
    );
    expect(container.querySelectorAll("pre")).toHaveLength(0);
  });

  test("a role granted to allUsers in the project counts for the identity", async () => {
    setProject({ groups: 1, identities: [DEPLOY_CI] });
    mocks.store.projectPoliciesByName = {
      "projects/shop": allUsersPolicy(),
    };
    await render();
    await pickGroup();
    await pickIdentity(DEPLOY_CI);

    expect(container.textContent).toContain("roles/gitopsServiceAgent");
    expect(container.textContent).not.toContain(
      "gitops.setup.identity.no-role-title"
    );
    expect(pending()).toBe(false);
  });

  test("a role granted to allUsers in the workspace counts, except on Bytebase Cloud", async () => {
    setProject({ groups: 1, identities: [DEPLOY_CI] });
    mocks.store.workspacePolicy = allUsersPolicy();
    await render();
    await pickGroup();
    await pickIdentity(DEPLOY_CI);
    expect(pending()).toBe(false);

    mocks.state.saas = true;
    await rerender();
    await pickGroup();
    await pickIdentity(DEPLOY_CI);
    expect(container.textContent).toContain(
      "gitops.setup.identity.no-role-title"
    );
    expect(pending()).toBe(true);
  });

  test("an identity for another OIDC provider is listed but cannot be picked", async () => {
    setProject({
      groups: 1,
      identities: [DEPLOY_CI, OTHER_CI],
      members: [DEPLOY_CI, OTHER_CI],
    });
    await render();

    const other = option(OTHER_CI.email);
    expect(other.disabled).toBe(true);
    expect(other.textContent).toBe("gitops.setup.identity.other-provider");
    expect(option(DEPLOY_CI.email).disabled).toBe(false);
  });

  test("creating an identity picks it", async () => {
    setProject({ groups: 1 });
    await render();
    expect(container.textContent).toContain("gitops.setup.identity.none");

    await click(button("gitops.setup.identity.create"));
    const createButton = button("dialog-create");
    expect(createButton?.getAttribute("data-close-on-create")).toBe("true");
    // Setup has workflow files for GitHub Actions and GitLab CI only.
    expect(createButton?.getAttribute("data-git-providers-only")).toBe("true");
    await click(createButton);

    expect(button("dialog-create")).toBeUndefined();
    expect(
      container
        .querySelector('[data-testid="identity-select"]')
        ?.getAttribute("data-value")
    ).toBe(NEW_CI.name);
  });

  test("the repository link shows only when the identity names a repository", async () => {
    await openFiles(DEPLOY_CI);
    const link = Array.from(container.querySelectorAll("a")).find(
      (a) => a.textContent === "acme-corp/deploy"
    );
    expect(link?.getAttribute("href")).toBe(
      "https://github.com/acme-corp/deploy"
    );

    act(() => root.unmount());
    container.remove();
    await openFiles(ORG_CI);
    expect(container.textContent).not.toContain("gitops.setup.files.repository");
  });

  test("the git provider's logo shows beside the identity and the repository link", async () => {
    for (const [index, { identity, logo }] of [
      { identity: DEPLOY_CI, logo: githubIcon },
      { identity: NEW_CI, logo: gitlabIcon },
    ].entries()) {
      if (index > 0) {
        act(() => root.unmount());
        container.remove();
      }
      await openFiles(identity);
      const logos = Array.from(container.querySelectorAll("img")).map((img) =>
        img.getAttribute("src")
      );
      expect(logos).toEqual([logo, logo]);
      const link = Array.from(container.querySelectorAll("a")).find(
        (a) => a.textContent === "acme-corp/deploy"
      );
      expect(link?.querySelector("img")?.getAttribute("src")).toBe(logo);
    }
  });

  test("the folder changes the files, and an invalid folder keeps the last valid one", async () => {
    await openFiles(DEPLOY_CI);
    expect(fileText()).toContain('"migrations/*.sql"');

    await typeInto(folderInput(), "db/migrations/");
    expect(fileText()).toContain('"db/migrations/*.sql"');
    expect(container.textContent).toContain(
      "db/migrations/0001_create_orders.sql"
    );

    await typeInto(folderInput(), "../outside");
    expect(container.textContent).toContain(
      "gitops.setup.files.folder-invalid"
    );
    expect(fileText()).toContain('"db/migrations/*.sql"');
  });

  test("an untouched folder follows the workflow type, a typed one stays", async () => {
    setProject({ groups: 1 });
    await render();
    expect(folderInput().value).toBe("migrations");
    await pickStateBased();
    expect(folderInput().value).toBe("schema");
    expect(container.textContent).toContain(
      "gitops.setup.files.schema-folder-label"
    );

    await typeInto(folderInput(), "db/schema");
    await click(container.querySelectorAll('[role="radio"]')[0]);
    expect(folderInput().value).toBe("db/schema");
  });

  test("state-based asks for the export database and adds the schema files step", async () => {
    const mysql = databaseOf("shop_mysql", "environments/prod", Engine.MYSQL);
    const mongo = databaseOf("shop_mongo", "environments/prod", Engine.MONGODB);
    setProject({
      groups: 1,
      databases: [SHOP, mysql, mongo],
      identities: [DEPLOY_CI],
      members: [DEPLOY_CI],
    });
    await render();
    await pickGroup();
    const unsupported = () =>
      container
        .querySelector('[data-testid="deploy-order"]')
        ?.getAttribute("data-unsupported");
    // Migration-based deploys to every engine, and shows no limits.
    expect(unsupported()).toBe("");
    const limits = () =>
      container.textContent?.includes("gitops.setup.workflow-type.limits-title");
    expect(limits()).toBe(false);
    await pickStateBased();
    expect(unsupported()).toBe(mongo.name);
    expect(limits()).toBe(true);
    await pickIdentity(DEPLOY_CI);

    // The steps do not wait for the export database: it is picked inside the
    // schema files step, and the form has no field for it.
    expect(pending()).toBe(false);
    const schemaStep = container.querySelectorAll("ol:not([data-locked]) > li")[1];
    expect(schemaStep.textContent).toContain("gitops.setup.export-schema.label");
    expect(container.querySelector('[data-testid="schema-files"]')).toBeNull();
    // An engine state-based does not support is not listed.
    expect(option(mongo.name)).toBeNull();

    await click(option(mysql.name));
    expect(stepTitles()).toEqual([
      "gitops.setup.files.workflow-step-title",
      "gitops.setup.files.schema-step-title",
      "gitops.setup.files.request-step-title-github",
    ]);
    const schema = container.querySelector('[data-testid="schema-files"]');
    expect(schema?.getAttribute("data-database")).toBe(mysql.name);
    expect(schema?.getAttribute("data-folder")).toBe("schema");

    await click(button("gitops.setup.files.show-all"));
    expect(fileText()).toContain('"schema/**/*.sql"');
    expect(fileText()).toContain("--declarative");
  });

  test("an unset External URL shows the task, a placeholder and no copy", async () => {
    mocks.serverState.externalUrl = "";
    mocks.serverState.needConfigureExternalUrl = true;
    await openFiles(DEPLOY_CI);

    expect(
      container.querySelector('[data-testid="external-url-alert"]')
    ).not.toBeNull();
    expect(fileText()).toContain("{EXTERNAL_URL}");
    const copyButtons = Array.from(
      container.querySelectorAll<HTMLButtonElement>('[data-copy="content"]')
    );
    expect(copyButtons.length).toBeGreaterThan(0);
    expect(copyButtons.every((b) => b.disabled)).toBe(true);
    // A path holds no address, so it can still be copied.
    const pathButtons = Array.from(
      container.querySelectorAll<HTMLButtonElement>('[data-copy="path"]')
    );
    expect(pathButtons.every((b) => !b.disabled)).toBe(true);
    expect(container.textContent).toContain("gitops.setup.files.url-missing");
  });

  test("a long file shows its first lines until Show all, and Copy takes the whole file", async () => {
    await openFiles(DEPLOY_CI);
    const shownLines = () =>
      Array.from(container.querySelectorAll("pre")).map(
        (pre) => (pre.textContent ?? "").split("\n").length
      );
    const copied = () =>
      Array.from(container.querySelectorAll('[data-copy="content"]')).map(
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

  test("GitLab says to add the include lines to an existing .gitlab-ci.yml", async () => {
    await openFiles(NEW_CI);
    // The workflow file, then the two include lines, which have no action.
    expect(container.querySelectorAll("pre")).toHaveLength(2);
    expect(
      container.querySelectorAll('[data-slot="code-block-footer"]')
    ).toHaveLength(1);
    expect(container.querySelectorAll("pre")[1].textContent).toBe(
      "include:\n  - local: bytebase.gitlab-ci.yml"
    );
    expect(container.textContent).toContain(
      "gitops.setup.files.gitlab-include-note"
    );
    expect(stepTitles().at(-1)).toBe(
      "gitops.setup.files.request-step-title-gitlab"
    );
  });

  test("a file's header bar names its path and nothing else", async () => {
    await openFiles(NEW_CI);
    const headers = [
      ...container.querySelectorAll('[data-slot="code-block-header"]'),
    ];
    expect(headers.map((header) => header.firstElementChild?.textContent)).toEqual(
      ["bytebase.gitlab-ci.yml", ".gitlab-ci.yml"]
    );
    for (const header of headers) {
      expect(header.querySelectorAll("span")).toHaveLength(1);
    }
    // The icon beside a path copies that path.
    expect(
      headers.map((header) =>
        header.querySelector('[data-copy="path"]')?.getAttribute("data-content")
      )
    ).toEqual(["bytebase.gitlab-ci.yml", ".gitlab-ci.yml"]);
  });

  test("the intro says which CI/CD platforms setup covers", async () => {
    setProject({ groups: 1 });
    await render();
    expect(container.textContent).toContain("gitops.setup.scope");
  });
});
