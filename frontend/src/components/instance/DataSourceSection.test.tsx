import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSourceSchema,
  DataSource_AuthenticationType,
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import type { DataSourceEditState, EditDataSource } from "./common";
import { DataSourceSection } from "./DataSourceSection";
import { validateDataSource } from "./validation";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  context: undefined as Record<string, unknown> | undefined,
  checkDataSource: vi.fn((_dataSources: EditDataSource[]) => true),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/lib/i18n", () => ({
  default: { t: (key: string) => key },
}));

vi.mock("./InstanceFormContext", () => ({
  useInstanceFormContext: () => mocks.context,
}));

vi.mock("./CreateDataSourceExample", () => ({ CreateDataSourceExample: () => null }));

vi.mock("./DataSourceForm", () => ({
  DataSourceForm: ({ dataSource }: { dataSource: EditDataSource }) => (
    <div data-testid="data-source-form" data-id={dataSource.id} />
  ),
}));

vi.mock("@/components/ui/alert", () => ({
  Alert: ({ description }: { description: React.ReactNode }) => (
    <div>{description}</div>
  ),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    className,
    disabled,
    onClick,
    type = "button",
  }: {
    children: React.ReactNode;
    className?: string;
    disabled?: boolean;
    onClick?: () => void;
    type?: "button" | "submit" | "reset";
  }) => (
    <button
      className={className}
      disabled={disabled}
      onClick={onClick}
      type={type}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ deleteDataSource: vi.fn() }) },
}));

vi.mock("@/types", () => ({
  DATASOURCE_READONLY_USER_NAME: "bytebase_readonly",
  unknownDataSource: () => create(DataSourceSchema, { authenticationType: DataSource_AuthenticationType.PASSWORD }),
}));

const editDataSource = (
  id: string,
  type: DataSourceType,
  host: string
): EditDataSource => ({
  ...create(DataSourceSchema, { id, type, host, port: "10000" }),
  pendingCreate: false,
  updatedPassword: "",
  updatedMasterPassword: "",
  updatedToken: "",
});

const adminDataSource = editDataSource(
  "admin",
  DataSourceType.ADMIN,
  "hive.example.com"
);
const readonlyDataSource = editDataSource(
  "readonly",
  DataSourceType.READ_ONLY,
  "hive-replica.example.com"
);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.checkDataSource.mockImplementation(() => true);
  mocks.context = {
    instance: create(InstanceSchema, { name: "instances/hive" }),
    isCreating: false,
    allowEdit: true,
    basicInfo: create(InstanceSchema, { engine: Engine.HIVE }),
    dataSourceEditState: {
      dataSources: [adminDataSource, readonlyDataSource],
      editingDataSourceId: "admin",
    },
    setDataSourceEditState: vi.fn(),
    adminDataSource,
    editingDataSource: adminDataSource,
    readonlyDataSourceList: [readonlyDataSource],
    hasReadOnlyDataSource: true,
    hasPermission: () => true,
    checkDataSource: mocks.checkDataSource,
    isEditing: true,
  };
});

const renderSection = async () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  await act(async () => {
    root.render(<DataSourceSection />);
  });
  const tabs = Array.from(container.querySelectorAll("button")).filter(
    (button) =>
      button.textContent?.startsWith("common.admin") ||
      button.textContent?.startsWith("common.read-only")
  );
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
    tabs: tabs.map((button) => ({
      label: button.textContent ?? "",
      marked: button.querySelector(".text-error") !== null,
      described:
        button.querySelector(".text-error")?.textContent?.includes(
          "instance.open-data-source"
        ) ?? false,
    })),
  };
};

// A data source that fails the check disables Update from any tab, but only
// the open tab renders its own errors. Without the marker the operator sees a
// dead button and nothing else — the silent failure this whole gate replaces.
describe("DataSourceSection incomplete markers", () => {
  test("marks no tab while every data source passes the check", async () => {
    const { tabs, unmount } = await renderSection();

    expect(tabs).toHaveLength(2);
    expect(tabs.every((tab) => tab.marked)).toBe(false);

    unmount();
  });

  test("marks the failing tab, and only that one", async () => {
    mocks.checkDataSource.mockImplementation(
      ([ds]: EditDataSource[]) => ds.id !== "readonly"
    );

    const { tabs, unmount } = await renderSection();

    expect(tabs.map((tab) => tab.marked)).toEqual([false, true]);
    // Every data source is asked about itself, never about the whole list.
    for (const call of mocks.checkDataSource.mock.calls) {
      expect(call[0]).toHaveLength(1);
    }

    unmount();
  });

  // Nothing to save means no Update button to explain.
  test("marks nothing while the form is unedited", async () => {
    mocks.checkDataSource.mockImplementation(() => false);
    mocks.context = { ...mocks.context, isEditing: false };

    const { tabs, unmount } = await renderSection();

    expect(tabs.some((tab) => tab.marked)).toBe(false);

    unmount();
  });

  // A bare asterisk in an error color says a tab is wrong to whoever can see
  // both, and nothing to anyone else.
  test("the marker carries text, not just a color", async () => {
    mocks.checkDataSource.mockImplementation(() => false);

    const { tabs, unmount } = await renderSection();

    expect(tabs.every((tab) => tab.described)).toBe(true);

    unmount();
  });
});


test.each([Engine.SPANNER, Engine.BIGQUERY])(
  "new read-only GCP connections use IAM and require SaaS credentials for engine %s",
  async (engine) => {
    let state: DataSourceEditState = {dataSources: [adminDataSource], editingDataSourceId: "admin"};
    mocks.context = {...mocks.context,
      basicInfo: create(InstanceSchema, {engine}),
      adminDataSource: {...adminDataSource, projectId: "valid-project", instanceId: "valid-instance"},
      setDataSourceEditState: (update: (previous: DataSourceEditState) => DataSourceEditState) => {state = update(state);},
    };
    const {container, unmount} = await renderSection();
    try {
      const add = container.querySelector("svg.lucide-plus")?.closest("button");
      expect(add).toBeTruthy();
      await act(async () => {add!.click();});
      expect(state.dataSources).toHaveLength(2);
      const draft = state.dataSources[1];
      expect(draft.authenticationType).toBe(DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM);
      expect(draft.type).toBe(DataSourceType.READ_ONLY);
      expect(validateDataSource(draft, {engine, isSaaSMode: true}).iamExtension).toBe("specific-credential");
    } finally {unmount();}
  }
);
