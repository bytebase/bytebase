import { create } from "@bufbuild/protobuf";
import {
  type DragEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { Textarea } from "@/react/components/ui/textarea";
import { useVueState } from "@/react/hooks/useVueState";
import { useActuatorV1Store } from "@/store";
import {
  DataSource_AuthenticationType,
  DataSource_AWSCredentialSchema,
  DataSource_AzureCredentialSchema,
  DataSource_GCPCredentialSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";

type CredentialSource = "default" | "specific-credential";

interface CredentialSourceFormProps {
  dataSource: EditDataSource;
  allowEdit: boolean;
  onDataSourceChange: (updates: Partial<EditDataSource>) => void;
}

function CredentialSourceForm({
  dataSource,
  allowEdit,
  onDataSourceChange,
}: CredentialSourceFormProps) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);

  const isIAMAuthentication =
    dataSource.authenticationType ===
      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM ||
    dataSource.authenticationType ===
      DataSource_AuthenticationType.AWS_RDS_IAM ||
    dataSource.authenticationType === DataSource_AuthenticationType.AZURE_IAM;

  const isDefaultCredentialDisabled = isSaaSMode && isIAMAuthentication;

  const deriveCredentialSource = useCallback((): CredentialSource => {
    const ext = dataSource.iamExtension;
    if (
      ext?.case === "azureCredential" ||
      ext?.case === "awsCredential" ||
      ext?.case === "gcpCredential"
    ) {
      return "specific-credential";
    }
    return "default";
  }, [dataSource.iamExtension]);

  const [credentialSource, setCredentialSource] = useState<CredentialSource>(
    () => {
      const derived = deriveCredentialSource();
      if (isDefaultCredentialDisabled && derived === "default") {
        return "specific-credential";
      }
      return derived;
    }
  );

  // Sync credentialSource when iamExtension changes externally
  useEffect(() => {
    setCredentialSource(deriveCredentialSource());
  }, [deriveCredentialSource]);

  // Force specific credential in SaaS mode for IAM authentication
  useEffect(() => {
    if (isDefaultCredentialDisabled && credentialSource === "default") {
      setCredentialSource("specific-credential");
    }
  }, [isDefaultCredentialDisabled, credentialSource]);

  // When credential source changes, update the iamExtension on the dataSource
  useEffect(() => {
    const authType = dataSource.authenticationType;
    if (credentialSource === "default") {
      onDataSourceChange({ iamExtension: { case: undefined } });
      return;
    }
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
  }, [credentialSource]);

  // Reset to default when authentication type changes
  const [prevAuthType, setPrevAuthType] = useState(
    dataSource.authenticationType
  );
  if (dataSource.authenticationType !== prevAuthType) {
    setPrevAuthType(dataSource.authenticationType);
    setCredentialSource("default");
  }

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
    setCredentialSource(value);
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
      <div className="textlabel mt-2 flex gap-x-4">
        {options.map((option) => (
          <label
            key={option.value}
            className="inline-flex items-center gap-x-1.5"
            title={
              option.disabled
                ? t(
                    "instance.iam-extension.saas-default-credential-restriction"
                  )
                : undefined
            }
          >
            <input
              type="radio"
              name="credential-source"
              value={option.value}
              checked={credentialSource === option.value}
              disabled={!allowEdit || option.disabled}
              onChange={() => handleCredentialSourceChange(option.value)}
            />
            <span>{option.label}</span>
          </label>
        ))}
      </div>

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
  return (
    <div className="mt-1 sm:col-span-3 sm:col-start-1 textinfolabel !leading-6">
      {authenticationType === DataSource_AuthenticationType.AZURE_IAM && (
        <span>
          Bytebase will read the credential from environment variables{" "}
          <Code>AZURE_CLIENT_ID</Code>/<Code>AZURE_TENANT_ID</Code>/
          <Code>AZURE_CLIENT_SECRET</Code> or{" "}
          <Code>AZURE_CLIENT_CERTIFICATE_PATH</Code>, and fallback to attached
          users in Azure VM
        </span>
      )}
      {authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM && (
        <span>
          Bytebase will read the credential from environment variables{" "}
          <Code>AWS_ACCESS_KEY_ID</Code>/<Code>AWS_SECRET_ACCESS_KEY</Code>/
          <Code>AWS_SESSION_TOKEN</Code>, fallback to shared credentials file{" "}
          <Code>~/.aws/credentials</Code> or IAM role in AWS ECS
        </span>
      )}
      {authenticationType ===
        DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
        <span>
          Bytebase will read the credential from environment variable{" "}
          <Code>GOOGLE_APPLICATION_CREDENTIALS</Code>, fallback to the attached
          service account in GCP GCE
        </span>
      )}
    </div>
  );
}

function Code({ children }: { children: React.ReactNode }) {
  return <code className="bg-gray-100 p-1 rounded-sm mr-1">{children}</code>;
}

export { CredentialSourceForm };
