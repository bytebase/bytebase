import { create } from "@bufbuild/protobuf";
import { type ReactElement, useState } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_CloudSQLIPType,
  DataSource_GCPCredentialSchema,
  DataSourceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  CredentialSourceForm,
  offeredCloudSQLIPTypes,
  showsCloudSQLIPType,
} from "./CredentialSourceForm";
import type { EditDataSource } from "./common";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  Trans: ({ i18nKey }: { i18nKey: string }) => <>{i18nKey}</>,
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

const serverState = vi.hoisted(() => ({ isSaaSMode: false }));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({
    isSaaSMode: serverState.isSaaSMode,
  }),
}));

const makeDataSource = (
  authenticationType: DataSource_AuthenticationType
): EditDataSource => ({
  ...create(DataSourceSchema, {
    id: "admin",
    authenticationType,
  }),
  pendingCreate: false,
  updatedPassword: "",
  updatedMasterPassword: "",
  updatedToken: "",
});

function render(element: ReactElement) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });

  return {
    container,
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
}

describe("CredentialSourceForm", () => {
  beforeEach(() => {
    serverState.isSaaSMode = false;
  });

  test.each([
    [
      DataSource_AuthenticationType.AZURE_IAM,
      "instance.iam-extension.default-credential.azure",
    ],
    [
      DataSource_AuthenticationType.AWS_RDS_IAM,
      "instance.iam-extension.default-credential.aws",
    ],
    [
      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
      "instance.iam-extension.default-credential.gcp",
    ],
  ])(
    "renders default credential help through i18n for authentication type %s",
    (authenticationType, expectedKey) => {
      const { container, unmount } = render(
        <CredentialSourceForm
          dataSource={makeDataSource(authenticationType)}
          engine={Engine.POSTGRES}
          allowEdit={true}
          onDataSourceChange={vi.fn()}
        />
      );

      expect(container.textContent).toContain(expectedKey);

      unmount();
    }
  );

  test("shows the Cloud SQL IP selector only for Cloud SQL MySQL/Postgres IAM", () => {
    const { GOOGLE_CLOUD_SQL_IAM, PASSWORD } = DataSource_AuthenticationType;
    expect(showsCloudSQLIPType(Engine.MYSQL, GOOGLE_CLOUD_SQL_IAM)).toBe(true);
    expect(showsCloudSQLIPType(Engine.POSTGRES, GOOGLE_CLOUD_SQL_IAM)).toBe(
      true
    );
    // Spanner/BigQuery reuse the Google IAM auth type but do not use cloudsqlconn.
    expect(showsCloudSQLIPType(Engine.SPANNER, GOOGLE_CLOUD_SQL_IAM)).toBe(
      false
    );
    expect(showsCloudSQLIPType(Engine.BIGQUERY, GOOGLE_CLOUD_SQL_IAM)).toBe(
      false
    );
    expect(showsCloudSQLIPType(Engine.POSTGRES, PASSWORD)).toBe(false);
  });

  test("renders the selector for Postgres GCP IAM but not for Spanner", () => {
    const shown = render(
      <CredentialSourceForm
        dataSource={makeDataSource(
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        )}
        engine={Engine.POSTGRES}
        allowEdit={true}
        onDataSourceChange={vi.fn()}
      />
    );
    expect(shown.container.textContent).toContain(
      "instance.cloud-sql-ip-type.label"
    );
    shown.unmount();

    const spanner = render(
      <CredentialSourceForm
        dataSource={makeDataSource(
          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        )}
        engine={Engine.SPANNER}
        allowEdit={true}
        onDataSourceChange={vi.fn()}
      />
    );
    expect(spanner.container.textContent).not.toContain(
      "instance.cloud-sql-ip-type.label"
    );
    spanner.unmount();
  });

  test("offers Public and Private only, grandfathering an existing PSC value", () => {
    const { PUBLIC, PRIVATE, PSC, CLOUD_SQL_IP_TYPE_UNSPECIFIED } =
      DataSource_CloudSQLIPType;
    expect(offeredCloudSQLIPTypes(CLOUD_SQL_IP_TYPE_UNSPECIFIED)).toEqual([
      PUBLIC,
      PRIVATE,
    ]);
    expect(offeredCloudSQLIPTypes(PRIVATE)).toEqual([PUBLIC, PRIVATE]);
    expect(offeredCloudSQLIPTypes(PSC)).toEqual([PUBLIC, PRIVATE, PSC]);
  });

  test("keeps the specific credential source selected after changing from default", () => {
    function StatefulCredentialSourceForm() {
      const [dataSource, setDataSource] = useState<EditDataSource>(() =>
        makeDataSource(DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM)
      );
      return (
        <CredentialSourceForm
          dataSource={dataSource}
          engine={Engine.POSTGRES}
          allowEdit={true}
          onDataSourceChange={(updates) =>
            setDataSource((prev) => ({ ...prev, ...updates }))
          }
        />
      );
    }

    const { container, unmount } = render(<StatefulCredentialSourceForm />);

    const radios = container.querySelectorAll('[role="radio"]');
    const specificCredential = radios[1] as HTMLElement;
    act(() => {
      specificCredential.click();
    });

    expect(specificCredential.getAttribute("aria-checked")).toBe("true");
    expect(
      container.querySelector("textarea")?.getAttribute("placeholder")
    ).toBe("instance.type-or-paste-credentials-write-only");

    unmount();
  });

  test("keeps the SaaS-forced specific credential source stable", () => {
    serverState.isSaaSMode = true;

    function StatefulCredentialSourceForm() {
      const [dataSource, setDataSource] = useState<EditDataSource>(() =>
        makeDataSource(DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM)
      );
      return (
        <CredentialSourceForm
          dataSource={dataSource}
          engine={Engine.POSTGRES}
          allowEdit={true}
          onDataSourceChange={(updates) =>
            setDataSource((prev) => ({ ...prev, ...updates }))
          }
        />
      );
    }

    const { container, unmount } = render(<StatefulCredentialSourceForm />);

    const radios = container.querySelectorAll('[role="radio"]');
    const defaultCredential = radios[0] as HTMLElement;
    const specificCredential = radios[1] as HTMLElement;

    expect(defaultCredential.getAttribute("aria-checked")).toBe("false");
    expect(specificCredential.getAttribute("aria-checked")).toBe("true");
    expect(
      container.querySelector("textarea")?.getAttribute("placeholder")
    ).toBe("instance.type-or-paste-credentials-write-only");

    unmount();
  });

  test("does not rewrite a credential that is already specific", () => {
    const onDataSourceChange = vi.fn();
    const dataSource = makeDataSource(
      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
    );
    dataSource.iamExtension = {
      case: "gcpCredential",
      value: create(DataSource_GCPCredentialSchema, {
        content: "{}",
      }),
    };

    const { container, unmount } = render(
      <CredentialSourceForm
        dataSource={dataSource}
        engine={Engine.POSTGRES}
        allowEdit={true}
        onDataSourceChange={onDataSourceChange}
      />
    );

    const radios = container.querySelectorAll('[role="radio"]');
    const specificCredential = radios[1] as HTMLElement;
    act(() => {
      specificCredential.click();
    });

    expect(specificCredential.getAttribute("aria-checked")).toBe("true");
    expect(onDataSourceChange).not.toHaveBeenCalled();

    unmount();
  });
});
