import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceExternalSecret_AppRoleAuthOption_SecretType,
  DataSourceExternalSecret_AppRoleAuthOptionSchema,
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceExternalSecretSchema,
  DataSourceType,
  KerberosConfigSchema,
  SASLConfigSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { onlyAllowNumber } from "@/utils";
import { CreateDataSourceExample } from "./CreateDataSourceExample";
import { CredentialSourceForm } from "./CredentialSourceForm";
import type { EditDataSource } from "./common";
import { useInstanceFormContext } from "./InstanceFormContext";
import { hasInfoContent, type InfoSection } from "./info-content";
import { SshConnectionForm } from "./SshConnectionForm";
import { SslCertificateForm } from "./SslCertificateForm";

interface DataSourceFormProps {
  dataSource: EditDataSource;
  hideOptions?: boolean;
  optionsOnly?: boolean;
  onDataSourceChange: (ds: EditDataSource) => void;
  onOpenInfoPanel?: (section: InfoSection) => void;
}

export function DataSourceForm({
  dataSource,
  hideOptions = false,
  optionsOnly = false,
  onDataSourceChange,
  onOpenInfoPanel,
}: DataSourceFormProps) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const {
    instance,
    specs,
    isCreating,
    allowEdit,
    basicInfo,
    adminDataSource,
    hasReadonlyReplicaFeature,
    setMissingFeature,
    hideAdvancedFeatures,
  } = ctx;

  const {
    showDatabase,
    showSSL,
    showSSH,
    allowUsingEmptyPassword,
    showAuthenticationDatabase,
    hasReadonlyReplicaHost,
    hasReadonlyReplicaPort,
    hasExtraParameters,
  } = specs;

  const [passwordType, setPasswordType] = useState(
    dataSource.externalSecret?.secretType ??
      DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED
  );
  const [newParamKey, setNewParamKey] = useState("");
  const [newParamValue, setNewParamValue] = useState("");

  // Sync passwordType when externalSecret changes
  useEffect(() => {
    if (dataSource.externalSecret) {
      setPasswordType(dataSource.externalSecret.secretType);
    } else {
      setPasswordType(
        DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED
      );
    }
  }, [dataSource.externalSecret]);

  const update = useCallback(
    (partial: Partial<EditDataSource>) => {
      onDataSourceChange({ ...dataSource, ...partial });
    },
    [dataSource, onDataSourceChange]
  );

  const hasAuthenticationInfo = hasInfoContent(
    basicInfo.engine,
    "authentication"
  );
  const hasSslInfo = hasInfoContent(basicInfo.engine, "ssl");
  const hasSshInfo = hasInfoContent(basicInfo.engine, "ssh");

  const hiveAuthentication =
    dataSource.saslConfig?.mechanism?.case === "krbConfig"
      ? "KERBEROS"
      : "PASSWORD";

  const onHiveAuthenticationChange = (val: "KERBEROS" | "PASSWORD") => {
    if (val === "KERBEROS") {
      update({
        saslConfig: create(SASLConfigSchema, {
          mechanism: {
            case: "krbConfig",
            value: create(KerberosConfigSchema, {
              kdcTransportProtocol: "tcp",
            }),
          },
        }),
      });
    } else {
      update({ saslConfig: undefined });
    }
  };

  const supportedAuthenticationTypes = useMemo(() => {
    switch (basicInfo.engine) {
      case Engine.COSMOSDB:
        return [
          {
            value: DataSource_AuthenticationType.AZURE_IAM,
            label: t("instance.password-type.azure-iam"),
          },
        ];
      case Engine.MSSQL:
        return [
          {
            value: DataSource_AuthenticationType.PASSWORD,
            label: t("instance.password-type.password"),
          },
          {
            value: DataSource_AuthenticationType.AZURE_IAM,
            label: t("instance.password-type.azure-iam"),
          },
        ];
      case Engine.ELASTICSEARCH:
        return [
          {
            value: DataSource_AuthenticationType.PASSWORD,
            label: t("instance.password-type.password"),
          },
          {
            value: DataSource_AuthenticationType.AWS_RDS_IAM,
            label: t("instance.password-type.aws-iam"),
          },
        ];
      case Engine.SPANNER:
      case Engine.BIGQUERY:
        return [
          {
            value: DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
            label: t("instance.password-type.google-iam"),
          },
        ];
      default:
        return [
          {
            value: DataSource_AuthenticationType.PASSWORD,
            label: t("instance.password-type.password"),
          },
          {
            value: DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
            label: t("instance.password-type.google-iam"),
          },
          {
            value: DataSource_AuthenticationType.AWS_RDS_IAM,
            label: t("instance.password-type.aws-iam"),
          },
        ];
    }
  }, [basicInfo.engine, t]);

  const extraConnectionParamsList = useMemo(() => {
    const params = dataSource.extraConnectionParameters || {};
    return Object.entries(params).map(([key, value]) => ({ key, value }));
  }, [dataSource.extraConnectionParameters]);

  const changeSecretType = (
    secretType: DataSourceExternalSecret_SecretType
  ) => {
    const ds = { ...dataSource };
    switch (secretType) {
      case DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED:
        ds.externalSecret = undefined;
        break;
      case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
        ds.externalSecret = create(DataSourceExternalSecretSchema, {
          authType: DataSourceExternalSecret_AuthType.TOKEN,
          secretType,
          authOption: { case: "token", value: "" },
          secretName: ds.externalSecret?.secretName ?? "",
          passwordKeyName: ds.externalSecret?.passwordKeyName ?? "",
        });
        break;
      case DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER:
        ds.externalSecret = create(DataSourceExternalSecretSchema, {
          authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
          secretType,
          authOption: { case: "token", value: "" },
          secretName: ds.externalSecret?.secretName ?? "",
          passwordKeyName: ds.externalSecret?.passwordKeyName ?? "",
        });
        break;
      case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
        ds.externalSecret = create(DataSourceExternalSecretSchema, {
          authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
          secretType,
          authOption: { case: "token", value: "" },
          secretName: ds.externalSecret?.secretName ?? "",
          passwordKeyName: "",
        });
        break;
      case DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT:
        ds.externalSecret = create(DataSourceExternalSecretSchema, {
          authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
          secretType,
          authOption: { case: "token", value: "" },
          url: ds.externalSecret?.url ?? "",
          secretName: ds.externalSecret?.secretName ?? "",
          passwordKeyName: "",
        });
        break;
    }
    setPasswordType(secretType);
    onDataSourceChange(ds);
  };

  const changeExternalSecretAuthType = (
    authType: DataSourceExternalSecret_AuthType
  ) => {
    if (!dataSource.externalSecret) return;
    const ds = { ...dataSource };
    if (authType === DataSourceExternalSecret_AuthType.VAULT_APP_ROLE) {
      ds.externalSecret = {
        ...ds.externalSecret!,
        authOption: {
          case: "appRole" as const,
          value: create(DataSourceExternalSecret_AppRoleAuthOptionSchema, {
            type: DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN,
          }),
        },
        authType,
      };
    } else {
      ds.externalSecret = {
        ...ds.externalSecret!,
        authOption: { case: "token" as const, value: "" },
        authType,
      };
    }
    onDataSourceChange(ds);
  };

  const toggleUseEmptyPassword = (on: boolean) => {
    update({
      useEmptyPassword: on,
      updatedPassword: on ? "" : dataSource.updatedPassword,
    });
  };

  const handleHostInput = (value: string) => {
    if (dataSource.type === DataSourceType.READ_ONLY) {
      if (!hasReadonlyReplicaFeature) {
        if (dataSource.host || dataSource.port) {
          update({
            host: adminDataSource.host,
            port: adminDataSource.port,
          });
          setMissingFeature(PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION);
          return;
        }
      }
    }
    update({ host: value.trim() });
  };

  const handlePortInput = (value: string) => {
    if (dataSource.type === DataSourceType.READ_ONLY) {
      if (!hasReadonlyReplicaFeature) {
        if (dataSource.host || dataSource.port) {
          update({
            host: adminDataSource.host,
            port: adminDataSource.port,
          });
          setMissingFeature(PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION);
          return;
        }
      }
    }
    update({ port: value.trim() });
  };

  const handleUseSslChanged = (useSSL: boolean) => {
    update({ useSsl: useSSL, updateSsl: true });
  };

  const handleEditSSL = () => {
    update({ sslCa: "", sslCert: "", sslKey: "", updateSsl: true });
  };

  const handleSSHChange = (
    value: Partial<{
      sshHost: string;
      sshPort: string;
      sshUser: string;
      sshPassword: string;
      sshPrivateKey: string;
    }>
  ) => {
    update(value);
  };

  const addNewParameter = () => {
    if (!newParamKey.trim()) return;
    const params = { ...(dataSource.extraConnectionParameters || {}) };
    params[newParamKey.trim()] = newParamValue;
    update({ extraConnectionParameters: params });
    setNewParamKey("");
    setNewParamValue("");
  };

  const updateExtraConnectionParamKey = (index: number, newKey: string) => {
    const params = extraConnectionParamsList;
    if (index >= params.length) return;
    const plain = { ...(dataSource.extraConnectionParameters || {}) };
    const oldKey = params[index].key;
    const value = params[index].value;
    if (oldKey === newKey) return;
    delete plain[oldKey];
    if (newKey.trim()) plain[newKey] = value;
    update({ extraConnectionParameters: plain });
  };

  const updateExtraConnectionParamValue = (index: number, newValue: string) => {
    const params = extraConnectionParamsList;
    if (index >= params.length) return;
    const plain = { ...(dataSource.extraConnectionParameters || {}) };
    plain[params[index].key] = newValue;
    update({ extraConnectionParameters: plain });
  };

  const removeExtraConnectionParam = (index: number) => {
    const params = extraConnectionParamsList;
    if (index >= params.length) return;
    const plain = { ...(dataSource.extraConnectionParameters || {}) };
    delete plain[params[index].key];
    update({ extraConnectionParameters: plain });
  };

  const secretNameLabel = useMemo(() => {
    switch (passwordType) {
      case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
        return t("instance.external-secret-vault.vault-secret-path");
      case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
        return t("instance.external-secret-gcp.secret-name");
      case DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT:
        return t("instance.external-secret-azure.secret-name");
      default:
        return t("instance.external-secret.secret-name");
    }
  }, [passwordType, t]);

  const secretKeyLabel = useMemo(() => {
    if (passwordType === DataSourceExternalSecret_SecretType.VAULT_KV_V2) {
      return t("instance.external-secret-vault.vault-secret-key");
    }
    return t("instance.external-secret.key-name");
  }, [passwordType, t]);

  const showMainFields =
    basicInfo.engine !== Engine.SPANNER &&
    basicInfo.engine !== Engine.BIGQUERY &&
    basicInfo.engine !== Engine.DYNAMODB &&
    basicInfo.engine !== Engine.DATABRICKS;

  const showAuthTypeRadio =
    basicInfo.engine === Engine.MYSQL ||
    basicInfo.engine === Engine.POSTGRES ||
    basicInfo.engine === Engine.COSMOSDB ||
    basicInfo.engine === Engine.MSSQL ||
    basicInfo.engine === Engine.ELASTICSEARCH;

  const isPasswordAuth =
    dataSource.authenticationType === DataSource_AuthenticationType.PASSWORD;
  const isAzureIAM =
    dataSource.authenticationType === DataSource_AuthenticationType.AZURE_IAM;
  const isAwsIAM =
    dataSource.authenticationType === DataSource_AuthenticationType.AWS_RDS_IAM;
  const isGoogleIAM =
    dataSource.authenticationType ===
    DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
  const isIAM = isAzureIAM || isAwsIAM || isGoogleIAM;

  const handleKeytabUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const data = new Uint8Array(reader.result as ArrayBuffer);
      if (dataSource.saslConfig?.mechanism?.case === "krbConfig") {
        const krbValue = dataSource.saslConfig.mechanism.value;
        update({
          saslConfig: create(SASLConfigSchema, {
            ...dataSource.saslConfig,
            mechanism: {
              case: "krbConfig",
              value: create(KerberosConfigSchema, {
                ...krbValue,
                keytab: data,
              }),
            },
          }),
        });
      }
    };
    reader.readAsArrayBuffer(file);
  };

  return (
    <div className="grid grid-cols-1 gap-y-4 gap-x-4 border-none sm:grid-cols-3">
      {!optionsOnly && (
        <>
          {/* Main credential fields */}
          {showMainFields && (
            <>
              {/* Authentication type selector */}
              {showAuthTypeRadio && (
                <div className="sm:col-span-3 sm:col-start-1">
                  <div className="flex items-center gap-x-2">
                    <div className="textlabel flex items-center gap-x-4">
                      {supportedAuthenticationTypes.map((item) => (
                        <label
                          key={item.value}
                          className="flex items-center gap-x-1.5 cursor-pointer"
                        >
                          <input
                            type="radio"
                            name="authenticationType"
                            checked={
                              dataSource.authenticationType === item.value
                            }
                            disabled={!allowEdit}
                            onChange={() =>
                              update({ authenticationType: item.value })
                            }
                          />
                          {item.label}
                        </label>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {/* Create data source example (edit mode only) */}
              {!isCreating && (
                <CreateDataSourceExample
                  className="sm:col-span-3 border-none"
                  createInstanceFlag={false}
                  engine={basicInfo.engine}
                  dataSourceType={dataSource.type}
                  authenticationType={dataSource.authenticationType}
                />
              )}

              {/* Hive authentication */}
              {basicInfo.engine === Engine.HIVE && (
                <div className="sm:col-span-3 sm:col-start-1">
                  <div className="textlabel flex items-center gap-x-4">
                    <label className="flex items-center gap-x-1.5 cursor-pointer">
                      <input
                        type="radio"
                        checked={hiveAuthentication === "PASSWORD"}
                        disabled={!allowEdit}
                        onChange={() => onHiveAuthenticationChange("PASSWORD")}
                      />
                      Plain Password
                    </label>
                    <label className="flex items-center gap-x-1.5 cursor-pointer">
                      <input
                        type="radio"
                        checked={hiveAuthentication === "KERBEROS"}
                        disabled={!allowEdit}
                        onChange={() => onHiveAuthenticationChange("KERBEROS")}
                      />
                      Kerberos
                    </label>
                  </div>
                </div>
              )}

              {/* Kerberos config */}
              {dataSource.saslConfig?.mechanism?.case === "krbConfig" && (
                <>
                  <div className="sm:col-span-3 sm:col-start-1">
                    <label className="textlabel block">
                      Principal <span className="text-red-600">*</span>
                    </label>
                    <div className="mt-2 flex items-center gap-x-2">
                      <Input
                        value={
                          dataSource.saslConfig.mechanism.value.primary ?? ""
                        }
                        disabled={!allowEdit}
                        placeholder="primary"
                        onChange={(e) => {
                          const updated = { ...dataSource };
                          if (
                            updated.saslConfig?.mechanism?.case === "krbConfig"
                          ) {
                            updated.saslConfig.mechanism.value.primary =
                              e.target.value;
                          }
                          onDataSourceChange(updated);
                        }}
                      />
                      <span>/</span>
                      <Input
                        value={
                          dataSource.saslConfig.mechanism.value.instance ?? ""
                        }
                        disabled={!allowEdit}
                        placeholder="instance, optional"
                        onChange={(e) => {
                          const updated = { ...dataSource };
                          if (
                            updated.saslConfig?.mechanism?.case === "krbConfig"
                          ) {
                            updated.saslConfig.mechanism.value.instance =
                              e.target.value;
                          }
                          onDataSourceChange(updated);
                        }}
                      />
                      <span>@</span>
                      <Input
                        value={
                          dataSource.saslConfig.mechanism.value.realm ?? ""
                        }
                        disabled={!allowEdit}
                        placeholder="realm"
                        onChange={(e) => {
                          const updated = { ...dataSource };
                          if (
                            updated.saslConfig?.mechanism?.case === "krbConfig"
                          ) {
                            updated.saslConfig.mechanism.value.realm =
                              e.target.value;
                          }
                          onDataSourceChange(updated);
                        }}
                      />
                    </div>
                  </div>
                  <div className="sm:col-span-3 sm:col-start-1">
                    <label className="textlabel block">
                      KDC <span className="text-red-600">*</span>
                    </label>
                    <div className="flex items-center gap-x-2">
                      <div className="w-fit textlabel flex gap-x-3">
                        {["tcp", "udp"].map((proto) => (
                          <label
                            key={proto}
                            className="flex items-center gap-x-1.5"
                          >
                            <input
                              type="radio"
                              checked={
                                dataSource.saslConfig?.mechanism?.value
                                  ?.kdcTransportProtocol === proto
                              }
                              disabled={!allowEdit}
                              onChange={() => {
                                const updated = { ...dataSource };
                                if (
                                  updated.saslConfig?.mechanism?.case ===
                                  "krbConfig"
                                ) {
                                  updated.saslConfig.mechanism.value.kdcTransportProtocol =
                                    proto;
                                }
                                onDataSourceChange(updated);
                              }}
                            />
                            {proto.toUpperCase()}
                          </label>
                        ))}
                      </div>
                      <Input
                        value={
                          dataSource.saslConfig.mechanism.value.kdcHost ?? ""
                        }
                        disabled={!allowEdit}
                        placeholder="KDC host"
                        onChange={(e) => {
                          const updated = { ...dataSource };
                          if (
                            updated.saslConfig?.mechanism?.case === "krbConfig"
                          ) {
                            updated.saslConfig.mechanism.value.kdcHost =
                              e.target.value;
                          }
                          onDataSourceChange(updated);
                        }}
                      />
                      <span>:</span>
                      <Input
                        value={
                          dataSource.saslConfig.mechanism.value.kdcPort ?? ""
                        }
                        disabled={!allowEdit}
                        placeholder="KDC port, optional"
                        onChange={(e) => {
                          if (
                            e.target.value &&
                            !onlyAllowNumber(e.target.value)
                          )
                            return;
                          const updated = { ...dataSource };
                          if (
                            updated.saslConfig?.mechanism?.case === "krbConfig"
                          ) {
                            updated.saslConfig.mechanism.value.kdcPort =
                              e.target.value;
                          }
                          onDataSourceChange(updated);
                        }}
                      />
                    </div>
                  </div>
                  <div className="sm:col-span-3 sm:col-start-1">
                    <label className="textlabel block">
                      Keytab File <span className="text-red-600">*</span>
                    </label>
                    <div className="mt-3 border-2 border-dashed rounded-lg p-6 text-center">
                      <input
                        type="file"
                        accept=".keytab"
                        className="hidden"
                        id="keytab-upload"
                        onChange={handleKeytabUpload}
                      />
                      <label
                        htmlFor="keytab-upload"
                        className="cursor-pointer textinfolabel"
                      >
                        Click or Drag your .keytab file here
                      </label>
                    </div>
                  </div>
                </>
              )}

              {/* Username field (non-Kerberos, non-Azure IAM) */}
              {dataSource.saslConfig?.mechanism?.case !== "krbConfig" &&
                !isAzureIAM && (
                  <div className="sm:col-span-3 sm:col-start-1">
                    <label className="textlabel flex items-center gap-x-1">
                      {t("common.username")}
                      {isCreating &&
                        onOpenInfoPanel &&
                        hasAuthenticationInfo && (
                          <button
                            type="button"
                            className="text-accent text-xs hover:underline"
                            onClick={() => onOpenInfoPanel("authentication")}
                          >
                            ⓘ
                          </button>
                        )}
                    </label>
                    <Input
                      value={dataSource.username}
                      className="mt-2 w-full max-w-[48rem]"
                      disabled={!allowEdit}
                      placeholder={
                        basicInfo.engine === Engine.CLICKHOUSE
                          ? t("common.default")
                          : ""
                      }
                      onChange={(e) => update({ username: e.target.value })}
                    />
                  </div>
                )}

              {/* IAM Credential Source Form */}
              {isIAM && (
                <CredentialSourceForm
                  dataSource={dataSource}
                  allowEdit={allowEdit}
                  onDataSourceChange={update}
                />
              )}

              {/* AWS Region */}
              {isAwsIAM && (
                <div className="sm:col-span-3 sm:col-start-1">
                  <label className="textlabel block">
                    {t("instance.database-region")}{" "}
                    <span className="text-red-600">*</span>
                  </label>
                  <Input
                    value={dataSource.region ?? ""}
                    className="mt-2 w-full"
                    disabled={!allowEdit}
                    placeholder="database region, for example, us-east-1"
                    onChange={(e) => update({ region: e.target.value })}
                  />
                </div>
              )}

              {/* Password / External Secret */}
              {isPasswordAuth &&
                dataSource.saslConfig?.mechanism?.case !== "krbConfig" && (
                  <div className="sm:col-span-3 sm:col-start-1">
                    {!hideAdvancedFeatures && (
                      <div className="mb-4">
                        <div className="textlabel flex flex-wrap gap-x-4 gap-y-2">
                          {[
                            {
                              value:
                                DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED,
                              label: t("instance.password-type.password"),
                            },
                            {
                              value:
                                DataSourceExternalSecret_SecretType.VAULT_KV_V2,
                              label: t(
                                "instance.password-type.external-secret-vault"
                              ),
                            },
                            {
                              value:
                                DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER,
                              label: t(
                                "instance.password-type.external-secret-aws"
                              ),
                            },
                            {
                              value:
                                DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER,
                              label: t(
                                "instance.password-type.external-secret-gcp"
                              ),
                            },
                            {
                              value:
                                DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT,
                              label: t(
                                "instance.password-type.external-secret-azure"
                              ),
                            },
                          ].map((item) => (
                            <label
                              key={item.value}
                              className="flex items-center gap-x-1.5 cursor-pointer"
                            >
                              <input
                                type="radio"
                                checked={passwordType === item.value}
                                disabled={!allowEdit}
                                onChange={() => changeSecretType(item.value)}
                              />
                              {item.label}
                              {item.value !==
                                DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED && (
                                <span className="text-xs bg-indigo-100 text-indigo-800 px-1.5 py-0.5 rounded-full">
                                  Pro
                                </span>
                              )}
                            </label>
                          ))}
                        </div>
                        <a
                          href="https://docs.bytebase.com/get-started/connect/overview#secret-manager-integration"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-sm text-accent hover:underline"
                        >
                          {t("common.learn-more")}
                        </a>
                      </div>
                    )}

                    {/* Plain password */}
                    {passwordType ===
                      DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED && (
                      <div>
                        <label className="textlabel block">
                          {t("common.password")}
                        </label>
                        <div className="mt-2">
                          {!isCreating && allowUsingEmptyPassword && (
                            <label className="flex items-center gap-x-1.5 mb-2 text-sm cursor-pointer">
                              <input
                                type="checkbox"
                                checked={dataSource.useEmptyPassword ?? false}
                                disabled={!allowEdit}
                                onChange={(e) =>
                                  toggleUseEmptyPassword(e.target.checked)
                                }
                              />
                              {t("instance.no-password")}
                            </label>
                          )}
                          <Input
                            type="password"
                            className="w-full max-w-[48rem]"
                            autoComplete="off"
                            placeholder={
                              dataSource.useEmptyPassword
                                ? t("instance.no-password")
                                : t("instance.password-write-only")
                            }
                            disabled={
                              !allowEdit || !!dataSource.useEmptyPassword
                            }
                            value={
                              dataSource.useEmptyPassword
                                ? ""
                                : dataSource.updatedPassword
                            }
                            onChange={(e) =>
                              update({
                                updatedPassword: e.target.value.trim(),
                              })
                            }
                          />
                        </div>
                      </div>
                    )}

                    {/* External secret fields */}
                    {passwordType !==
                      DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED &&
                      dataSource.externalSecret && (
                        <div className="flex flex-col gap-y-4">
                          {/* Vault KV V2 */}
                          {passwordType ===
                            DataSourceExternalSecret_SecretType.VAULT_KV_V2 && (
                            <div className="flex flex-col gap-y-4">
                              <div>
                                <label className="textlabel block">
                                  {t(
                                    "instance.external-secret-vault.vault-url"
                                  )}{" "}
                                  <span className="text-red-600">*</span>
                                </label>
                                <Input
                                  value={dataSource.externalSecret.url ?? ""}
                                  required
                                  className="mt-2 w-full"
                                  disabled={!allowEdit}
                                  placeholder={t(
                                    "instance.external-secret-vault.vault-url"
                                  )}
                                  onChange={(e) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      url: e.target.value,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                />
                              </div>
                              <div className="flex flex-col gap-y-2">
                                <label className="textlabel block">
                                  {t(
                                    "instance.external-secret-vault.vault-auth-type.self"
                                  )}
                                </label>
                                <div className="textlabel flex gap-x-4">
                                  <label className="flex items-center gap-x-1.5 cursor-pointer">
                                    <input
                                      type="radio"
                                      checked={
                                        dataSource.externalSecret.authType ===
                                        DataSourceExternalSecret_AuthType.TOKEN
                                      }
                                      onChange={() =>
                                        changeExternalSecretAuthType(
                                          DataSourceExternalSecret_AuthType.TOKEN
                                        )
                                      }
                                    />
                                    {t(
                                      "instance.external-secret-vault.vault-auth-type.token.self"
                                    )}
                                  </label>
                                  <label className="flex items-center gap-x-1.5 cursor-pointer">
                                    <input
                                      type="radio"
                                      checked={
                                        dataSource.externalSecret.authType ===
                                        DataSourceExternalSecret_AuthType.VAULT_APP_ROLE
                                      }
                                      onChange={() =>
                                        changeExternalSecretAuthType(
                                          DataSourceExternalSecret_AuthType.VAULT_APP_ROLE
                                        )
                                      }
                                    />
                                    {t(
                                      "instance.external-secret-vault.vault-auth-type.approle.self"
                                    )}
                                  </label>
                                </div>
                              </div>
                              {/* Token input */}
                              {dataSource.externalSecret.authType ===
                                DataSourceExternalSecret_AuthType.TOKEN && (
                                <div>
                                  <label className="textlabel block">
                                    {t(
                                      "instance.external-secret-vault.vault-auth-type.token.self"
                                    )}{" "}
                                    <span className="text-red-600">*</span>
                                  </label>
                                  <Input
                                    value={
                                      dataSource.externalSecret.authOption
                                        ?.case === "token"
                                        ? (dataSource.externalSecret.authOption
                                            .value as string)
                                        : ""
                                    }
                                    className="mt-2 w-full"
                                    disabled={!allowEdit}
                                    placeholder={`${t("instance.external-secret-vault.vault-auth-type.token.self")} - ${t("common.write-only")}`}
                                    onChange={(e) => {
                                      const ds = { ...dataSource };
                                      ds.externalSecret = {
                                        ...ds.externalSecret!,
                                        authOption: {
                                          case: "token" as const,
                                          value: e.target.value,
                                        },
                                      };
                                      onDataSourceChange(ds);
                                    }}
                                  />
                                </div>
                              )}
                              {/* AppRole fields */}
                              {dataSource.externalSecret.authOption?.case ===
                                "appRole" && (
                                <div className="flex flex-col gap-y-4">
                                  <div>
                                    <label className="textlabel block">
                                      {t(
                                        "instance.external-secret-vault.vault-auth-type.approle.role-id"
                                      )}{" "}
                                      <span className="text-red-600">*</span>
                                    </label>
                                    <Input
                                      value={
                                        dataSource.externalSecret.authOption
                                          .value.roleId ?? ""
                                      }
                                      className="mt-2 w-full"
                                      disabled={!allowEdit}
                                      placeholder={`${t("instance.external-secret-vault.vault-auth-type.approle.role-id")} - ${t("common.write-only")}`}
                                      onChange={(e) => {
                                        const ds = { ...dataSource };
                                        if (
                                          ds.externalSecret?.authOption
                                            ?.case === "appRole"
                                        ) {
                                          ds.externalSecret = {
                                            ...ds.externalSecret,
                                            authOption: {
                                              ...ds.externalSecret.authOption,
                                              value: {
                                                ...ds.externalSecret.authOption
                                                  .value,
                                                roleId: e.target.value,
                                              },
                                            },
                                          };
                                        }
                                        onDataSourceChange(ds);
                                      }}
                                    />
                                  </div>
                                  <div>
                                    <label className="textlabel block">
                                      {t(
                                        "instance.external-secret-vault.vault-auth-type.approle.secret-id"
                                      )}{" "}
                                      <span className="text-red-600">*</span>
                                    </label>
                                    <div className="textlabel my-1 flex gap-x-4">
                                      <label className="flex items-center gap-x-1.5 cursor-pointer">
                                        <input
                                          type="radio"
                                          checked={
                                            dataSource.externalSecret.authOption
                                              .value.type === 0
                                          }
                                          disabled={!allowEdit}
                                          onChange={() => {
                                            const ds = { ...dataSource };
                                            if (
                                              ds.externalSecret?.authOption
                                                ?.case === "appRole"
                                            ) {
                                              ds.externalSecret = {
                                                ...ds.externalSecret,
                                                authOption: {
                                                  ...ds.externalSecret
                                                    .authOption,
                                                  value: {
                                                    ...ds.externalSecret
                                                      .authOption.value,
                                                    type: 0,
                                                  },
                                                },
                                              };
                                            }
                                            onDataSourceChange(ds);
                                          }}
                                        />
                                        {t(
                                          "instance.external-secret-vault.vault-auth-type.approle.secret-plain-text"
                                        )}
                                      </label>
                                      <label className="flex items-center gap-x-1.5 cursor-pointer">
                                        <input
                                          type="radio"
                                          checked={
                                            dataSource.externalSecret.authOption
                                              .value.type === 1
                                          }
                                          disabled={!allowEdit}
                                          onChange={() => {
                                            const ds = { ...dataSource };
                                            if (
                                              ds.externalSecret?.authOption
                                                ?.case === "appRole"
                                            ) {
                                              ds.externalSecret = {
                                                ...ds.externalSecret,
                                                authOption: {
                                                  ...ds.externalSecret
                                                    .authOption,
                                                  value: {
                                                    ...ds.externalSecret
                                                      .authOption.value,
                                                    type: 1,
                                                  },
                                                },
                                              };
                                            }
                                            onDataSourceChange(ds);
                                          }}
                                        />
                                        {t(
                                          "instance.external-secret-vault.vault-auth-type.approle.secret-env-name"
                                        )}
                                      </label>
                                    </div>
                                    <Input
                                      value={
                                        dataSource.externalSecret.authOption
                                          .value.secretId ?? ""
                                      }
                                      className="mt-2 w-full"
                                      disabled={!allowEdit}
                                      onChange={(e) => {
                                        const ds = { ...dataSource };
                                        if (
                                          ds.externalSecret?.authOption
                                            ?.case === "appRole"
                                        ) {
                                          ds.externalSecret = {
                                            ...ds.externalSecret,
                                            authOption: {
                                              ...ds.externalSecret.authOption,
                                              value: {
                                                ...ds.externalSecret.authOption
                                                  .value,
                                                secretId: e.target.value,
                                              },
                                            },
                                          };
                                        }
                                        onDataSourceChange(ds);
                                      }}
                                    />
                                  </div>
                                </div>
                              )}
                              {/* Vault TLS config */}
                              <div>
                                <label className="textlabel block">
                                  {t(
                                    "instance.external-secret-vault.vault-tls-config"
                                  )}
                                </label>
                                <SslCertificateForm
                                  verify={
                                    !dataSource.externalSecret
                                      .skipVaultTlsVerification
                                  }
                                  onVerifyChange={(val) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      skipVaultTlsVerification: !val,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                  ca={dataSource.externalSecret.vaultSslCa}
                                  onCaChange={(val) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      vaultSslCa: val,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                  cert={dataSource.externalSecret.vaultSslCert}
                                  onCertChange={(val) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      vaultSslCert: val,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                  sslKey={dataSource.externalSecret.vaultSslKey}
                                  onKeyChange={(val) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      vaultSslKey: val,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                  disabled={!allowEdit}
                                  showKeyAndCert
                                />
                              </div>
                              {/* Engine name */}
                              <div>
                                <label className="textlabel block">
                                  {t(
                                    "instance.external-secret-vault.vault-secret-engine-name"
                                  )}{" "}
                                  <span className="text-red-600">*</span>
                                </label>
                                <div className="flex gap-x-2 text-sm textinfolabel">
                                  {t(
                                    "instance.external-secret-vault.vault-secret-engine-tips"
                                  )}
                                </div>
                                <Input
                                  value={
                                    dataSource.externalSecret.engineName ?? ""
                                  }
                                  required
                                  className="mt-2 w-full"
                                  disabled={!allowEdit}
                                  placeholder={t(
                                    "instance.external-secret-vault.vault-secret-engine-name"
                                  )}
                                  onChange={(e) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      engineName: e.target.value,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                />
                              </div>
                            </div>
                          )}

                          {/* Azure Key Vault URL */}
                          {passwordType ===
                            DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT && (
                            <div>
                              <label className="textlabel block">
                                {t("instance.external-secret-azure.vault-url")}{" "}
                                <span className="text-red-600">*</span>
                              </label>
                              <div className="flex gap-x-2 text-sm textinfolabel">
                                {t(
                                  "instance.external-secret-azure.vault-url-tips"
                                )}
                              </div>
                              <Input
                                value={dataSource.externalSecret.url ?? ""}
                                required
                                className="mt-2 w-full"
                                disabled={!allowEdit}
                                placeholder={t(
                                  "instance.external-secret-azure.vault-url"
                                )}
                                onChange={(e) => {
                                  const ds = { ...dataSource };
                                  ds.externalSecret = {
                                    ...ds.externalSecret!,
                                    url: e.target.value,
                                  };
                                  onDataSourceChange(ds);
                                }}
                              />
                            </div>
                          )}

                          {/* Secret name (common) */}
                          <div>
                            <label className="textlabel block">
                              {secretNameLabel}{" "}
                              <span className="text-red-600">*</span>
                            </label>
                            {passwordType ===
                              DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER && (
                              <div className="flex gap-x-2 text-sm textinfolabel">
                                {t(
                                  "instance.external-secret-gcp.secret-name-tips"
                                )}
                              </div>
                            )}
                            {passwordType ===
                              DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT && (
                              <div className="flex gap-x-2 text-sm textinfolabel">
                                {t(
                                  "instance.external-secret-azure.secret-name-tips"
                                )}
                              </div>
                            )}
                            <Input
                              value={dataSource.externalSecret.secretName ?? ""}
                              required
                              className="mt-2 w-full"
                              disabled={!allowEdit}
                              placeholder={secretNameLabel}
                              onChange={(e) => {
                                const ds = { ...dataSource };
                                ds.externalSecret = {
                                  ...ds.externalSecret!,
                                  secretName: e.target.value,
                                };
                                onDataSourceChange(ds);
                              }}
                            />
                          </div>

                          {/* Secret key (not for GCP/Azure) */}
                          {passwordType !==
                            DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER &&
                            passwordType !==
                              DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT && (
                              <div>
                                <label className="textlabel block">
                                  {secretKeyLabel}{" "}
                                  <span className="text-red-600">*</span>
                                </label>
                                <Input
                                  value={
                                    dataSource.externalSecret.passwordKeyName ??
                                    ""
                                  }
                                  required
                                  className="mt-2 w-full"
                                  disabled={!allowEdit}
                                  placeholder={secretKeyLabel}
                                  onChange={(e) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      passwordKeyName: e.target.value,
                                    };
                                    onDataSourceChange(ds);
                                  }}
                                />
                              </div>
                            )}
                        </div>
                      )}

                    {/* Redis Sentinel fields */}
                    {basicInfo.engine === Engine.REDIS &&
                      dataSource.redisType ===
                        DataSource_RedisType.SENTINEL && (
                        <>
                          <div className="mt-2">
                            <label className="textlabel">
                              Master Name{" "}
                              <span className="text-red-600">*</span>
                            </label>
                            <Input
                              value={dataSource.masterName ?? ""}
                              className="mt-1 w-full"
                              disabled={!allowEdit}
                              onChange={(e) =>
                                update({ masterName: e.target.value })
                              }
                            />
                          </div>
                          <div className="mt-2">
                            <label className="textlabel">Master Username</label>
                            <Input
                              value={dataSource.masterUsername ?? ""}
                              className="mt-1 w-full"
                              disabled={!allowEdit}
                              onChange={(e) =>
                                update({ masterUsername: e.target.value })
                              }
                            />
                          </div>
                          <div className="mt-2">
                            <label className="textlabel block">
                              Master Password
                            </label>
                            <div className="mt-2">
                              {!isCreating && allowUsingEmptyPassword && (
                                <label className="flex items-center gap-x-1.5 mb-2 text-sm cursor-pointer">
                                  <input
                                    type="checkbox"
                                    checked={
                                      dataSource.useEmptyMasterPassword ?? false
                                    }
                                    disabled={!allowEdit}
                                    onChange={(e) => {
                                      update({
                                        useEmptyMasterPassword:
                                          e.target.checked,
                                        updatedMasterPassword: e.target.checked
                                          ? ""
                                          : dataSource.updatedMasterPassword,
                                      });
                                    }}
                                  />
                                  {t("instance.no-password")}
                                </label>
                              )}
                              <Input
                                type="password"
                                className="w-full"
                                autoComplete="off"
                                placeholder={
                                  dataSource.useEmptyMasterPassword
                                    ? t("instance.no-password")
                                    : t("instance.password-write-only")
                                }
                                disabled={
                                  !allowEdit ||
                                  !!dataSource.useEmptyMasterPassword
                                }
                                value={
                                  dataSource.useEmptyMasterPassword
                                    ? ""
                                    : dataSource.updatedMasterPassword
                                }
                                onChange={(e) =>
                                  update({
                                    updatedMasterPassword:
                                      e.target.value.trim(),
                                  })
                                }
                              />
                            </div>
                          </div>
                        </>
                      )}
                  </div>
                )}
            </>
          )}

          {/* Spanner/BigQuery auth */}
          {(basicInfo.engine === Engine.SPANNER ||
            basicInfo.engine === Engine.BIGQUERY) && (
            <>
              <div className="sm:col-span-3 sm:col-start-1 textlabel flex gap-x-4">
                {supportedAuthenticationTypes.map((item) => (
                  <label
                    key={item.value}
                    className="flex items-center gap-x-1.5 cursor-pointer"
                  >
                    <input
                      type="radio"
                      checked={dataSource.authenticationType === item.value}
                      disabled={!allowEdit}
                      onChange={() =>
                        update({ authenticationType: item.value })
                      }
                    />
                    {item.label}
                  </label>
                ))}
              </div>
              <CredentialSourceForm
                dataSource={dataSource}
                allowEdit={allowEdit}
                onDataSourceChange={update}
              />
            </>
          )}

          {/* Oracle SID/Service Name */}
          {basicInfo.engine === Engine.ORACLE && (
            <OracleSIDServiceNameInput
              sid={dataSource.sid ?? ""}
              serviceName={dataSource.serviceName ?? ""}
              allowEdit={allowEdit}
              onSidChange={(val) => update({ sid: val })}
              onServiceNameChange={(val) => update({ serviceName: val })}
            />
          )}

          {/* Snowflake keypair */}
          {basicInfo.engine === Engine.SNOWFLAKE && (
            <>
              <div className="sm:col-span-3 sm:col-start-1">
                <div className="textlabel block">
                  {t("data-source.ssh.private-key")}
                </div>
                <div className="flex gap-x-2 text-sm">
                  <span className="textinfolabel">
                    {t("data-source.snowflake-keypair-tip")}
                  </span>
                  <a
                    href="https://docs.snowflake.com/en/user-guide/key-pair-auth"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-accent hover:underline"
                  >
                    {t("common.learn-more")}
                  </a>
                </div>
                <textarea
                  value={dataSource.authenticationPrivateKey ?? ""}
                  disabled={!allowEdit}
                  className="w-full h-32 mt-2 whitespace-pre-wrap rounded-sm border border-control-border p-2 text-sm font-mono"
                  placeholder={`-----BEGIN PRIVATE KEY-----\nMIIEvQ...\n-----END PRIVATE KEY-----`}
                  onChange={(e) =>
                    update({ authenticationPrivateKey: e.target.value })
                  }
                />
              </div>
              <div className="sm:col-span-3 sm:col-start-1">
                <div className="textlabel block">
                  {t("data-source.private-key-passphrase")}
                </div>
                <div className="textinfolabel text-sm">
                  {t("data-source.private-key-passphrase-tip")}
                </div>
                <Input
                  value={dataSource.authenticationPrivateKeyPassphrase ?? ""}
                  type="password"
                  className="mt-2 w-full"
                  disabled={!allowEdit}
                  placeholder={t(
                    "data-source.private-key-passphrase-placeholder"
                  )}
                  onChange={(e) =>
                    update({
                      authenticationPrivateKeyPassphrase: e.target.value,
                    })
                  }
                />
              </div>
            </>
          )}

          {/* Databricks */}
          {basicInfo.engine === Engine.DATABRICKS && (
            <>
              <div>
                <div className="textlabel mt-2">
                  Warehouse ID <span className="text-red-600">*</span>
                </div>
                <Input
                  value={dataSource.warehouseId ?? ""}
                  className="mt-2"
                  disabled={!allowEdit}
                  onChange={(e) => update({ warehouseId: e.target.value })}
                />
              </div>
              <div>
                <div className="textlabel mt-2">
                  Token <span className="text-red-600">*</span>
                </div>
                <Input
                  value={dataSource.authenticationPrivateKey ?? ""}
                  className="mt-2 w-full"
                  disabled={!allowEdit}
                  placeholder="personal access token"
                  onChange={(e) =>
                    update({ authenticationPrivateKey: e.target.value })
                  }
                />
              </div>
            </>
          )}

          {/* MongoDB authentication database */}
          {showAuthenticationDatabase && (
            <div className="sm:col-span-3 sm:col-start-1">
              <label className="textlabel block">
                {t("instance.authentication-database")}
              </label>
              <Input
                className="mt-2 w-full"
                autoComplete="off"
                placeholder="admin"
                disabled={!allowEdit}
                value={dataSource.authenticationDatabase ?? ""}
                onChange={(e) =>
                  update({ authenticationDatabase: e.target.value.trim() })
                }
              />
            </div>
          )}

          {/* Read-only replica host/port */}
          {dataSource.type === DataSourceType.READ_ONLY &&
            (hasReadonlyReplicaHost || hasReadonlyReplicaPort) && (
              <>
                {hasReadonlyReplicaHost && (
                  <div className="sm:col-span-3 sm:col-start-1">
                    <label className="textlabel block">
                      {t("data-source.read-replica-host")}
                    </label>
                    <Input
                      className="mt-2 w-full"
                      autoComplete="off"
                      value={dataSource.host}
                      disabled={!allowEdit}
                      onChange={(e) => handleHostInput(e.target.value)}
                    />
                  </div>
                )}
                {hasReadonlyReplicaPort && (
                  <div className="sm:col-span-3 sm:col-start-1">
                    <label className="textlabel block">
                      {t("data-source.read-replica-port")}
                    </label>
                    <Input
                      className="mt-2 w-full"
                      autoComplete="off"
                      value={dataSource.port}
                      disabled={!allowEdit}
                      onChange={(e) => {
                        if (e.target.value && !onlyAllowNumber(e.target.value))
                          return;
                        handlePortInput(e.target.value);
                      }}
                    />
                  </div>
                )}
              </>
            )}

          {/* Database field */}
          {showDatabase && (
            <div className="sm:col-span-3 sm:col-start-1">
              <label className="textlabel block">{t("common.database")}</label>
              <Input
                value={dataSource.database ?? ""}
                className="mt-2 w-full"
                disabled={!allowEdit}
                placeholder={t("common.database")}
                onChange={(e) => update({ database: e.target.value })}
              />
            </div>
          )}
        </>
      )}

      {/* Connection options (SSL, SSH, Extra params) */}
      {!hideOptions && (
        <>
          {/* SSL */}
          {showSSL && isPasswordAuth && (
            <div className="sm:col-span-3 sm:col-start-1">
              <div className="flex items-center justify-start gap-x-2 textlabel">
                {t("data-source.ssl-connection")}
                <button
                  type="button"
                  className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                    dataSource.useSsl ? "bg-accent" : "bg-gray-200"
                  }`}
                  onClick={() => handleUseSslChanged(!dataSource.useSsl)}
                >
                  <span
                    className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                      dataSource.useSsl ? "translate-x-4.5" : "translate-x-0.5"
                    }`}
                  />
                </button>
                {isCreating && onOpenInfoPanel && hasSslInfo && (
                  <button
                    type="button"
                    className="text-accent text-xs hover:underline"
                    onClick={() => onOpenInfoPanel("ssl")}
                  >
                    ⓘ
                  </button>
                )}
              </div>
              {dataSource.useSsl && (
                <>
                  {dataSource.pendingCreate || dataSource.updateSsl ? (
                    <SslCertificateForm
                      verify={dataSource.verifyTlsCertificate}
                      onVerifyChange={(val) =>
                        update({ verifyTlsCertificate: val })
                      }
                      ca={dataSource.sslCa}
                      onCaChange={(val) =>
                        update({ sslCa: val, updateSsl: true })
                      }
                      cert={dataSource.sslCert}
                      onCertChange={(val) =>
                        update({ sslCert: val, updateSsl: true })
                      }
                      sslKey={dataSource.sslKey}
                      onKeyChange={(val) =>
                        update({ sslKey: val, updateSsl: true })
                      }
                      engineType={basicInfo.engine}
                      disabled={!allowEdit}
                    />
                  ) : (
                    <Button
                      variant="outline"
                      className="mt-2"
                      disabled={!allowEdit}
                      onClick={handleEditSSL}
                    >
                      {t("common.edit")} - {t("common.write-only")}
                    </Button>
                  )}
                </>
              )}
            </div>
          )}

          {/* SSH */}
          {!hideAdvancedFeatures && showSSH && isPasswordAuth && (
            <div className="sm:col-span-3 sm:col-start-1">
              <div className="flex flex-row items-center gap-x-1">
                <label className="textlabel block">
                  {t("data-source.ssh-connection")}
                </label>
                {isCreating && onOpenInfoPanel && hasSshInfo && (
                  <button
                    type="button"
                    className="text-accent text-xs hover:underline"
                    onClick={() => onOpenInfoPanel("ssh")}
                  >
                    ⓘ
                  </button>
                )}
              </div>
              <SshConnectionForm
                value={dataSource}
                instance={instance}
                disabled={!allowEdit}
                onChange={handleSSHChange}
              />
            </div>
          )}

          {/* Extra connection parameters */}
          {hasExtraParameters && (
            <div className="sm:col-span-3 sm:col-start-1">
              <div className="flex flex-row items-center justify-between">
                <label className="textlabel block">
                  {t("data-source.extra-params.self")}
                </label>
              </div>
              <div className="textinfolabel text-sm mt-1 mb-2">
                {t("data-source.extra-params.description")}
              </div>

              {allowEdit && (
                <div className="flex mt-2 mb-2 gap-x-2 bg-gray-50 p-3 rounded-sm">
                  <Input
                    value={newParamKey}
                    className="w-full"
                    placeholder={t("instance.parameter-name-placeholder")}
                    onChange={(e) => setNewParamKey(e.target.value)}
                  />
                  <Input
                    value={newParamValue}
                    className="w-full"
                    placeholder={t("instance.parameter-value-placeholder")}
                    onChange={(e) => setNewParamValue(e.target.value)}
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={!newParamKey.trim()}
                    onClick={addNewParameter}
                  >
                    Add
                  </Button>
                </div>
              )}

              {extraConnectionParamsList.map((param, index) => (
                <div key={param.key} className="flex mt-2 gap-x-2">
                  <Input
                    className="w-full"
                    value={param.key}
                    disabled={!allowEdit}
                    placeholder="Parameter name"
                    onChange={(e) =>
                      updateExtraConnectionParamKey(index, e.target.value)
                    }
                  />
                  <Input
                    className="w-full"
                    value={param.value}
                    disabled={!allowEdit}
                    placeholder="Parameter value"
                    onChange={(e) =>
                      updateExtraConnectionParamValue(index, e.target.value)
                    }
                  />
                  {allowEdit && (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => removeExtraConnectionParam(index)}
                    >
                      Remove
                    </Button>
                  )}
                </div>
              ))}

              {extraConnectionParamsList.length === 0 && (
                <div className="textinfolabel text-sm italic">
                  {allowEdit
                    ? t("instance.no-params-yet-add-above")
                    : t("instance.no-extra-params-configured")}
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}

// Matches Vue OracleSIDAndServiceNameInput.vue exactly
function OracleSIDServiceNameInput({
  sid,
  serviceName,
  allowEdit,
  onSidChange,
  onServiceNameChange,
}: {
  sid: string;
  serviceName: string;
  allowEdit: boolean;
  onSidChange: (val: string) => void;
  onServiceNameChange: (val: string) => void;
}) {
  // Track which mode is selected — default to "serviceName" if both empty
  const mode = sid ? "sid" : "serviceName";

  const handleModeChange = (newMode: "sid" | "serviceName") => {
    if (newMode === "sid") {
      onServiceNameChange("");
      if (!sid) onSidChange("XE");
    } else {
      onSidChange("");
    }
  };

  return (
    <div className="sm:col-span-3 sm:col-start-1">
      <div className="textlabel flex gap-x-4 mb-2">
        <label className="flex items-center gap-x-1.5 cursor-pointer">
          <input
            type="radio"
            checked={mode === "sid"}
            onChange={() => handleModeChange("sid")}
            disabled={!allowEdit}
          />
          SID
        </label>
        <label className="flex items-center gap-x-1.5 cursor-pointer">
          <input
            type="radio"
            checked={mode === "serviceName"}
            onChange={() => handleModeChange("serviceName")}
            disabled={!allowEdit}
          />
          Service Name
        </label>
      </div>
      <Input
        value={mode === "sid" ? sid : serviceName}
        className="w-full"
        disabled={!allowEdit}
        onChange={(e) => {
          if (mode === "sid") {
            onSidChange(e.target.value);
          } else {
            onServiceNameChange(e.target.value);
          }
        }}
      />
    </div>
  );
}
