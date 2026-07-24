import { create } from "@bufbuild/protobuf";
import {
  type DragEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Trans, useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";
import { useServerState } from "@/hooks/useAppState";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_AWSCredentialSchema,
  DataSource_AzureCredentialSchema,
  DataSource_CloudSQLIPType,
  DataSource_GCPCredentialSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";

type CredentialSource = "default" | "specific-credential";

interface CredentialSourceFormProps {
  dataSource: EditDataSource;
  engine: Engine;
  allowEdit: boolean;
  onDataSourceChange: (updates: Partial<EditDataSource>) => void;
}

function CredentialSourceForm({
  dataSource,
  engine,
  allowEdit,
  onDataSourceChange,
}: Readonly<CredentialSourceFormProps>) {
  const { t } = useTranslation();
  const { isSaaSMode } = useServerState();

  const isIAMAuthentication =
    dataSource.authenticationType ===
      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM ||
    dataSource.authenticationType ===
      DataSource_AuthenticationType.AWS_RDS_IAM ||
    dataSource.authenticationType === DataSource_AuthenticationType.AZURE_IAM;

  const isDefaultCredentialDisabled = isSaaSMode && isIAMAuthentication;

  const expectedCredentialCase = useMemo(() => {
    switch (dataSource.authenticationType) {
      case DataSource_AuthenticationType.AWS_RDS_IAM:
        return "awsCredential";
      case DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
        return "gcpCredential";
      case DataSource_AuthenticationType.AZURE_IAM:
        return "azureCredential";
      default:
        return undefined;
    }
  }, [dataSource.authenticationType]);
  const hasCredentialForAuthenticationType =
    dataSource.iamExtension?.case === expectedCredentialCase;
  const credentialSource: CredentialSource =
    hasCredentialForAuthenticationType || isDefaultCredentialDisabled
      ? "specific-credential"
      : "default";

  const clearCredential = useCallback(() => {
    onDataSourceChange({ iamExtension: { case: undefined } });
  }, [onDataSourceChange]);

  const applySpecificCredential = useCallback(() => {
    if (dataSource.iamExtension?.case === expectedCredentialCase) {
      return;
    }
    const authType = dataSource.authenticationType;
    switch (authType) {
      case DataSource_AuthenticationType.AWS_RDS_IAM:
        onDataSourceChange({
          iamExtension: {
            case: "awsCredential",
            value: create(
              DataSource_AWSCredentialSchema,
              dataSource.iamExtension?.case === "awsCredential"
                ? dataSource.iamExtension.value
                : {}
            ),
          },
        });
        break;
      case DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
        onDataSourceChange({
          iamExtension: {
            case: "gcpCredential",
            value: create(
              DataSource_GCPCredentialSchema,
              dataSource.iamExtension?.case === "gcpCredential"
                ? dataSource.iamExtension.value
                : {}
            ),
          },
        });
        break;
      case DataSource_AuthenticationType.AZURE_IAM:
        onDataSourceChange({
          iamExtension: {
            case: "azureCredential",
            value: create(
              DataSource_AzureCredentialSchema,
              dataSource.iamExtension?.case === "azureCredential"
                ? dataSource.iamExtension.value
                : {}
            ),
          },
        });
        break;
    }
  }, [
    dataSource.authenticationType,
    dataSource.iamExtension,
    expectedCredentialCase,
    onDataSourceChange,
  ]);

  useEffect(() => {
    if (
      dataSource.iamExtension?.case &&
      dataSource.iamExtension.case !== expectedCredentialCase
    ) {
      clearCredential();
      return;
    }
    if (isDefaultCredentialDisabled && !hasCredentialForAuthenticationType) {
      applySpecificCredential();
    }
  }, [
    applySpecificCredential,
    clearCredential,
    dataSource.iamExtension?.case,
    expectedCredentialCase,
    hasCredentialForAuthenticationType,
    isDefaultCredentialDisabled,
  ]);

  const options = useMemo(
    () => [
      {
        label: t("common.default"),
        value: "default" as CredentialSource,
        disabled: isDefaultCredentialDisabled,
      },
      {
        label: t("instance.iam-extension.specific-credential"),
        value: "specific-credential" as CredentialSource,
      },
    ],
    [t, isDefaultCredentialDisabled]
  );

  const handleCredentialSourceChange = (value: CredentialSource) => {
    if (!allowEdit) return;
    if (value === "default") {
      clearCredential();
      return;
    }
    applySpecificCredential();
  };

  const updateAzureField = (
    field: "tenantId" | "clientId" | "clientSecret",
    val: string
  ) => {
    if (dataSource.iamExtension?.case === "azureCredential") {
      onDataSourceChange({
        iamExtension: {
          case: "azureCredential",
          value: create(DataSource_AzureCredentialSchema, {
            ...dataSource.iamExtension.value,
            [field]: val,
          }),
        },
      });
    }
  };

  const updateAwsField = (
    field:
      | "accessKeyId"
      | "secretAccessKey"
      | "sessionToken"
      | "roleArn"
      | "externalId",
    val: string
  ) => {
    if (dataSource.iamExtension?.case === "awsCredential") {
      onDataSourceChange({
        iamExtension: {
          case: "awsCredential",
          value: create(DataSource_AWSCredentialSchema, {
            ...dataSource.iamExtension.value,
            [field]: val,
          }),
        },
      });
    }
  };

  const updateGcpContent = (val: string) => {
    onDataSourceChange({
      iamExtension: {
        case: "gcpCredential",
        value: create(DataSource_GCPCredentialSchema, { content: val }),
      },
    });
  };

  return (
    <div className="sm:col-span-3 sm:col-start-1">
      <label htmlFor="credential-source" className="textlabel block">
        {t("instance.iam-extension.credential-source")}
      </label>
      <RadioGroup
        className="textlabel mt-2 gap-x-4"
        value={credentialSource}
        onValueChange={(value) =>
          handleCredentialSourceChange(value as CredentialSource)
        }
      >
        {options.map((option) => (
          <RadioGroupItem
            key={option.value}
            value={option.value}
            disabled={!allowEdit || option.disabled}
            title={
              option.disabled
                ? t(
                    "instance.iam-extension.saas-default-credential-restriction"
                  )
                : undefined
            }
          >
            {option.label}
          </RadioGroupItem>
        ))}
      </RadioGroup>

      {credentialSource === "specific-credential" && (
        <>
          {dataSource.authenticationType ===
            DataSource_AuthenticationType.AZURE_IAM && (
            <AzureCredentialFields
              dataSource={dataSource}
              allowEdit={allowEdit}
              onFieldChange={updateAzureField}
            />
          )}
          {dataSource.authenticationType ===
            DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
            <GcpCredentialField
              value={
                dataSource.iamExtension?.case === "gcpCredential"
                  ? dataSource.iamExtension.value.content
                  : ""
              }
              onChange={updateGcpContent}
            />
          )}
          {dataSource.authenticationType ===
            DataSource_AuthenticationType.AWS_RDS_IAM && (
            <AwsCredentialFields
              dataSource={dataSource}
              allowEdit={allowEdit}
              onFieldChange={updateAwsField}
            />
          )}
        </>
      )}

      {credentialSource === "default" && (
        <DefaultCredentialInfo
          authenticationType={dataSource.authenticationType}
        />
      )}

      {showsCloudSQLIPType(engine, dataSource.authenticationType) && (
        <CloudSQLIPTypeField
          value={dataSource.cloudSqlIpType}
          allowEdit={allowEdit}
          onChange={(cloudSqlIpType) => onDataSourceChange({ cloudSqlIpType })}
        />
      )}
    </div>
  );
}

// The Cloud SQL IP type only applies to Cloud SQL connections (which use the
// cloudsqlconn dialer). Google Cloud SQL IAM auth is also offered for Spanner and
// BigQuery, whose drivers do not use cloudsqlconn, so restrict the selector to the
// Cloud SQL MySQL/Postgres engines.
export function showsCloudSQLIPType(
  engine: Engine,
  authType: DataSource_AuthenticationType
): boolean {
  return (
    authType === DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM &&
    (engine === Engine.MYSQL || engine === Engine.POSTGRES)
  );
}

// PSC is accepted via the API and Terraform provider but not offered in the UI
// until it is verified end-to-end. An instance that already has PSC set (via the
// API) still shows it so the stored value renders correctly.
export function offeredCloudSQLIPTypes(
  current: DataSource_CloudSQLIPType
): DataSource_CloudSQLIPType[] {
  const offered = [
    DataSource_CloudSQLIPType.PUBLIC,
    DataSource_CloudSQLIPType.PRIVATE,
  ];
  if (current === DataSource_CloudSQLIPType.PSC) {
    offered.push(DataSource_CloudSQLIPType.PSC);
  }
  return offered;
}

function CloudSQLIPTypeField({
  value,
  allowEdit,
  onChange,
}: Readonly<{
  value: DataSource_CloudSQLIPType;
  allowEdit: boolean;
  onChange: (value: DataSource_CloudSQLIPType) => void;
}>) {
  const { t } = useTranslation();
  // Treat unspecified as public for display, matching the backend default.
  const current =
    value === DataSource_CloudSQLIPType.CLOUD_SQL_IP_TYPE_UNSPECIFIED
      ? DataSource_CloudSQLIPType.PUBLIC
      : value;
  const label = (ipType: DataSource_CloudSQLIPType) => {
    switch (ipType) {
      case DataSource_CloudSQLIPType.PRIVATE:
        return t("instance.cloud-sql-ip-type.private");
      case DataSource_CloudSQLIPType.PSC:
        return t("instance.cloud-sql-ip-type.psc");
      default:
        return t("instance.cloud-sql-ip-type.public");
    }
  };
  const options = offeredCloudSQLIPTypes(current).map((ipType) => ({
    value: ipType,
    label: label(ipType),
  }));

  return (
    <div className="mt-4 sm:col-span-3 sm:col-start-1">
      <label className="textlabel block">
        {t("instance.cloud-sql-ip-type.label")}
      </label>
      <RadioGroup
        className="textlabel mt-2 gap-x-4"
        value={String(current)}
        onValueChange={(next) => onChange(Number(next))}
      >
        {options.map((option) => (
          <RadioGroupItem
            key={option.value}
            value={String(option.value)}
            disabled={!allowEdit}
          >
            {option.label}
          </RadioGroupItem>
        ))}
      </RadioGroup>
      <p className="textinfolabel mt-1">
        {t("instance.cloud-sql-ip-type.description")}
      </p>
    </div>
  );
}

function AzureCredentialFields({
  dataSource,
  allowEdit,
  onFieldChange,
}: {
  dataSource: EditDataSource;
  allowEdit: boolean;
  onFieldChange: (
    field: "tenantId" | "clientId" | "clientSecret",
    val: string
  ) => void;
}) {
  const { t } = useTranslation();
  const azure =
    dataSource.iamExtension?.case === "azureCredential"
      ? dataSource.iamExtension.value
      : undefined;

  return (
    <div className="mt-4 sm:col-span-3 sm:col-start-1">
      <label className="textlabel block mt-2">
        {t("instance.iam-extension.tenant-id")}
      </label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        value={azure?.tenantId ?? ""}
        onChange={(e) => onFieldChange("tenantId", e.target.value)}
      />
      <label className="textlabel block mt-2">
        {t("instance.iam-extension.client-id")}
      </label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        value={azure?.clientId ?? ""}
        onChange={(e) => onFieldChange("clientId", e.target.value)}
      />
      <label className="textlabel block mt-2">
        {t("instance.iam-extension.client-secret")}
      </label>
      <Input
        type="password"
        className="mt-2 w-full"
        disabled={!allowEdit}
        placeholder={t("instance.type-or-paste-credentials-write-only")}
        value={azure?.clientSecret ?? ""}
        onChange={(e) => onFieldChange("clientSecret", e.target.value)}
      />
    </div>
  );
}

function AwsCredentialFields({
  dataSource,
  allowEdit,
  onFieldChange,
}: {
  dataSource: EditDataSource;
  allowEdit: boolean;
  onFieldChange: (
    field:
      | "accessKeyId"
      | "secretAccessKey"
      | "sessionToken"
      | "roleArn"
      | "externalId",
    val: string
  ) => void;
}) {
  const { t } = useTranslation();
  const aws =
    dataSource.iamExtension?.case === "awsCredential"
      ? dataSource.iamExtension.value
      : undefined;

  return (
    <div className="mt-4 sm:col-span-3 sm:col-start-1">
      <label className="textlabel block mt-2">Access Key ID</label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        placeholder={t("common.sensitive-placeholder")}
        value={aws?.accessKeyId ?? ""}
        onChange={(e) => onFieldChange("accessKeyId", e.target.value)}
      />
      <label className="textlabel block mt-2">Secret Access Key</label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        placeholder={t("common.sensitive-placeholder")}
        value={aws?.secretAccessKey ?? ""}
        onChange={(e) => onFieldChange("secretAccessKey", e.target.value)}
      />
      <label className="textlabel block mt-2">Session Token</label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        placeholder={t("common.sensitive-placeholder")}
        value={aws?.sessionToken ?? ""}
        onChange={(e) => onFieldChange("sessionToken", e.target.value)}
      />
      <label className="textlabel block mt-2">{t("instance.role-arn")}</label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        placeholder={t("instance.role-arn-placeholder")}
        value={aws?.roleArn ?? ""}
        onChange={(e) => onFieldChange("roleArn", e.target.value)}
      />
      <div className="text-sm text-gray-500 mt-1">
        {t("instance.role-arn-description")}
      </div>
      <label className="textlabel block mt-2">
        {t("instance.external-id")}
      </label>
      <Input
        className="mt-2 w-full"
        disabled={!allowEdit}
        placeholder={t("instance.external-id-placeholder")}
        value={aws?.externalId ?? ""}
        onChange={(e) => onFieldChange("externalId", e.target.value)}
      />
      <div className="text-sm text-gray-500 mt-1">
        {t("instance.external-id-description")}
      </div>
    </div>
  );
}

function GcpCredentialField({
  value,
  onChange,
}: {
  value: string;
  onChange: (val: string) => void;
}) {
  const { t } = useTranslation();
  const [isDragOver, setIsDragOver] = useState(false);

  const handleDragOver = (e: DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = () => {
    setIsDragOver(false);
  };

  const handleDrop = (e: DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
    setIsDragOver(false);
    const files = e.dataTransfer.files;
    if (files.length > 0) {
      const file = files[0];
      const reader = new FileReader();
      reader.onload = (event) => {
        const content = event.target?.result;
        if (typeof content === "string") {
          onChange(content);
        }
      };
      reader.readAsText(file);
    }
  };

  return (
    <div className="mt-2 sm:col-span-3 sm:col-start-1">
      <div className="flex flex-col gap-y-1 w-full">
        <p className="textinfolabel">
          <span>{t("instance.create-gcp-credentials")}</span>
          <a
            href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
            target="_blank"
            rel="noreferrer"
            className="normal-link inline-flex items-center ml-1"
          >
            <span>{t("common.detailed-guide")}</span>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-4 h-4 ml-1"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
          </a>
        </p>
        <Textarea
          value={value}
          placeholder={t("instance.type-or-paste-credentials-write-only")}
          className={`w-full h-24 whitespace-pre-wrap resize-none ${isDragOver ? "border-accent" : ""}`}
          onChange={(e) => onChange(e.target.value)}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        />
      </div>
    </div>
  );
}

function DefaultCredentialInfo({
  authenticationType,
}: {
  authenticationType: DataSource_AuthenticationType;
}) {
  const { t } = useTranslation();

  return (
    <div className="mt-1 sm:col-span-3 sm:col-start-1 textinfolabel !leading-6">
      {authenticationType === DataSource_AuthenticationType.AZURE_IAM && (
        <Trans
          t={t}
          i18nKey="instance.iam-extension.default-credential.azure"
          components={{
            azureClientId: <Code />,
            azureTenantId: <Code />,
            azureClientSecret: <Code />,
            azureClientCertificatePath: <Code />,
          }}
        />
      )}
      {authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM && (
        <Trans
          t={t}
          i18nKey="instance.iam-extension.default-credential.aws"
          components={{
            awsAccessKeyId: <Code />,
            awsSecretAccessKey: <Code />,
            awsSessionToken: <Code />,
            awsCredentialsPath: <Code />,
          }}
        />
      )}
      {authenticationType ===
        DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
        <Trans
          t={t}
          i18nKey="instance.iam-extension.default-credential.gcp"
          components={{
            googleApplicationCredentials: <Code />,
          }}
        />
      )}
    </div>
  );
}

function Code({ children }: { children?: React.ReactNode }) {
  return <code className="bg-gray-100 p-1 rounded-sm mr-1">{children}</code>;
}

export { CredentialSourceForm };
