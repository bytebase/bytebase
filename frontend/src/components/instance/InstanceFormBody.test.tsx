import { create } from "@bufbuild/protobuf";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceType,
  InstanceSchema,
  SyncDatabasesSchema,
  Vendor,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";
import {
  InstanceFormProvider,
  useInstanceFormContext,
} from "./InstanceFormContext";

const mocks = vi.hoisted(() => ({
  translate: (key: string) => key,
  hasInstancePermission: vi.fn(() => true),
  pushNotification: vi.fn(),
  createInstance: vi.fn(),
  isSaaSMode: false,
  hasSSL: false,
  hasExtraParameters: false,
  RealDataSourceForm: null as
    | typeof import("./DataSourceForm")["DataSourceForm"]
    | null,
  RealRedisSentinelFields: null as
    | typeof import("./DataSourceForm")["RedisSentinelFields"]
    | null,
  listInstanceDatabases: vi.fn(async () => ({
    databases: ["app", "analytics"],
  })),
}));

vi.mock("@/components/EngineIcon", () => ({ EngineIcon: () => null }));
vi.mock("@/components/EnvironmentSelect", () => ({
  EnvironmentSelect: ({ className }: { className?: string }) => (
    <div data-testid="environment-select" className={className} />
  ),
}));
vi.mock("@/components/FeatureBadge", () => ({ FeatureBadge: () => null }));
vi.mock("@/components/LabelListEditor", () => ({
  LabelListEditor: () => <div data-testid="label-list-editor" />,
}));
vi.mock("@/components/LearnMoreLink", () => ({ LearnMoreLink: () => null }));
vi.mock("@/components/ResourceIdField", () => ({
  ResourceIdField: () => null,
}));
vi.mock("@/components/RouterLink", () => ({ RouterLink: () => null }));
vi.mock("./DataSourceForm", () => ({
  DataSourceForm: (props: {
    dataSource: EditDataSource;
    authOnly?: boolean;
    hideAuthentication?: boolean;
    optionsOnly?: boolean;
    onDataSourceChange: (value: EditDataSource) => void;
  }) => {
    if (mocks.RealDataSourceForm) {
      const Real = mocks.RealDataSourceForm;
      return <Real {...props} />;
    }
    const { dataSource, authOnly, hideAuthentication, onDataSourceChange } =
      props;
    return authOnly ? (
      <button
        type="button"
        onClick={() =>
          onDataSourceChange({
            ...dataSource,
            authenticationType:
              DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
          })
        }
      >
        authentication:{dataSource.id}
      </button>
    ) : (
      <div
        data-testid="credentials"
        data-source={dataSource.id}
        data-hide-auth={hideAuthentication}
      />
    );
  },
  RedisSentinelFields: (props: {
    dataSource: EditDataSource;
    isCreating: boolean;
    allowEdit: boolean;
    allowUsingEmptyPassword: boolean;
    onDataSourceChange: (dataSource: EditDataSource) => void;
  }) => {
    if (mocks.RealRedisSentinelFields) {
      const Real = mocks.RealRedisSentinelFields;
      return <Real {...props} />;
    }
    return null;
  },
}));

vi.mock("./permission", () => ({
  hasInstancePermission: mocks.hasInstancePermission,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: mocks.translate,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  default: {
    t: mocks.translate,
  },
}));

vi.mock("monaco-editor", () => ({}));

vi.mock(
  "@codingame/monaco-vscode-editor-api/vscode/src/vs/editor/standalone/browser/standalone-tokens.css",
  () => ({})
);

vi.mock("@/types", () => ({
  UNKNOWN_ID: -1,
  languageOfEngineV1: () => "sql",
  isValidEnvironmentName: () => false,
  DATASOURCE_ADMIN_USER_NAME: "bytebase",
  DATASOURCE_READONLY_USER_NAME: "bytebase_readonly",
  UNKNOWN_INSTANCE_NAME: "instances/-",
  unknownDataSource: () => ({
    id: "admin",
    type: 1,
    host: "",
    port: "",
    username: "",
    password: "",
    database: "",
    additionalAddresses: [],
    extraConnectionParameters: {},
  }),
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

let mockEnvironmentList: { id: string; name: string }[] = [];

vi.mock("@/stores/app", () => {
  const appState = () => ({
    createDataSource: vi.fn(),
    createInstance: mocks.createInstance,
    listInstanceDatabases: mocks.listInstanceDatabases,
    updateDataSource: vi.fn(),
    getEnvironmentByName: (name: string) => ({ name }),
    hasInstanceFeature: () => false,
    hasUnifiedInstanceLicense: () => false,
    instanceLicenseCount: () => 1,
    activatedInstanceCount: () => 0,
    currentPlan: () => 1,
    isSaaSMode: () => mocks.isSaaSMode,
    environmentList: mockEnvironmentList,
  });
  return {
    useAppStore: Object.assign(
      (selector: (state: unknown) => unknown) => selector(appState()),
      { getState: appState }
    ),
  };
});

vi.mock("@/utils", () => ({
  MAX_LABEL_VALUE_LENGTH: 63,
  calcUpdateMask: () => [],
  convertKVListToLabels: (list: { key: string; value: string }[]) =>
    Object.fromEntries(list.map(({ key, value }) => [key, value])),
  convertLabelsToKVList: (labels: Record<string, string>) =>
    Object.entries(labels).map(([key, value]) => ({ key, value })),
  hasWorkspacePermissionV2: () => true,
  isValidEnvironmentName: () => false,
  engineNameV1: () => "Engine",
  supportedEngineV1List: () => [],
  extractInstanceResourceName: () => "prod",
  onlyAllowNumber: (value: string) => /^\d+$/.test(value),
  RE_GCP_PROJECT_ID: /^[a-z-]+$/,
  RE_GCP_INSTANCE_ID: /^[a-z-]+$/,
  instanceV1HasExtraParameters: () => mocks.hasExtraParameters,
  instanceV1HasSSH: () => false,
  instanceV1HasSSL: () => mocks.hasSSL,
  isValidSpannerDataSource: (ds: { projectId: string; instanceId: string }) =>
    ds.projectId !== "" && ds.instanceId !== "",
  isValidBigQueryDataSource: (ds: { projectId: string }) => ds.projectId !== "",
}));

vi.mock("@/utils/connect", () => ({
  extractGrpcErrorMessage: (error: unknown) =>
    error instanceof Error ? error.message : String(error),
}));

vi.mock("@/components/ui/feature-modal", () => ({
  FeatureModal: ({
    open,
    feature,
    instance,
    onOpenChange,
  }: {
    open: boolean;
    feature: number | undefined;
    instance?: { name: string };
    onOpenChange: (open: boolean) => void;
  }) =>
    open ? (
      <div
        data-testid="feature-modal"
        data-feature={String(feature)}
        data-instance={instance?.name}
      >
        <button
          data-testid="feature-modal-close"
          type="button"
          onClick={() => onOpenChange(false)}
        />
      </div>
    ) : null,
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

import { InstanceFormBody, SyncDatabases } from "./InstanceFormBody";

let context: ReturnType<typeof useInstanceFormContext>;
function Editor() {
  context = useInstanceFormContext();
  return <InstanceFormBody />;
}
function mount(engine: Engine) {
  return render(
    <InstanceFormProvider
      instance={create(InstanceSchema, {
        name: "instances/prod",
        title: "Prod",
        engine,
        state: State.ACTIVE,
        dataSources: [
          {
            id: "admin",
            type: DataSourceType.ADMIN,
            authenticationType: DataSource_AuthenticationType.PASSWORD,
            host: "primary.example.com",
            port: "5432",
            database: "postgres",
            projectId: "admin-project",
            instanceId: "admin-instance",
            additionalAddresses: [{ host: "admin-node", port: "1234" }],
          },
          {
            id: "readonly",
            type: DataSourceType.READ_ONLY,
            host: "replica.example.com",
            port: "5433",
            projectId: "reader-project",
            instanceId: "reader-instance",
            additionalAddresses: [{ host: "reader-node", port: "5678" }],
          },
        ],
      })}
    >
      <Editor />
    </InstanceFormProvider>
  );
}

afterEach(() => {
  cleanup();
  mocks.pushNotification.mockClear();
});

const input = (container: HTMLElement, id: string) =>
  container.querySelector<HTMLInputElement>(`#${id}`)!;

test("uses a switch for synchronization scope", () => {
  const { container } = render(
    <InstanceFormProvider>
      <SyncDatabases
        isCreating
        showLabel
        allowEdit
        onSyncDatabasesChange={() => undefined}
      />
    </InstanceFormProvider>
  );

  expect(container.querySelector('[role="switch"]')).toHaveAttribute(
    "aria-checked",
    "true"
  );
});

test("places general Agent help beside the connection section heading", () => {
  const onAskAgent = vi.fn();
  render(
    <InstanceFormProvider>
      <InstanceFormBody onAskAgent={onAskAgent} />
    </InstanceFormProvider>
  );

  const connectionTitle = screen.getByRole("heading", {
    name: "instance.section.connection",
  });
  const headingRow = connectionTitle.closest(
    '[data-slot="form-section-heading-row"]'
  );
  const askAgentButton = screen.getByRole("button", {
    name: "instance.ask-agent",
  });
  const importUrlButton = screen.getByRole("button", {
    name: "instance.connection-string.trigger",
  });

  expect(headingRow?.contains(askAgentButton)).toBe(true);
  expect(headingRow?.contains(importUrlButton)).toBe(true);
  expect(importUrlButton).toHaveClass("h-7", "bg-control-bg", "text-control");
  expect(importUrlButton).not.toHaveClass("border");
  expect(importUrlButton.querySelector(".lucide-link")).not.toBeNull();
  expect(askAgentButton).toHaveClass("ml-auto");
  expect(
    importUrlButton.compareDocumentPosition(askAgentButton) &
      Node.DOCUMENT_POSITION_FOLLOWING
  ).toBeTruthy();
  expect(connectionTitle).toHaveStyle({ width: "fit-content" });
  expect(askAgentButton.querySelector(".lucide-bot")).not.toBeNull();
  expect(askAgentButton).toHaveClass("h-7", "bg-transparent", "text-accent");
  expect(askAgentButton.querySelector(".truncate")).toHaveTextContent(
    "instance.ask-agent"
  );
  expect(askAgentButton).toHaveTextContent("instance.ask-agent");
  expect(
    screen.queryByText("instance.need-help-connecting")
  ).not.toBeInTheDocument();

  fireEvent.click(askAgentButton);
  expect(onAskAgent).toHaveBeenCalledOnce();
});

test("hides database-selection guidance while syncing all databases", () => {
  render(
    <InstanceFormProvider>
      <SyncDatabases
        isCreating
        showLabel
        allowEdit
        disabledReason="Enter a hostname to choose which databases to sync."
        onSyncDatabasesChange={() => undefined}
      />
    </InstanceFormProvider>
  );

  expect(
    screen.queryByText(
      "Enter a hostname to choose which databases to sync."
    )
  ).not.toBeInTheDocument();
  expect(screen.getByRole("switch")).not.toHaveAttribute("aria-describedby");
});

test("keeps newly loaded selections in the database preview", async () => {
  mocks.listInstanceDatabases.mockResolvedValue({ databases: ["analytics"] });
  const instance = create(InstanceSchema, {
    name: "instances/production",
    engine: Engine.POSTGRES,
  });
  const { rerender } = render(
    <InstanceFormProvider instance={instance}>
      <SyncDatabases
        isCreating={false}
        showLabel={false}
        allowEdit
        syncDatabases={create(SyncDatabasesSchema, {
          databases: ["analytics"],
        })}
        onSyncDatabasesChange={() => undefined}
      />
    </InstanceFormProvider>
  );

  try {
    await waitFor(() => {
      expect(screen.getByText("analytics")).toBeInTheDocument();
    });

    rerender(
      <InstanceFormProvider instance={instance}>
        <SyncDatabases
          isCreating={false}
          showLabel={false}
          allowEdit
          syncDatabases={create(SyncDatabasesSchema, {
            databases: ["app"],
          })}
          onSyncDatabasesChange={() => undefined}
        />
      </InstanceFormProvider>
    );

    await waitFor(() => {
      expect(screen.getByText("app")).toBeInTheDocument();
    });
    expect(
      screen
        .getByText("app")
        .closest("label")
        ?.querySelector('[role="checkbox"]')
    ).toHaveAttribute("data-checked");
  } finally {
    mocks.listInstanceDatabases.mockResolvedValue({
      databases: ["app", "analytics"],
    });
  }
});

test("keeps selections added while the database preview request is pending", async () => {
  const requestCount = mocks.listInstanceDatabases.mock.calls.length;
  let resolvePreview:
    | ((value: { databases: string[] }) => void)
    | undefined;
  mocks.listInstanceDatabases.mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        resolvePreview = resolve;
      })
  );
  const instance = create(InstanceSchema, {
    name: "instances/production",
    engine: Engine.POSTGRES,
  });
  const { rerender } = render(
    <InstanceFormProvider instance={instance}>
      <SyncDatabases
        isCreating={false}
        showLabel={false}
        allowEdit
        syncDatabases={create(SyncDatabasesSchema, {
          databases: ["analytics"],
        })}
        onSyncDatabasesChange={() => undefined}
      />
    </InstanceFormProvider>
  );

  try {
    await waitFor(() => {
      expect(mocks.listInstanceDatabases).toHaveBeenCalledTimes(requestCount + 1);
    });

    rerender(
      <InstanceFormProvider instance={instance}>
        <SyncDatabases
          isCreating={false}
          showLabel={false}
          allowEdit
          syncDatabases={create(SyncDatabasesSchema, {
            databases: ["app"],
          })}
          onSyncDatabasesChange={() => undefined}
        />
      </InstanceFormProvider>
    );

    await act(async () => {
      resolvePreview?.({ databases: ["analytics"] });
    });
    await waitFor(() => {
      expect(screen.getByText("app")).toBeInTheDocument();
    });
  } finally {
    mocks.listInstanceDatabases.mockResolvedValue({
      databases: ["app", "analytics"],
    });
  }
});

test("keeps labels in their own form field", () => {
  const { container } = render(
    <InstanceFormProvider>
      <InstanceFormBody />
    </InstanceFormProvider>
  );

  const environmentField = Array.from(
    container.querySelectorAll('[data-slot="form-field"]')
  ).find((field) =>
    field
      .querySelector('[data-slot="form-field-title"]')
      ?.textContent?.includes("common.environment")
  );
  const labelsField = Array.from(
    container.querySelectorAll('[data-slot="form-field"]')
  ).find((field) =>
    field
      .querySelector('[data-slot="form-field-title"]')
      ?.textContent?.includes("common.labels")
  );

  expect(environmentField?.textContent).not.toContain("instance.add-labels");
  expect(labelsField).toContainElement(
    container.querySelector('[data-testid="label-list-editor"]')
  );
});

test("uses the shared form width for basic info controls", () => {
  const { container } = render(
    <InstanceFormProvider>
      <InstanceFormBody />
    </InstanceFormProvider>
  );

  expect(container.querySelector("#name")).toHaveClass("w-full");
  expect(container.querySelector("#name")).not.toHaveClass("max-w-[40rem]");
  expect(
    container.querySelector('[data-testid="environment-select"]')
  ).toHaveClass("w-full");
  expect(
    container.querySelector('[data-testid="environment-select"]')
  ).not.toHaveClass("max-w-[40rem]");
});

test("names the external link info button distinctly", async () => {
  const { container } = render(
    <InstanceFormProvider
      instance={create(InstanceSchema, {
        name: "instances/production",
        engine: Engine.POSTGRES,
        externalLink: "https://example.com",
      })}
    >
      <InstanceFormBody />
    </InstanceFormProvider>
  );
  vi.useFakeTimers();

  try {
    const externalLinkInput = container.querySelector("#external-link");
    const field = externalLinkInput?.closest('[data-slot="form-field"]');
    const infoButton = field?.querySelector<HTMLButtonElement>(
      'button[aria-label="instance.external-link common.info"]'
    );
    const openLinkButton = field?.querySelector<HTMLButtonElement>(
      'button[aria-label="instance.external-link"]'
    );

    expect(infoButton).toBeDefined();
    expect(openLinkButton).toBeDefined();

    fireEvent.focus(infoButton!);
    await act(async () => {
      vi.advanceTimersByTime(100);
    });

    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("instance.sentence.console.snowflake");

    const scanIntervalField = Array.from(
      container.querySelectorAll('[data-slot="form-field"]')
    ).find((candidate) =>
      candidate
        .querySelector('[data-slot="form-field-title"]')
        ?.textContent?.includes("instance.scan-interval.self")
    );
    const scanIntervalInfoButton =
      scanIntervalField?.querySelector<HTMLButtonElement>(
        'button[aria-label="instance.scan-interval.self"]'
      );

    expect(
      scanIntervalField?.querySelector('[data-slot="form-field-description"]')
    ).toBeNull();
    expect(scanIntervalInfoButton).toBeDefined();

    fireEvent.focus(scanIntervalInfoButton!);
    await act(async () => {
      vi.advanceTimersByTime(100);
    });
    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("instance.scan-interval.description");
  } finally {
    vi.useRealTimers();
  }
});

test("each connection tab edits its own endpoint and authentication", () => {
  const { container } = mount(Engine.POSTGRES);
  expect(input(container, "host").value).toBe("primary.example.com");
  fireEvent.click(screen.getByRole("button", { name: "common.read-only" }));
  expect(input(container, "host").value).toBe("replica.example.com");
  expect(input(container, "port").value).toBe("5433");
  expect(screen.getAllByTestId("credentials")[0].dataset.hideAuth).toBe("true");
  fireEvent.change(input(container, "host"), {
    target: { value: "new-replica.example.com" },
  });
  fireEvent.change(input(container, "port"), { target: { value: "6432" } });
  fireEvent.click(
    screen.getByRole("button", { name: "authentication:readonly" })
  );
  expect(context.editingDataSource).toMatchObject({
    host: "new-replica.example.com",
    port: "6432",
    authenticationType: DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
  });
  expect(context.adminDataSource).toMatchObject({
    host: "primary.example.com",
    port: "5432",
    authenticationType: DataSource_AuthenticationType.PASSWORD,
  });
  expect(
    context.extractDataSourceFromEdit(Engine.POSTGRES, context.adminDataSource)
      .database
  ).toBe("postgres");
  fireEvent.click(screen.getByRole("button", { name: "common.admin" }));
  expect(input(container, "host").value).toBe("primary.example.com");
  fireEvent.change(input(container, "host"), {
    target: { value: "new-primary.example.com" },
  });
  expect(context.readonlyDataSourceList[0].host).toBe(
    "new-replica.example.com"
  );
  fireEvent.click(screen.getByRole("button", { name: "common.read-only" }));
  expect(input(container, "host").value).toBe("new-replica.example.com");
});

test.each([Engine.CASSANDRA, Engine.MONGODB])(
  "additional addresses belong to the selected connection for engine %s",
  (engine) => {
    mount(engine);
    fireEvent.click(screen.getByRole("button", { name: "common.read-only" }));
    fireEvent.change(screen.getByDisplayValue("reader-node"), {
      target: { value: "new-reader-node" },
    });
    fireEvent.change(screen.getByDisplayValue("5678"), {
      target: { value: "8765" },
    });
    expect(context.readonlyDataSourceList[0].additionalAddresses).toEqual([
      {
        host: "new-reader-node",
        port: "8765",
        $typeName: "bytebase.v1.DataSource.Address",
      },
    ]);
    fireEvent.click(
      screen.getByRole("button", { name: "common.delete" })
    );
    expect(context.readonlyDataSourceList[0].additionalAddresses).toHaveLength(
      0
    );
    fireEvent.click(
      screen.getByRole("button", { name: "common.add" })
    );
    expect(context.readonlyDataSourceList[0].additionalAddresses).toHaveLength(
      1
    );
    expect(context.adminDataSource.additionalAddresses[0]).toMatchObject({
      host: "admin-node",
      port: "1234",
    });
  }
);

test.each([Engine.SPANNER, Engine.BIGQUERY])(
  "GCP resource fields belong to the selected connection for engine %s",
  (engine) => {
    mount(engine);
    fireEvent.click(screen.getByRole("button", { name: "common.read-only" }));
    fireEvent.change(screen.getByDisplayValue("reader-project"), {
      target: { value: "updated-project" },
    });
    expect(context.readonlyDataSourceList[0].projectId).toBe("updated-project");
    expect(context.adminDataSource.projectId).toBe("admin-project");
    if (engine === Engine.SPANNER) {
      fireEvent.change(screen.getByDisplayValue("reader-instance"), {
        target: { value: "updated-instance" },
      });
      expect(context.readonlyDataSourceList[0].instanceId).toBe(
        "updated-instance"
      );
      expect(context.adminDataSource.instanceId).toBe("admin-instance");
    }
  }
);

test("MongoDB SRV mode only clears the selected connection's address fields", () => {
  const { container } = mount(Engine.MONGODB);
  fireEvent.click(screen.getByRole("button", { name: "common.read-only" }));
  fireEvent.click(screen.getByRole("radio", { name: "mongodb+srv://" }));
  expect(context.readonlyDataSourceList[0]).toMatchObject({
    srv: true,
    port: "",
    additionalAddresses: [],
  });
  expect(input(container, "port").disabled).toBe(true);
  expect(context.adminDataSource).toMatchObject({ srv: false, port: "5432" });
  expect(context.adminDataSource.additionalAddresses).toHaveLength(1);
});

test("Redis connection mode belongs to the selected data source", async () => {
  mount(Engine.REDIS);
  fireEvent.click(screen.getByRole("button", { name: "common.read-only" }));
  fireEvent.click(
    screen.getByRole("combobox", { name: "data-source.connection-type" })
  );
  const cluster = await screen.findByRole("option", { name: "Cluster" });
  fireEvent.pointerDown(cluster, { pointerType: "mouse" });
  fireEvent.click(cluster);
  expect(context.readonlyDataSourceList[0].redisType).toBe(
    DataSource_RedisType.CLUSTER
  );
  expect(context.adminDataSource.redisType).not.toBe(
    DataSource_RedisType.CLUSTER
  );
});

function mountCreate(engine: Engine) {
  render(
    <InstanceFormProvider>
      <Editor />
    </InstanceFormProvider>
  );
  act(() => context.setBasicInfo((previous) => ({ ...previous, engine })));
}
const hostingToggle = () =>
  screen.getByRole("button", { name: /^instance\.vendor\.self/ });
const hosting = (name: RegExp) => screen.getByRole("radio", { name });
const selectHosting = (name: RegExp) => {
  const toggle = hostingToggle();
  if (toggle.getAttribute("aria-expanded") === "false") {
    fireEvent.click(toggle);
  }
  fireEvent.click(hosting(name));
};

test("PostgreSQL reveals where it is hosted, with self-managed as the default", () => {
  mountCreate(Engine.POSTGRES);
  const toggle = hostingToggle();
  expect(toggle).toHaveAttribute("aria-expanded", "true");
  expect(toggle).toHaveTextContent("instance.vendor.generic");
  expect(
    screen.getByRole("radiogroup", { name: "instance.vendor.self" })
  ).toBeVisible();

  const generic = hosting(/^instance.vendor.generic/);
  const neon = hosting(/^instance.vendor.neon/);
  expect(generic).toHaveAttribute("aria-checked", "true");
  expect(generic.closest("label")).toHaveClass("min-h-9");
  expect(neon.closest("label")).toHaveClass("min-h-9");
  expect(generic.closest("label")?.textContent).toBe("instance.vendor.generic");
  expect(neon.closest("label")?.textContent).toBe("instance.vendor.neon");

  fireEvent.click(neon);
  expect(context.basicInfo.vendor).toBe(Vendor.NEON);
  expect(context.basicInfo.engine).toBe(Engine.POSTGRES);
  expect(toggle).toHaveAttribute("aria-expanded", "false");
  expect(toggle).toHaveTextContent("instance.vendor.neon");
  expect(
    screen.queryByRole("radiogroup", { name: "instance.vendor.self" })
  ).not.toBeInTheDocument();
});

test("does not ask where other engines are hosted", () => {
  mountCreate(Engine.MYSQL);
  expect(
    screen.queryByRole("button", { name: /^instance\.vendor\.self/ })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("radiogroup", { name: "instance.vendor.self" })
  ).not.toBeInTheDocument();
});

test("preserves separate self-managed and Neon manual connection drafts", () => {
  mountCreate(Engine.POSTGRES);
  const genericDraft = context.dataSourceEditState;
  selectHosting(/^instance.vendor.neon/);
  expect(context.adminDataSource.useSsl).toBe(true);
  expect(context.adminDataSource.verifyTlsCertificate).toBe(true);
  expect(context.adminDataSource.port).toBe("5432");
  act(() =>
    context.setDataSourceEditState((previous) => ({
      ...previous,
      dataSources: previous.dataSources.map((source) => ({
        ...source,
        host: "ep-example.neon.tech",
      })),
    }))
  );
  selectHosting(/^instance.vendor.generic/);
  expect(context.dataSourceEditState).toBe(genericDraft);
  selectHosting(/^instance.vendor.neon/);
  expect(context.adminDataSource.host).toBe("ep-example.neon.tech");
});

const NEON_STRING =
  "postgresql://app_owner:example-password@ep-example-123456.us-east-2.aws.neon.tech/storefront?sslmode=require";
const paste = (field: HTMLElement, text: string) =>
  fireEvent.paste(field, { clipboardData: { getData: () => text } });

test("Import from URL popover autofills Neon connection strings without keeping secrets and preserves verified TLS", () => {
  mountCreate(Engine.POSTGRES);
  selectHosting(/^instance.vendor.neon/);
  expect(
    screen.queryByLabelText(/instance\.neon\.manual\.connection-string/)
  ).not.toBeInTheDocument();

  const { field } = openImportUrlPopover();
  expect(field).toHaveAttribute("data-agent-sensitive");
  expect(field).toHaveAttribute("data-1p-ignore");
  paste(field, NEON_STRING);
  expect(context.adminDataSource).toMatchObject({
    host: "ep-example-123456.us-east-2.aws.neon.tech",
    port: "5432",
    username: "app_owner",
    database: "storefront",
    updatedPassword: "example-password",
    useSsl: true,
    verifyTlsCertificate: true,
  });
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
  expect(document.body.textContent).not.toContain("example-password");
  expect(mocks.pushNotification).toHaveBeenLastCalledWith({
    module: "bytebase",
    style: "SUCCESS",
    title: "instance.connection-string.filled",
  });
});

test("Neon Import from URL submits typed strings, rejects plaintext TLS or invalid input, and resets when switching hosting", () => {
  mountCreate(Engine.POSTGRES);
  selectHosting(/^instance.vendor.neon/);
  const { field } = openImportUrlPopover();
  // Typing does not autofill prematurely before submission.
  const partial = NEON_STRING.slice(0, NEON_STRING.indexOf("/storefront") + 2);
  fireEvent.change(field, { target: { value: partial } });
  expect(field).toHaveValue(partial);
  expect(context.adminDataSource.database).not.toBe("s");

  fireEvent.change(field, {
    target: {
      value: NEON_STRING.replace("sslmode=require", "sslmode=disable"),
    },
  });
  fireEvent.submit(field.closest("form") as HTMLFormElement);
  expect(field).toHaveAccessibleDescription(
    "instance.connection-string.invalid.neon"
  );

  fireEvent.change(field, { target: { value: NEON_STRING } });
  expect(field).not.toHaveAttribute("aria-invalid");
  fireEvent.click(screen.getByRole("button", { name: "common.import" }));
  expect(context.adminDataSource).toMatchObject({
    username: "app_owner",
    database: "storefront",
  });

  openImportUrlPopover();
  selectHosting(/^instance.vendor.generic/);
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
});

test("importing a pooled Neon connection string warns, surfaces the host error, and focuses the host field", async () => {
  mountCreate(Engine.POSTGRES);
  selectHosting(/^instance.vendor.neon/);
  const { field } = openImportUrlPopover();
  paste(
    field,
    NEON_STRING.replace("ep-example-123456", "ep-example-123456-pooler")
  );
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
  expect(
    screen.getByText("instance.validation.neon-pooled-host")
  ).toBeVisible();
  expect(mocks.pushNotification).toHaveBeenLastCalledWith({
    module: "bytebase",
    style: "WARN",
    title: "instance.connection-string.filled-neon-pooled",
  });
  await waitFor(() => {
    expect(screen.getByLabelText(/instance.hostname/)).toHaveFocus();
  });
});

test("rejects a pooled Neon host and asks for the direct endpoint", () => {
  mountCreate(Engine.POSTGRES);
  selectHosting(/^instance.vendor.neon/);
  act(() =>
    context.setDataSourceEditState((previous) => ({
      ...previous,
      dataSources: previous.dataSources.map((source) => ({
        ...source,
        host: "ep-example-123456-pooler.us-east-2.aws.neon.tech",
      })),
    }))
  );
  expect(
    screen.getByText("instance.validation.neon-pooled-host")
  ).toBeVisible();
  expect(context.getDataSourceErrors(context.adminDataSource).host).toBe(
    "neon-pooled-host"
  );
});

test("editing a Neon instance shows its connection settings without the hosting choices or URL import", () => {
  render(
    <InstanceFormProvider
      instance={create(InstanceSchema, {
        name: "instances/neon-storefront-main",
        title: "storefront / main",
        engine: Engine.POSTGRES,
        vendor: Vendor.NEON,
        state: State.ACTIVE,
        dataSources: [
          {
            id: "admin",
            type: DataSourceType.ADMIN,
            authenticationType: DataSource_AuthenticationType.PASSWORD,
            host: "ep-example-123456.us-east-2.aws.neon.tech",
            port: "5432",
            useSsl: true,
            verifyTlsCertificate: true,
          },
        ],
      })}
    >
      <Editor />
    </InstanceFormProvider>
  );
  expect(
    screen.queryByRole("button", { name: /^instance\.vendor\.self/ })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("radiogroup", { name: "instance.vendor.self" })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", {
      name: "instance.connection-string.trigger",
    })
  ).not.toBeInTheDocument();
  expect(screen.getByLabelText(/instance.hostname/)).toHaveValue(
    "ep-example-123456.us-east-2.aws.neon.tech"
  );
});

test("Neon TLS mode omits Disabled and allows toggling certificate verification off", async () => {
  mocks.hasSSL = true;
  try {
    const { DataSourceForm } = await vi.importActual<
      typeof import("./DataSourceForm")
    >("./DataSourceForm");
    const NeonWithOptions = () => {
      context = useInstanceFormContext();
      return (
        <>
          <InstanceFormBody />
          <DataSourceForm
            dataSource={context.adminDataSource}
            optionsOnly
            onDataSourceChange={(updated) =>
              context.setDataSourceEditState((previous) => ({
                ...previous,
                dataSources: previous.dataSources.map((source) =>
                  source.id === updated.id ? updated : source
                ),
              }))
            }
          />
        </>
      );
    };
    render(
      <InstanceFormProvider>
        <NeonWithOptions />
      </InstanceFormProvider>
    );
    act(() =>
      context.setBasicInfo((previous) => ({
        ...previous,
        engine: Engine.POSTGRES,
      }))
    );
    selectHosting(/^instance.vendor.neon/);

    const posture = screen.getByRole("combobox", {
      name: "data-source.ssl.posture.self",
    });
    expect(posture).toHaveTextContent("data-source.ssl.posture.tls");
    fireEvent.click(posture);
    expect(
      await screen.findByRole("option", { name: "data-source.ssl.posture.tls" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("option", {
        name: "data-source.ssl.posture.mutual-tls",
      })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "data-source.ssl.posture.disabled" })
    ).not.toBeInTheDocument();

    const verifySwitch = screen.getByRole("switch", {
      name: "data-source.ssl.verify-certificate",
    });
    expect(verifySwitch).toHaveAttribute("aria-checked", "true");
    fireEvent.click(verifySwitch);
    expect(context.adminDataSource.useSsl).toBe(true);
    expect(context.adminDataSource.verifyTlsCertificate).toBe(false);
  } finally {
    mocks.hasSSL = false;
  }
});

const openImportUrlPopover = () => {
  const trigger = screen.getByRole("button", {
    name: "instance.connection-string.trigger",
  });
  fireEvent.click(trigger);
  return {
    trigger,
    field: screen.getByLabelText("instance.connection-string.label"),
  };
};

test("Import from URL popover autofills MySQL, PostgreSQL, and MSSQL on paste and notifies via toast without keeping secrets", () => {
  mountCreate(Engine.MYSQL);
  let { field } = openImportUrlPopover();
  expect(field).toHaveAttribute("data-agent-sensitive");
  expect(field).toHaveAttribute("data-1p-ignore");
  paste(
    field,
    "mysql://admin:s3cr3t@mysql.example.com:25060/ignored_db?ssl-mode=VERIFY_IDENTITY"
  );
  expect(context.adminDataSource).toMatchObject({
    host: "mysql.example.com",
    port: "25060",
    username: "admin",
    updatedPassword: "s3cr3t",
    database: "",
    useSsl: true,
    verifyTlsCertificate: true,
  });
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
  expect(document.body.textContent).not.toContain("s3cr3t");
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
  expect(mocks.pushNotification).toHaveBeenLastCalledWith({
    module: "bytebase",
    style: "SUCCESS",
    title: "instance.connection-string.filled",
  });

  act(() =>
    context.setBasicInfo((previous) => ({
      ...previous,
      engine: Engine.POSTGRES,
    }))
  );
  ({ field } = openImportUrlPopover());
  paste(
    field,
    "postgresql://pguser:pgpass@pg.example.com:5433/appdb?sslmode=require"
  );
  expect(context.adminDataSource).toMatchObject({
    host: "pg.example.com",
    port: "5433",
    username: "pguser",
    updatedPassword: "pgpass",
    database: "appdb",
    useSsl: true,
    verifyTlsCertificate: false,
  });
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
  expect(mocks.pushNotification).toHaveBeenLastCalledWith({
    module: "bytebase",
    style: "SUCCESS",
    title: "instance.connection-string.filled",
  });

  act(() =>
    context.setBasicInfo((previous) => ({
      ...previous,
      engine: Engine.MSSQL,
    }))
  );
  ({ field } = openImportUrlPopover());
  paste(
    field,
    "Server=tcp:sql.example.com,1433;Initial Catalog=orders;User ID=sa;Password={your_password};Encrypt=True;TrustServerCertificate=False;"
  );
  expect(context.adminDataSource).toMatchObject({
    host: "sql.example.com",
    port: "1433",
    username: "sa",
    updatedPassword: "",
    useEmptyPassword: false,
    database: "orders",
    useSsl: true,
    verifyTlsCertificate: true,
  });
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
  expect(mocks.pushNotification).toHaveBeenLastCalledWith({
    module: "bytebase",
    style: "SUCCESS",
    title: "instance.connection-string.filled-missing-password",
  });
});

test("Import from URL supports form submission for typed strings and discards drafts on cancel", () => {
  mountCreate(Engine.MYSQL);
  let { field } = openImportUrlPopover();
  fireEvent.change(field, {
    target: { value: "mysql://typed:secret@typed.example.com:3307/app" },
  });
  fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
  expect(context.adminDataSource.host).not.toBe("typed.example.com");

  ({ field } = openImportUrlPopover());
  expect(field).toHaveValue("");
  fireEvent.change(field, {
    target: { value: "mysql://typed:secret@typed.example.com:3307/app" },
  });
  fireEvent.click(screen.getByRole("button", { name: "common.import" }));
  expect(context.adminDataSource).toMatchObject({
    host: "typed.example.com",
    port: "3307",
    username: "typed",
    updatedPassword: "secret",
  });
});

test("Import from URL flags engine mismatches, clears errors on fix, and resets when switching engines", () => {
  mountCreate(Engine.POSTGRES);
  const { field } = openImportUrlPopover();
  paste(field, "mysql://root:pass@mysql.example.com:3306/app");
  expect(field).toHaveValue("mysql://root:pass@mysql.example.com:3306/app");
  expect(field).toHaveAccessibleDescription(
    "instance.connection-string.invalid.engine-mismatch"
  );

  paste(field, "");
  fireEvent.change(field, { target: { value: "not-a-valid-string" } });
  fireEvent.submit(field.closest("form") as HTMLFormElement);
  expect(field).toHaveAccessibleDescription(
    "instance.connection-string.invalid.postgres"
  );

  fireEvent.change(field, {
    target: { value: "postgresql://pg.example.com:5432/app" },
  });
  expect(field).not.toHaveAttribute("aria-invalid");

  act(() =>
    context.setBasicInfo((previous) => ({
      ...previous,
      engine: Engine.MYSQL,
    }))
  );
  expect(
    screen.queryByLabelText("instance.connection-string.label")
  ).not.toBeInTheDocument();
  expect(screen.queryByRole("status")).not.toBeInTheDocument();
});

test("Import from URL trigger is hidden for unsupported engines and edit mode", () => {
  for (const engine of [
    Engine.DYNAMODB,
    Engine.BIGQUERY,
    Engine.SPANNER,
    Engine.COSMOSDB,
  ]) {
    mountCreate(engine);
    expect(
      screen.queryByRole("button", {
        name: "instance.connection-string.trigger",
      })
    ).not.toBeInTheDocument();
    cleanup();
  }

  render(
    <InstanceFormProvider
      instance={create(InstanceSchema, {
        name: "instances/prod-mysql",
        title: "Prod MySQL",
        engine: Engine.MYSQL,
        state: State.ACTIVE,
        dataSources: [
          {
            id: "admin",
            type: DataSourceType.ADMIN,
            authenticationType: DataSource_AuthenticationType.PASSWORD,
            host: "mysql.example.com",
            port: "3306",
          },
        ],
      })}
    >
      <Editor />
    </InstanceFormProvider>
  );
  expect(
    screen.queryByRole("button", {
      name: "instance.connection-string.trigger",
    })
  ).not.toBeInTheDocument();
});

test("importing an Oracle SID URL updates OracleConnectionFields, TLS controls, and focuses password when omitted", async () => {
  mocks.hasSSL = true;
  try {
    const { DataSourceForm } = await vi.importActual<
      typeof import("./DataSourceForm")
    >("./DataSourceForm");
    mocks.RealDataSourceForm = DataSourceForm;
    mountCreate(Engine.ORACLE);

    const { field } = openImportUrlPopover();
    paste(field, "oracle://system@ora.example.com:1522?SID=XE&SSL=true");

    expect(screen.getByRole("radio", { name: "instance.sid" })).toBeChecked();
    expect(
      screen.getByRole("textbox", { name: /^instance\.sid/ })
    ).toHaveValue("XE");
    expect(
      screen.getByRole("combobox", { name: "data-source.ssl.posture.self" })
    ).toHaveTextContent("data-source.ssl.posture.tls");
    await waitFor(() => {
      expect(screen.getByLabelText("common.password")).toHaveFocus();
    });
  } finally {
    mocks.RealDataSourceForm = null;
    mocks.hasSSL = false;
  }
});

test("importing MongoDB SRV and multi-host URLs updates schema radio, additional addresses, replica set, and auth database", async () => {
  try {
    const { DataSourceForm } = await vi.importActual<
      typeof import("./DataSourceForm")
    >("./DataSourceForm");
    mocks.RealDataSourceForm = DataSourceForm;
    mountCreate(Engine.MONGODB);

    let { field } = openImportUrlPopover();
    paste(
      field,
      "mongodb+srv://app:secret@cluster0.example.mongodb.net/storefront?authSource=admin"
    );

    expect(
      screen.getByRole("radio", { name: "mongodb+srv://" })
    ).toHaveAttribute("aria-checked", "true");
    expect(screen.getByDisplayValue("admin")).toBeInTheDocument();
    expect(context.adminDataSource).toMatchObject({
      host: "cluster0.example.mongodb.net",
      port: "",
      srv: true,
      authenticationDatabase: "admin",
      additionalAddresses: [],
    });

    ({ field } = openImportUrlPopover());
    paste(
      field,
      "mongodb://app:secret@n1.example.com:27017,n2.example.com:27018/storefront?replicaSet=rs0&authSource=authdb"
    );

    expect(screen.getByRole("radio", { name: "mongodb://" })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByDisplayValue("n2.example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("27018")).toBeInTheDocument();
    expect(screen.getByDisplayValue("rs0")).toBeInTheDocument();
    expect(screen.getByDisplayValue("authdb")).toBeInTheDocument();
  } finally {
    mocks.RealDataSourceForm = null;
  }
});

test("importing a Redis Sentinel URL without password switches mode, fills master and nodes, and focuses primary password", async () => {
  try {
    const { DataSourceForm, RedisSentinelFields } = await vi.importActual<
      typeof import("./DataSourceForm")
    >("./DataSourceForm");
    mocks.RealDataSourceForm = DataSourceForm;
    mocks.RealRedisSentinelFields = RedisSentinelFields;
    mountCreate(Engine.REDIS);

    const { field } = openImportUrlPopover();
    paste(
      field,
      "redis-sentinel://sentinel@s1.example.com:26379,s2.example.com:26380?master=mymaster"
    );

    expect(
      screen.getByRole("combobox", { name: "data-source.connection-type" })
    ).toHaveTextContent("Sentinel");
    expect(screen.getByDisplayValue("mymaster")).toBeInTheDocument();
    expect(screen.getByDisplayValue("s2.example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("26380")).toBeInTheDocument();
    expect(
      screen.getByLabelText("instance.master-password")
    ).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByLabelText("common.password")).toHaveFocus();
    });
    expect(screen.getByLabelText("instance.master-password")).not.toHaveFocus();
  } finally {
    mocks.RealDataSourceForm = null;
    mocks.RealRedisSentinelFields = null;
  }
});

test("importing a Databricks URL without PWD fills host and warehouseId, notifies missing token, and focuses token", async () => {
  try {
    const { DataSourceForm } = await vi.importActual<
      typeof import("./DataSourceForm")
    >("./DataSourceForm");
    mocks.RealDataSourceForm = DataSourceForm;
    mountCreate(Engine.DATABRICKS);

    const { field } = openImportUrlPopover();
    paste(
      field,
      "jdbc:databricks://dbc.example.com:443/default;transportMode=http;httpPath=/sql/1.0/warehouses/wh123"
    );

    expect(context.adminDataSource).toMatchObject({
      host: "https://dbc.example.com",
      port: "",
      warehouseId: "wh123",
      updatedToken: "",
    });
    expect(screen.getByDisplayValue("wh123")).toBeInTheDocument();
    expect(mocks.pushNotification).toHaveBeenLastCalledWith({
      module: "bytebase",
      style: "SUCCESS",
      title: "instance.connection-string.filled-missing-token",
    });
    await waitFor(() => {
      expect(screen.getByLabelText("common.token")).toHaveFocus();
    });
  } finally {
    mocks.RealDataSourceForm = null;
  }
});

test("Import from URL calls out cross-engine mismatches for newly supported engines", () => {
  mountCreate(Engine.MYSQL);
  let { field } = openImportUrlPopover();
  paste(field, "mongodb+srv://user:pass@cluster0.example.mongodb.net/mydb");
  expect(field).toHaveAccessibleDescription(
    "instance.connection-string.invalid.engine-mismatch"
  );
  fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

  act(() =>
    context.setBasicInfo((previous) => ({
      ...previous,
      engine: Engine.CLICKHOUSE,
    }))
  );
  ({ field } = openImportUrlPopover());
  paste(field, "redis://default:pass@redis.example.com:6379");
  expect(field).toHaveAccessibleDescription(
    "instance.connection-string.invalid.engine-mismatch"
  );
});
