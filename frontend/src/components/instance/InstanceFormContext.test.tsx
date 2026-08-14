import { create } from "@bufbuild/protobuf";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSource_AuthenticationType,
  DataSource_AWSCredentialSchema,
  DataSourceSchema,
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { unknownInstance } from "@/types/v1/instance";
import type { EditDataSource } from "./common";
import { wrapEditDataSource } from "./common";
import {
  InstanceFormProvider,
  useInstanceFormContext,
} from "./InstanceFormContext";

const mocks = vi.hoisted(() => ({
  hasInstancePermission: vi.fn(() => true),
}));

vi.mock("./permission", () => ({
  hasInstancePermission: mocks.hasInstancePermission,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("monaco-editor", () => ({}));

vi.mock(
  "@codingame/monaco-vscode-editor-api/vscode/src/vs/editor/standalone/browser/standalone-tokens.css",
  () => ({})
);

vi.mock("@/types", () => ({
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
  pushNotification: vi.fn(),
}));

let mockEnvironmentList: { id: string; name: string }[] = [];

vi.mock("@/stores/app", () => {
  const appState = () => ({
    createDataSource: vi.fn(),
    createInstance: vi.fn(),
    updateDataSource: vi.fn(),
    getEnvironmentByName: (name: string) => ({ name }),
    hasInstanceFeature: () => false,
    instanceLicenseCount: () => 1,
    activatedInstanceCount: () => 0,
    currentPlan: () => 1,
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
  calcUpdateMask: () => [],
  convertKVListToLabels: (list: { key: string; value: string }[]) =>
    Object.fromEntries(list.map(({ key, value }) => [key, value])),
  convertLabelsToKVList: (labels: Record<string, string>) =>
    Object.entries(labels).map(([key, value]) => ({ key, value })),
  hasWorkspacePermissionV2: () => true,
  instanceV1HasExtraParameters: () => false,
  instanceV1HasSSH: () => false,
  instanceV1HasSSL: () => false,
  isValidSpannerDataSource: (ds: { projectId: string; instanceId: string }) =>
    ds.projectId !== "" && ds.instanceId !== "",
  isValidBigQueryDataSource: (ds: { projectId: string }) =>
    ds.projectId !== "",
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

const Probe = () => {
  const ctx = useInstanceFormContext();
  return (
    <div
      data-title={ctx.basicInfo.title}
      data-name={ctx.basicInfo.name}
      data-parent={ctx.parent}
      data-host={ctx.adminDataSource.host}
      data-environment={ctx.basicInfo.environment}
      data-value-changed={String(ctx.valueChanged)}
      data-is-editing={String(ctx.isEditing)}
      data-can-update={String(ctx.hasPermission("bb.instances.update"))}
    />
  );
};

const renderIntoContainer = () => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: async (nextElement: ReactElement) => {
      await act(async () => {
        root.render(nextElement);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
    },
  };
};

describe("InstanceFormProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasInstancePermission.mockReturnValue(true);
    mockEnvironmentList = [];
    vi.useRealTimers();
  });

  test("uses project ownership for create names and permissions", async () => {
    const project = create(ProjectSchema, { name: "projects/app" });
    const harness = renderIntoContainer();

    await harness.render(
      <InstanceFormProvider parent="projects/app" project={project}>
        <Probe />
      </InstanceFormProvider>
    );

    const probe = harness.container.firstElementChild as HTMLElement;
    expect(probe.dataset.name).toBe("projects/app/instances/-");
    expect(probe.dataset.parent).toBe("projects/app");
    expect(probe.dataset.canUpdate).toBe("true");
    expect(mocks.hasInstancePermission).toHaveBeenCalledWith(
      project,
      "bb.instances.update"
    );

    harness.unmount();
  });

  test("selects the first environment by default when creating an instance", async () => {
    mockEnvironmentList = [{ id: "dev", name: "environments/dev" }];
    const harness = renderIntoContainer();

    await harness.render(
      <InstanceFormProvider>
        <Probe />
      </InstanceFormProvider>
    );

    const probe = harness.container.firstElementChild as HTMLElement;
    expect(probe.dataset.environment).toBe("environments/dev");

    harness.unmount();
  });

  test("refreshes form state when an unknown instance is replaced by the fetched instance", async () => {
    const fetchedInstance = create(InstanceSchema, {
      name: "instances/prod",
      title: "Production",
      engine: Engine.POSTGRES,
      environment: "environments/prod",
      dataSources: [
        create(DataSourceSchema, {
          id: "admin",
          type: DataSourceType.ADMIN,
          host: "prod.example.com",
          port: "5432",
        }),
      ],
    });
    const harness = renderIntoContainer();

    await harness.render(
      <InstanceFormProvider instance={unknownInstance()}>
        <Probe />
      </InstanceFormProvider>
    );
    await harness.render(
      <InstanceFormProvider instance={fetchedInstance}>
        <Probe />
      </InstanceFormProvider>
    );

    const probe = harness.container.firstElementChild as HTMLElement;
    expect(probe.dataset.title).toBe("Production");
    expect(probe.dataset.host).toBe("prod.example.com");

    harness.unmount();
  });

  test("does not mark an archived instance as changed after restore", async () => {
    vi.useFakeTimers();
    const archivedInstance = create(InstanceSchema, {
      name: "instances/prod",
      state: State.DELETED,
      title: "Production",
      engine: Engine.POSTGRES,
      environment: "environments/prod",
      dataSources: [
        create(DataSourceSchema, {
          id: "admin",
          type: DataSourceType.ADMIN,
          host: "prod.example.com",
          port: "5432",
        }),
      ],
    });
    const restoredInstance = create(InstanceSchema, {
      ...archivedInstance,
      state: State.ACTIVE,
    });
    const harness = renderIntoContainer();

    await harness.render(
      <InstanceFormProvider instance={archivedInstance}>
        <Probe />
      </InstanceFormProvider>
    );
    await act(async () => {
      vi.advanceTimersByTime(350);
    });
    await harness.render(
      <InstanceFormProvider instance={restoredInstance}>
        <Probe />
      </InstanceFormProvider>
    );
    await act(async () => {
      vi.advanceTimersByTime(350);
    });

    const probe = harness.container.firstElementChild as HTMLElement;
    expect(probe.dataset.valueChanged).toBe("false");
    expect(probe.dataset.isEditing).toBe("false");

    harness.unmount();
    vi.useRealTimers();
  });

  // Regression test for BYT-9696: setting missingFeature (e.g. saving a
  // read-only connection on an unlicensed instance) must surface the
  // FeatureModal paywall instead of failing silently.
  test("renders the FeatureModal when missingFeature is set", async () => {
    const instance = create(InstanceSchema, {
      name: "instances/prod",
      title: "Production",
      engine: Engine.POSTGRES,
      environment: "environments/prod",
      dataSources: [
        create(DataSourceSchema, {
          id: "admin",
          type: DataSourceType.ADMIN,
          host: "prod.example.com",
          port: "5432",
        }),
      ],
    });

    const MissingFeatureProbe = () => {
      const ctx = useInstanceFormContext();
      return (
        <button
          data-testid="set-missing-feature"
          type="button"
          onClick={() =>
            ctx.setMissingFeature(
              PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION
            )
          }
        />
      );
    };

    const harness = renderIntoContainer();
    await harness.render(
      <InstanceFormProvider instance={instance}>
        <MissingFeatureProbe />
      </InstanceFormProvider>
    );

    expect(
      harness.container.querySelector("[data-testid='feature-modal']")
    ).toBeNull();

    const trigger = harness.container.querySelector(
      "[data-testid='set-missing-feature']"
    ) as HTMLButtonElement;
    await act(async () => {
      trigger.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const modal = harness.container.querySelector(
      "[data-testid='feature-modal']"
    ) as HTMLElement;
    expect(modal).not.toBeNull();
    expect(modal.dataset.feature).toBe(
      String(PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION)
    );
    expect(modal.dataset.instance).toBe("instances/prod");

    const close = harness.container.querySelector(
      "[data-testid='feature-modal-close']"
    ) as HTMLButtonElement;
    await act(async () => {
      close.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(
      harness.container.querySelector("[data-testid='feature-modal']")
    ).toBeNull();

    harness.unmount();
  });

  describe("checkDataSource AWS region requirement", () => {
    const awsDataSource = (region: string, withCredential: boolean) => {
      const ds = wrapEditDataSource(
        create(DataSourceSchema, {
          id: "admin",
          type: DataSourceType.ADMIN,
          authenticationType: DataSource_AuthenticationType.AWS_RDS_IAM,
          region,
        })
      );
      if (withCredential) {
        ds.iamExtension = {
          case: "awsCredential",
          value: create(DataSource_AWSCredentialSchema, {}),
        };
      }
      return ds;
    };

    const CheckDataSourceProbe = () => {
      const ctx = useInstanceFormContext();
      return (
        <div
          data-credential-no-region={String(
            ctx.checkDataSource([awsDataSource("", true)])
          )}
          data-credential-with-region={String(
            ctx.checkDataSource([awsDataSource("us-east-1", true)])
          )}
          data-default-chain-no-region={String(
            ctx.checkDataSource([awsDataSource("", false)])
          )}
        />
      );
    };

    const instanceOfEngine = (engine: Engine) =>
      create(InstanceSchema, {
        name: "instances/aws-check",
        title: "AWS check",
        engine,
        environment: "environments/prod",
        dataSources: [
          create(DataSourceSchema, { id: "admin", type: DataSourceType.ADMIN }),
        ],
      });

    test("DynamoDB requires a region only with a specific credential", async () => {
      const harness = renderIntoContainer();

      await harness.render(
        <InstanceFormProvider instance={instanceOfEngine(Engine.DYNAMODB)}>
          <CheckDataSourceProbe />
        </InstanceFormProvider>
      );

      const probe = harness.container.firstElementChild as HTMLElement;
      expect(probe.dataset.credentialNoRegion).toBe("false");
      expect(probe.dataset.credentialWithRegion).toBe("true");
      expect(probe.dataset.defaultChainNoRegion).toBe("true");

      harness.unmount();
    });

    test("other engines require a region for AWS IAM even on the default credential chain", async () => {
      const harness = renderIntoContainer();

      await harness.render(
        <InstanceFormProvider instance={instanceOfEngine(Engine.POSTGRES)}>
          <CheckDataSourceProbe />
        </InstanceFormProvider>
      );

      const probe = harness.container.firstElementChild as HTMLElement;
      expect(probe.dataset.credentialNoRegion).toBe("false");
      expect(probe.dataset.credentialWithRegion).toBe("true");
      expect(probe.dataset.defaultChainNoRegion).toBe("false");

      harness.unmount();
    });
  });

  // The server refuses to carry a stored Kerberos keytab to a destination the
  // caller moved, so the form has to fail the data source before it offers to
  // save it — otherwise the refusal arrives as a connection failure.
  describe("checkDataSource Kerberos keytab resupply", () => {
    const storedDataSource = create(DataSourceSchema, {
      id: "admin",
      type: DataSourceType.ADMIN,
      host: "hive.example.com",
      port: "10000",
      saslConfig: {
        mechanism: {
          case: "krbConfig",
          value: {
            primary: "bytebase",
            realm: "EXAMPLE.COM",
            kdcHost: "kdc.example.com",
            kdcPort: "88",
          },
        },
      },
    });

    const editedDataSource = (edit: (ds: EditDataSource) => void) => {
      const ds = wrapEditDataSource(storedDataSource);
      edit(ds);
      return ds;
    };

    const movedHost = () =>
      editedDataSource((ds) => {
        ds.host = "hive-2.example.com";
      });

    const KeytabProbe = () => {
      const ctx = useInstanceFormContext();
      return (
        <div
          data-username-only={String(
            ctx.checkDataSource([
              editedDataSource((ds) => {
                ds.username = "hive";
              }),
            ])
          )}
          data-moved-host={String(ctx.checkDataSource([movedHost()]))}
          data-moved-kdc={String(
            ctx.checkDataSource([
              editedDataSource((ds) => {
                if (ds.saslConfig?.mechanism?.case === "krbConfig") {
                  ds.saslConfig.mechanism.value.kdcHost = "kdc-2.example.com";
                }
              }),
            ])
          )}
          data-moved-with-keytab={String(
            ctx.checkDataSource([
              editedDataSource((ds) => {
                ds.host = "hive-2.example.com";
                if (ds.saslConfig?.mechanism?.case === "krbConfig") {
                  ds.saslConfig.mechanism.value.keytab = new Uint8Array([5, 2]);
                }
              }),
            ])
          )}
          data-needs-resupply={String(ctx.needsKeytabResupply(movedHost()))}
        />
      );
    };

    test("a moved destination fails the data source until the keytab is uploaded again", async () => {
      const harness = renderIntoContainer();

      await harness.render(
        <InstanceFormProvider
          instance={create(InstanceSchema, {
            name: "instances/hive",
            title: "Hive",
            engine: Engine.HIVE,
            environment: "environments/prod",
            dataSources: [storedDataSource],
          })}
        >
          <KeytabProbe />
        </InstanceFormProvider>
      );

      const probe = harness.container.firstElementChild as HTMLElement;
      expect(probe.dataset.usernameOnly).toBe("true");
      expect(probe.dataset.movedHost).toBe("false");
      expect(probe.dataset.movedKdc).toBe("false");
      expect(probe.dataset.movedWithKeytab).toBe("true");
      expect(probe.dataset.needsResupply).toBe("true");

      harness.unmount();
    });

    const ResupplyProbe = ({ build }: { build: () => EditDataSource }) => {
      const ctx = useInstanceFormContext();
      return <div data-needs-resupply={String(ctx.needsKeytabResupply(build()))} />;
    };

    const renderWithStored = async (
      stored: DataSource[],
      build: () => EditDataSource
    ) => {
      const harness = renderIntoContainer();
      await harness.render(
        <InstanceFormProvider
          instance={create(InstanceSchema, {
            name: "instances/hive",
            title: "Hive",
            engine: Engine.HIVE,
            environment: "environments/prod",
            dataSources: stored,
          })}
        >
          <ResupplyProbe build={build} />
        </InstanceFormProvider>
      );
      const probe = harness.container.firstElementChild as HTMLElement;
      const result = probe.dataset.needsResupply;
      harness.unmount();
      return result;
    };

    // The comparison runs on what the form would send, not on the edit state,
    // because that is what the server merges and then compares.
    test("the port the form fills in for the engine counts as a move", async () => {
      const withoutPort = create(DataSourceSchema, {
        ...storedDataSource,
        port: "",
      });
      expect(
        await renderWithStored([withoutPort], () =>
          wrapEditDataSource(withoutPort)
        )
      ).toBe("true");
    });

    test("an SSH tunnel the form drops for the engine counts as a move", async () => {
      const withTunnel = create(DataSourceSchema, {
        ...storedDataSource,
        sshHost: "bastion.example.com",
        sshPort: "22",
      });
      expect(
        await renderWithStored([withTunnel], () =>
          wrapEditDataSource(withTunnel)
        )
      ).toBe("true");
    });

    // checkDataSource runs over every data source, so the stored record has to
    // be the one with the same ID rather than whichever comes first.
    test("each data source is compared against its own stored record", async () => {
      const readonlyDataSource = create(DataSourceSchema, {
        ...storedDataSource,
        id: "readonly",
        type: DataSourceType.READ_ONLY,
        host: "hive-replica.example.com",
      });
      expect(
        await renderWithStored([storedDataSource, readonlyDataSource], () =>
          wrapEditDataSource(readonlyDataSource)
        )
      ).toBe("false");
      expect(
        await renderWithStored([storedDataSource, readonlyDataSource], () => {
          const ds = wrapEditDataSource(readonlyDataSource);
          ds.host = "hive-replica-2.example.com";
          return ds;
        })
      ).toBe("true");
    });
  });
});
