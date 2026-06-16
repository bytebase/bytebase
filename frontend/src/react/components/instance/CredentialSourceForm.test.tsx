import { create } from "@bufbuild/protobuf";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  DataSource_AuthenticationType,
  DataSourceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { CredentialSourceForm } from "./CredentialSourceForm";
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

vi.mock("@/react/hooks/useAppState", () => ({
  useServerState: () => ({
    isSaaSMode: false,
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
  ])("renders default credential help through i18n for authentication type %s", (authenticationType, expectedKey) => {
    const { container, unmount } = render(
      <CredentialSourceForm
        dataSource={makeDataSource(authenticationType)}
        allowEdit={true}
        onDataSourceChange={vi.fn()}
      />
    );

    expect(container.textContent).toContain(expectedKey);

    unmount();
  });
});
