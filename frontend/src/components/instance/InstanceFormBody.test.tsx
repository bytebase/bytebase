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
  DataSourceForm: ({
    dataSource,
    authOnly,
    hideAuthentication,
    onDataSourceChange,
  }: {
    dataSource: EditDataSource;
    authOnly?: boolean;
    hideAuthentication?: boolean;
    onDataSourceChange: (value: EditDataSource) => void;
  }) =>
    authOnly ? (
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
    ),
  RedisSentinelFields: () => null,
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

afterEach(cleanup);

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

  expect(headingRow?.contains(askAgentButton)).toBe(true);
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
const hosting = (name: RegExp) => screen.getByRole("radio", { name });

test("PostgreSQL reveals where it is hosted, with self-managed as the default", () => {
  mountCreate(Engine.POSTGRES);
  expect(hosting(/^instance.vendor.generic/)).toHaveAttribute(
    "aria-checked",
    "true"
  );
  fireEvent.click(hosting(/^instance.vendor.neon/));
  expect(context.basicInfo.vendor).toBe(Vendor.NEON);
  expect(context.basicInfo.engine).toBe(Engine.POSTGRES);
});

test("does not ask where other engines are hosted", () => {
  mountCreate(Engine.MYSQL);
  expect(
    screen.queryByRole("radiogroup", { name: "instance.vendor.self" })
  ).not.toBeInTheDocument();
});

test("preserves separate self-managed and Neon manual connection drafts", () => {
  mountCreate(Engine.POSTGRES);
  const genericDraft = context.dataSourceEditState;
  fireEvent.click(hosting(/^instance.vendor.neon/));
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
  fireEvent.click(hosting(/^instance.vendor.generic/));
  expect(context.dataSourceEditState).toBe(genericDraft);
  fireEvent.click(hosting(/^instance.vendor.neon/));
  expect(context.adminDataSource.host).toBe("ep-example.neon.tech");
});

const NEON_STRING =
  "postgresql://app_owner:example-password@ep-example-123456.us-east-2.aws.neon.tech/storefront?sslmode=require";
const paste = (field: HTMLElement, text: string) =>
  fireEvent.paste(field, { clipboardData: { getData: () => text } });

test("a pasted Neon connection string fills the connection fields and is not kept", () => {
  mountCreate(Engine.POSTGRES);
  fireEvent.click(hosting(/^instance.vendor.neon/));
  const field = screen.getByLabelText(/instance.neon.manual.connection-string/);
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
  expect(field).toHaveValue("");
  expect(field).toHaveAccessibleDescription(
    "instance.neon.manual.connection-string-filled"
  );
  expect(screen.getByText("instance.neon.manual.tls-note")).toBeVisible();
});

test("a connection string typed by hand is read when the field is left, not per keystroke", () => {
  mountCreate(Engine.POSTGRES);
  fireEvent.click(hosting(/^instance.vendor.neon/));
  const field = screen.getByLabelText(/instance.neon.manual.connection-string/);
  // This prefix already parses, with a one-letter database name.
  const partial = NEON_STRING.slice(0, NEON_STRING.indexOf("/storefront") + 2);
  fireEvent.change(field, { target: { value: partial } });
  expect(field).toHaveValue(partial);
  expect(context.adminDataSource.database).not.toBe("s");
  expect(
    screen.queryByText("instance.neon.manual.connection-string-invalid")
  ).toBeNull();
  fireEvent.change(field, { target: { value: NEON_STRING } });
  expect(field).toHaveValue(NEON_STRING);
  fireEvent.blur(field);
  expect(field).toHaveValue("");
  expect(context.adminDataSource).toMatchObject({
    username: "app_owner",
    database: "storefront",
  });
});

test("text that is not a connection string is reported once the field is left and stays to be corrected", () => {
  mountCreate(Engine.POSTGRES);
  fireEvent.click(hosting(/^instance.vendor.neon/));
  const field = screen.getByLabelText(/instance.neon.manual.connection-string/);
  fireEvent.change(field, { target: { value: "not a connection string" } });
  expect(
    screen.queryByText("instance.neon.manual.connection-string-invalid")
  ).toBeNull();
  fireEvent.blur(field);
  expect(field).toHaveAccessibleDescription(
    "instance.neon.manual.connection-string-invalid"
  );
  expect(field).toHaveValue("not a connection string");
});

test("a pasted pooled connection string says so beside the field until the host is corrected", () => {
  mountCreate(Engine.POSTGRES);
  fireEvent.click(hosting(/^instance.vendor.neon/));
  const field = screen.getByLabelText(/instance.neon.manual.connection-string/);
  paste(
    field,
    NEON_STRING.replace("ep-example-123456", "ep-example-123456-pooler")
  );
  expect(field).toHaveAccessibleDescription(
    "instance.neon.manual.connection-string-pooled"
  );
  expect(
    screen.getByText("instance.validation.neon-pooled-host")
  ).toBeVisible();
  act(() =>
    context.setDataSourceEditState((previous) => ({
      ...previous,
      dataSources: previous.dataSources.map((source) => ({
        ...source,
        host: "ep-example-123456.us-east-2.aws.neon.tech",
      })),
    }))
  );
  expect(field).toHaveAccessibleDescription(
    "instance.neon.manual.connection-string-filled"
  );
});

test("rejects a pooled Neon host and asks for the direct endpoint", () => {
  mountCreate(Engine.POSTGRES);
  fireEvent.click(hosting(/^instance.vendor.neon/));
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

test("editing a Neon instance shows its connection settings without the hosting or setup choices", () => {
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
    screen.queryByRole("radiogroup", { name: "instance.vendor.self" })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByLabelText(/instance.neon.manual.connection-string/)
  ).not.toBeInTheDocument();
  expect(screen.getByLabelText(/instance.hostname/)).toHaveValue(
    "ep-example-123456.us-east-2.aws.neon.tech"
  );
  expect(screen.getByText("instance.neon.manual.tls-note")).toBeVisible();
});
