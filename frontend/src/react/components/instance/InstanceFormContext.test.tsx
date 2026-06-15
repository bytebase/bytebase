import { create } from "@bufbuild/protobuf";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSourceSchema,
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { unknownInstance } from "@/types/v1/instance";
import {
  InstanceFormProvider,
  useInstanceFormContext,
} from "./InstanceFormContext";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/i18n", () => ({
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

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/react/stores/app", () => {
  const appState = () => ({
    createDataSource: vi.fn(),
    createInstance: vi.fn(),
    updateDataSource: vi.fn(),
    getEnvironmentByName: (name: string) => ({ name }),
    hasInstanceFeature: () => false,
    instanceLicenseCount: () => 1,
    activatedInstanceCount: () => 0,
    currentPlan: () => 1,
    environmentList: [],
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
  isValidSpannerHost: (host: string) => host !== "",
}));

vi.mock("@/utils/connect", () => ({
  extractGrpcErrorMessage: (error: unknown) =>
    error instanceof Error ? error.message : String(error),
}));

vi.mock("@/react/components/ui/feature-modal", () => ({
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
      data-host={ctx.adminDataSource.host}
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
});
