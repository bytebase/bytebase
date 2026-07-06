import { create } from "@bufbuild/protobuf";
import { Info } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  FormControlGroup,
  FormControlRow,
  FormField,
} from "@/react/components/ui/form";
import { Input } from "@/react/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { useAppStore } from "@/react/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceExternalSecret_AppRoleAuthOption_SecretType,
  DataSourceExternalSecret_AppRoleAuthOptionSchema,
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceExternalSecret_TokenType,
  DataSourceExternalSecretSchema,
  DataSourceType,
  KerberosConfigSchema,
  SASLConfigSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { onlyAllowNumber } from "@/utils";
import { CreateDataSourceExample } from "./CreateDataSourceExample";
import { CredentialSourceForm } from "./CredentialSourceForm";
import type { EditDataSource, TlsUpdateState } from "./common";
import { useInstanceFormContext } from "./InstanceFormContext";
import { hasInfoContent, type InfoSection } from "./info-content";
import { SshConnectionForm } from "./SshConnectionForm";
import { SslCertificateForm } from "./SslCertificateForm";
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  applyLocalTlsPosture,
  disableLocalTls,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  getLocalTlsPosture,
  LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
  LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
  LOCAL_TLS_POSTURE_DISABLED,
  LOCAL_TLS_POSTURE_MUTUAL_TLS,
  LOCAL_TLS_POSTURE_TLS,
  type LocalTlsPosture,
} from "./tls";

interface DataSourceFormProps {
  dataSource: EditDataSource;
  hideOptions?: boolean;
  optionsOnly?: boolean;
  onDataSourceChange: (ds: EditDataSource) => void;
  onOpenInfoPanel?: (section: InfoSection) => void;
}

const mergeTlsUpdateState = (
  current: TlsUpdateState | undefined,
  next: TlsUpdateState
): TlsUpdateState => {
  if (current === true || next === true) {
    return true;
  }
  if (!current) {
    return next;
  }
  if (typeof current === "boolean") {
    return next;
  }
  if (typeof next === "boolean") {
    return current;
  }
  return {
    useSsl: current.useSsl || next.useSsl,
    ca: current.ca || next.ca,
    clientCert: current.clientCert || next.clientCert,
  };
};

export function DataSourceForm({
  dataSource,
  hideOptions = false,
  optionsOnly = false,
  onDataSourceChange,
  onOpenInfoPanel,
}: DataSourceFormProps) {
  const { t } = useTranslation();
  const currentPlan = useAppStore((s) => s.currentPlan());
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());
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
  const [localTlsCaSource, setLocalTlsCaSource] = useState(
    getLocalTlsCaSource(dataSource)
  );
  const [localTlsClientCertSource, setLocalTlsClientCertSource] = useState(
    getLocalTlsClientCertSource(dataSource)
  );
  const [localTlsPosture, setLocalTlsPosture] = useState(
    getLocalTlsPosture(dataSource)
  );
  const previousDataSourceIdRef = useRef(dataSource.id);

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

  useEffect(() => {
    if (previousDataSourceIdRef.current !== dataSource.id) {
      previousDataSourceIdRef.current = dataSource.id;
      setLocalTlsCaSource(getLocalTlsCaSource(dataSource));
      setLocalTlsClientCertSource(getLocalTlsClientCertSource(dataSource));
      setLocalTlsPosture(getLocalTlsPosture(dataSource));
      return;
    }
    if (!dataSource.updateSsl) {
      setLocalTlsCaSource(getLocalTlsCaSource(dataSource));
      setLocalTlsClientCertSource(getLocalTlsClientCertSource(dataSource));
      setLocalTlsPosture(getLocalTlsPosture(dataSource));
    }
  }, [
    dataSource.id,
    dataSource.useSsl,
    dataSource.sslCa,
    dataSource.sslCert,
    dataSource.sslKey,
    dataSource.sslCaPath,
    dataSource.sslCertPath,
    dataSource.sslKeyPath,
    dataSource.sslCaSet,
    dataSource.sslCertSet,
    dataSource.sslKeySet,
    dataSource.sslCaPathSet,
    dataSource.sslCertPathSet,
    dataSource.sslKeyPathSet,
    dataSource.updateSsl,
  ]);

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

  const onLocalTlsPostureChange = useCallback(
    (posture: LocalTlsPosture) => {
      setLocalTlsPosture(posture);
      if (posture === LOCAL_TLS_POSTURE_DISABLED) {
        setLocalTlsCaSource("SYSTEM_TRUST");
        setLocalTlsClientCertSource(LOCAL_TLS_CLIENT_CERT_SOURCE_NONE);
        update({ ...disableLocalTls(dataSource), updateSsl: true });
        return;
      }

      const next = applyLocalTlsPosture(dataSource, posture);
      const enablingTls = !dataSource.useSsl;
      if (posture === LOCAL_TLS_POSTURE_TLS) {
        setLocalTlsClientCertSource(LOCAL_TLS_CLIENT_CERT_SOURCE_NONE);
      }
      if (
        posture === LOCAL_TLS_POSTURE_MUTUAL_TLS &&
        localTlsClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
      ) {
        setLocalTlsClientCertSource(LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM);
      }

      update({
        ...next,
        verifyTlsCertificate: enablingTls
          ? true
          : dataSource.verifyTlsCertificate,
        updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
          useSsl: true,
          clientCert: posture === LOCAL_TLS_POSTURE_TLS,
        }),
      });
    },
    [dataSource, localTlsClientCertSource, update]
  );

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
                    <RadioGroup
                      className="textlabel gap-x-4"
                      value={String(dataSource.authenticationType)}
                      onValueChange={(value) =>
                        update({
                          authenticationType: Number(
                            value
                          ) as DataSource_AuthenticationType,
                        })
                      }
                    >
                      {supportedAuthenticationTypes.map((item) => (
                        <RadioGroupItem
                          key={item.value}
                          value={String(item.value)}
                          disabled={!allowEdit}
                        >
                          {item.label}
                        </RadioGroupItem>
                      ))}
                    </RadioGroup>
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
                  <RadioGroup
                    className="textlabel gap-x-4"
                    value={hiveAuthentication}
                    onValueChange={(value) =>
                      onHiveAuthenticationChange(
                        value as typeof hiveAuthentication
                      )
                    }
                  >
                    <RadioGroupItem value="PASSWORD" disabled={!allowEdit}>
                      Plain Password
                    </RadioGroupItem>
                    <RadioGroupItem value="KERBEROS" disabled={!allowEdit}>
                      Kerberos
                    </RadioGroupItem>
                  </RadioGroup>
                </div>
              )}

              {/* Kerberos config */}
              {dataSource.saslConfig?.mechanism?.case === "krbConfig" && (
                <>
                  <FormField
                    className="sm:col-span-3 sm:col-start-1"
                    title={
                      <>
                        Principal <span className="text-error">*</span>
                      </>
                    }
                  >
                    <FormControlRow>
                      <Input
                        className="min-w-0 flex-1"
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
                      <span className="shrink-0 whitespace-nowrap text-sm leading-5 text-control-light">
                        /
                      </span>
                      <Input
                        className="min-w-0 flex-1"
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
                      <span className="shrink-0 whitespace-nowrap text-sm leading-5 text-control-light">
                        @
                      </span>
                      <Input
                        className="min-w-0 flex-1"
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
                    </FormControlRow>
                  </FormField>
                  <FormField
                    className="sm:col-span-3 sm:col-start-1"
                    title={
                      <>
                        KDC <span className="text-error">*</span>
                      </>
                    }
                  >
                    <FormControlRow>
                      <RadioGroup
                        className="w-fit textlabel gap-x-3"
                        value={
                          dataSource.saslConfig?.mechanism?.value
                            ?.kdcTransportProtocol ?? ""
                        }
                        onValueChange={(proto) => {
                          const updated = { ...dataSource };
                          if (
                            updated.saslConfig?.mechanism?.case === "krbConfig"
                          ) {
                            updated.saslConfig.mechanism.value.kdcTransportProtocol =
                              String(proto);
                          }
                          onDataSourceChange(updated);
                        }}
                      >
                        {["tcp", "udp"].map((proto) => (
                          <RadioGroupItem
                            key={proto}
                            value={proto}
                            disabled={!allowEdit}
                          >
                            {proto.toUpperCase()}
                          </RadioGroupItem>
                        ))}
                      </RadioGroup>
                      <Input
                        className="min-w-0 flex-1"
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
                      <span className="shrink-0 whitespace-nowrap text-sm leading-5 text-control-light">
                        :
                      </span>
                      <Input
                        className="min-w-0 flex-1"
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
                    </FormControlRow>
                  </FormField>
                  <FormField
                    className="sm:col-span-3 sm:col-start-1"
                    title={
                      <>
                        Keytab File <span className="text-error">*</span>
                      </>
                    }
                  >
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
                  </FormField>
                </>
              )}

              {/* Username field (non-Kerberos, non-Azure IAM) */}
              {dataSource.saslConfig?.mechanism?.case !== "krbConfig" &&
                !isAzureIAM && (
                  <FormField
                    className="sm:col-span-3 sm:col-start-1"
                    title={
                      <span className="flex items-center gap-x-1">
                        {t("common.username")}
                        {onOpenInfoPanel && hasAuthenticationInfo && (
                          <button
                            type="button"
                            className="inline-flex items-center gap-x-0.5 text-accent text-xs"
                            onClick={() => onOpenInfoPanel("authentication")}
                          >
                            <Info className="size-3.5" />
                          </button>
                        )}
                      </span>
                    }
                  >
                    <Input
                      value={dataSource.username}
                      className="w-full max-w-[48rem]"
                      disabled={!allowEdit}
                      placeholder={
                        basicInfo.engine === Engine.CLICKHOUSE
                          ? t("common.default")
                          : ""
                      }
                      onChange={(e) => update({ username: e.target.value })}
                    />
                  </FormField>
                )}

              {/* IAM Credential Source Form */}
              {isIAM && (
                <CredentialSourceForm
                  dataSource={dataSource}
                  engine={basicInfo.engine}
                  allowEdit={allowEdit}
                  onDataSourceChange={update}
                />
              )}

              {/* AWS Region */}
              {isAwsIAM && (
                <FormField
                  className="sm:col-span-3 sm:col-start-1"
                  title={
                    <>
                      {t("instance.database-region")}{" "}
                      <span className="text-error">*</span>
                    </>
                  }
                >
                  <Input
                    value={dataSource.region ?? ""}
                    className="w-full"
                    disabled={!allowEdit}
                    placeholder="database region, for example, us-east-1"
                    onChange={(e) => update({ region: e.target.value })}
                  />
                </FormField>
              )}

              {/* Password / External Secret */}
              {isPasswordAuth &&
                dataSource.saslConfig?.mechanism?.case !== "krbConfig" && (
                  <div className="sm:col-span-3 sm:col-start-1">
                    {!hideAdvancedFeatures && (
                      <div className="mb-4">
                        <RadioGroup
                          className="textlabel flex-wrap gap-x-4 gap-y-2"
                          value={String(passwordType)}
                          onValueChange={(value) =>
                            changeSecretType(
                              Number(
                                value
                              ) as DataSourceExternalSecret_SecretType
                            )
                          }
                        >
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
                            <RadioGroupItem
                              key={item.value}
                              value={String(item.value)}
                              disabled={!allowEdit}
                              contentClassName="flex items-center gap-x-1.5"
                            >
                              {item.label}
                              {item.value !==
                                DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED &&
                                currentPlan === PlanType.FREE && (
                                  <span className="text-xs bg-indigo-100 text-indigo-800 px-1.5 py-0.5 rounded-full">
                                    Pro
                                  </span>
                                )}
                            </RadioGroupItem>
                          ))}
                          <LearnMoreLink
                            href="https://docs.bytebase.com/get-started/connect/overview#secret-manager-integration"
                            className="text-sm text-accent"
                          />
                        </RadioGroup>
                      </div>
                    )}

                    {/* Plain password */}
                    {passwordType ===
                      DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED && (
                      <FormField title={<>{t("common.password")}</>}>
                        <div>
                          {!isCreating && allowUsingEmptyPassword && (
                            <label className="flex items-center gap-x-1.5 mb-2 text-sm cursor-pointer">
                              <Checkbox
                                checked={dataSource.useEmptyPassword ?? false}
                                disabled={!allowEdit}
                                onCheckedChange={(checked) =>
                                  toggleUseEmptyPassword(checked)
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
                      </FormField>
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
                              <FormField
                                title={
                                  <>
                                    {t(
                                      "instance.external-secret-vault.vault-url"
                                    )}{" "}
                                    <span className="text-error">*</span>
                                  </>
                                }
                              >
                                <Input
                                  value={dataSource.externalSecret.url ?? ""}
                                  required
                                  className="w-full"
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
                              </FormField>
                              <FormField
                                title={
                                  <>
                                    {t(
                                      "instance.external-secret-vault.vault-auth-type.self"
                                    )}
                                  </>
                                }
                              >
                                <RadioGroup
                                  className="textlabel gap-x-4"
                                  value={String(
                                    dataSource.externalSecret.authType
                                  )}
                                  onValueChange={(value) =>
                                    changeExternalSecretAuthType(
                                      Number(
                                        value
                                      ) as DataSourceExternalSecret_AuthType
                                    )
                                  }
                                >
                                  <RadioGroupItem
                                    value={String(
                                      DataSourceExternalSecret_AuthType.TOKEN
                                    )}
                                  >
                                    {t(
                                      "instance.external-secret-vault.vault-auth-type.token.self"
                                    )}
                                  </RadioGroupItem>
                                  <RadioGroupItem
                                    value={String(
                                      DataSourceExternalSecret_AuthType.VAULT_APP_ROLE
                                    )}
                                  >
                                    {t(
                                      "instance.external-secret-vault.vault-auth-type.approle.self"
                                    )}
                                  </RadioGroupItem>
                                </RadioGroup>
                              </FormField>
                              {/* Token input */}
                              {dataSource.externalSecret.authType ===
                                DataSourceExternalSecret_AuthType.TOKEN &&
                                (() => {
                                  const tokenType =
                                    dataSource.externalSecret.tokenType ===
                                      DataSourceExternalSecret_TokenType.ENVIRONMENT ||
                                    dataSource.externalSecret.tokenType ===
                                      DataSourceExternalSecret_TokenType.FILE
                                      ? dataSource.externalSecret.tokenType
                                      : DataSourceExternalSecret_TokenType.PLAIN;
                                  const changeTokenType = (
                                    type: DataSourceExternalSecret_TokenType
                                  ) => {
                                    const ds = { ...dataSource };
                                    ds.externalSecret = {
                                      ...ds.externalSecret!,
                                      tokenType: type,
                                    };
                                    onDataSourceChange(ds);
                                  };
                                  const tokenLabel = t(
                                    "instance.external-secret-vault.vault-auth-type.token.self"
                                  );
                                  let tokenPlaceholder = `${tokenLabel} - ${t("common.write-only")}`;
                                  if (
                                    tokenType ===
                                    DataSourceExternalSecret_TokenType.ENVIRONMENT
                                  ) {
                                    tokenPlaceholder = t(
                                      "instance.external-secret-vault.vault-auth-type.token.env-name"
                                    );
                                  } else if (
                                    tokenType ===
                                    DataSourceExternalSecret_TokenType.FILE
                                  ) {
                                    tokenPlaceholder = t(
                                      "instance.external-secret-vault.vault-auth-type.token.file-path"
                                    );
                                  }
                                  return (
                                    <FormField
                                      title={
                                        <>
                                          {tokenLabel}{" "}
                                          <span className="text-error">*</span>
                                        </>
                                      }
                                    >
                                      {/* Token source is host-backed for env/file,
                                          which is disallowed in SaaS mode; only
                                          plain is offered there. */}
                                      {!isSaaSMode && (
                                        <RadioGroup
                                          className="textlabel my-1 gap-x-4"
                                          value={String(tokenType)}
                                          onValueChange={(value) =>
                                            changeTokenType(
                                              Number(
                                                value
                                              ) as DataSourceExternalSecret_TokenType
                                            )
                                          }
                                        >
                                          <RadioGroupItem
                                            value={String(
                                              DataSourceExternalSecret_TokenType.PLAIN
                                            )}
                                            disabled={!allowEdit}
                                          >
                                            {t(
                                              "instance.external-secret-vault.vault-auth-type.token.type-plain"
                                            )}
                                          </RadioGroupItem>
                                          <RadioGroupItem
                                            value={String(
                                              DataSourceExternalSecret_TokenType.ENVIRONMENT
                                            )}
                                            disabled={!allowEdit}
                                          >
                                            {t(
                                              "instance.external-secret-vault.vault-auth-type.token.type-environment"
                                            )}
                                          </RadioGroupItem>
                                          <RadioGroupItem
                                            value={String(
                                              DataSourceExternalSecret_TokenType.FILE
                                            )}
                                            disabled={!allowEdit}
                                          >
                                            {t(
                                              "instance.external-secret-vault.vault-auth-type.token.type-file"
                                            )}
                                          </RadioGroupItem>
                                        </RadioGroup>
                                      )}
                                      <Input
                                        value={
                                          dataSource.externalSecret.authOption
                                            ?.case === "token"
                                            ? (dataSource.externalSecret
                                                .authOption.value as string)
                                            : ""
                                        }
                                        className="w-full"
                                        disabled={!allowEdit}
                                        placeholder={tokenPlaceholder}
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
                                    </FormField>
                                  );
                                })()}
                              {/* AppRole fields */}
                              {dataSource.externalSecret.authOption?.case ===
                                "appRole" && (
                                <div className="flex flex-col gap-y-4">
                                  <FormField
                                    title={
                                      <>
                                        {t(
                                          "instance.external-secret-vault.vault-auth-type.approle.role-id"
                                        )}{" "}
                                        <span className="text-error">*</span>
                                      </>
                                    }
                                  >
                                    <Input
                                      value={
                                        dataSource.externalSecret.authOption
                                          .value.roleId ?? ""
                                      }
                                      className="w-full"
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
                                  </FormField>
                                  <FormField
                                    title={
                                      <>
                                        {t(
                                          "instance.external-secret-vault.vault-auth-type.approle.secret-id"
                                        )}{" "}
                                        <span className="text-error">*</span>
                                      </>
                                    }
                                  >
                                    {/* Environment secret id reads from the host,
                                        which is disallowed in SaaS mode; only plain
                                        is offered there. */}
                                    {!isSaaSMode && (
                                      <RadioGroup
                                        className="textlabel my-1 gap-x-4"
                                        value={String(
                                          dataSource.externalSecret.authOption
                                            .value.type
                                        )}
                                        onValueChange={(value) => {
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
                                                  ...ds.externalSecret
                                                    .authOption.value,
                                                  type: Number(
                                                    value
                                                  ) as DataSourceExternalSecret_AppRoleAuthOption_SecretType,
                                                },
                                              },
                                            };
                                          }
                                          onDataSourceChange(ds);
                                        }}
                                      >
                                        <RadioGroupItem
                                          value={String(
                                            DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN
                                          )}
                                          disabled={!allowEdit}
                                        >
                                          {t(
                                            "instance.external-secret-vault.vault-auth-type.approle.secret-plain-text"
                                          )}
                                        </RadioGroupItem>
                                        <RadioGroupItem
                                          value={String(
                                            DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT
                                          )}
                                          disabled={!allowEdit}
                                        >
                                          {t(
                                            "instance.external-secret-vault.vault-auth-type.approle.secret-env-name"
                                          )}
                                        </RadioGroupItem>
                                      </RadioGroup>
                                    )}
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
                                  </FormField>
                                </div>
                              )}
                              {/* Vault TLS config */}
                              <FormField
                                title={
                                  <>
                                    {t(
                                      "instance.external-secret-vault.vault-tls-config"
                                    )}
                                  </>
                                }
                              >
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
                              </FormField>
                              {/* Engine name */}
                              <FormField
                                title={
                                  <>
                                    {t(
                                      "instance.external-secret-vault.vault-secret-engine-name"
                                    )}{" "}
                                    <span className="text-error">*</span>
                                  </>
                                }
                                description={
                                  <>
                                    {t(
                                      "instance.external-secret-vault.vault-secret-engine-tips"
                                    )}
                                  </>
                                }
                              >
                                <Input
                                  value={
                                    dataSource.externalSecret.engineName ?? ""
                                  }
                                  required
                                  className="w-full"
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
                              </FormField>
                            </div>
                          )}

                          {/* Azure Key Vault URL */}
                          {passwordType ===
                            DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT && (
                            <FormField
                              title={
                                <>
                                  {t(
                                    "instance.external-secret-azure.vault-url"
                                  )}{" "}
                                  <span className="text-error">*</span>
                                </>
                              }
                              description={
                                <>
                                  {t(
                                    "instance.external-secret-azure.vault-url-tips"
                                  )}
                                </>
                              }
                            >
                              <Input
                                value={dataSource.externalSecret.url ?? ""}
                                required
                                className="w-full"
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
                            </FormField>
                          )}

                          {/* Secret name (common) */}
                          <FormField
                            title={
                              <>
                                {secretNameLabel}{" "}
                                <span className="text-error">*</span>
                              </>
                            }
                            description={
                              passwordType ===
                              DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER
                                ? t(
                                    "instance.external-secret-gcp.secret-name-tips"
                                  )
                                : passwordType ===
                                    DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT
                                  ? t(
                                      "instance.external-secret-azure.secret-name-tips"
                                    )
                                  : undefined
                            }
                          >
                            <Input
                              value={dataSource.externalSecret.secretName ?? ""}
                              required
                              className="w-full"
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
                          </FormField>

                          {/* Secret key (not for GCP/Azure) */}
                          {passwordType !==
                            DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER &&
                            passwordType !==
                              DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT && (
                              <FormField
                                title={
                                  <>
                                    {secretKeyLabel}{" "}
                                    <span className="text-error">*</span>
                                  </>
                                }
                              >
                                <Input
                                  value={
                                    dataSource.externalSecret.passwordKeyName ??
                                    ""
                                  }
                                  required
                                  className="w-full"
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
                              </FormField>
                            )}
                        </div>
                      )}

                    {/* Redis Sentinel fields */}
                    {basicInfo.engine === Engine.REDIS &&
                      dataSource.redisType ===
                        DataSource_RedisType.SENTINEL && (
                        <>
                          <FormField
                            title={
                              <>
                                Master Name{" "}
                                <span className="text-error">*</span>
                              </>
                            }
                          >
                            <Input
                              value={dataSource.masterName ?? ""}
                              className="w-full"
                              disabled={!allowEdit}
                              onChange={(e) =>
                                update({ masterName: e.target.value })
                              }
                            />
                          </FormField>
                          <FormField title={<>Master Username</>}>
                            <Input
                              value={dataSource.masterUsername ?? ""}
                              className="w-full"
                              disabled={!allowEdit}
                              onChange={(e) =>
                                update({ masterUsername: e.target.value })
                              }
                            />
                          </FormField>
                          <FormField title={<>Master Password</>}>
                            <div>
                              {!isCreating && allowUsingEmptyPassword && (
                                <label className="flex items-center gap-x-1.5 mb-2 text-sm cursor-pointer">
                                  <Checkbox
                                    checked={
                                      dataSource.useEmptyMasterPassword ?? false
                                    }
                                    disabled={!allowEdit}
                                    onCheckedChange={(checked) => {
                                      update({
                                        useEmptyMasterPassword: checked,
                                        updatedMasterPassword: checked
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
                          </FormField>
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
              <RadioGroup
                className="sm:col-span-3 sm:col-start-1 textlabel gap-x-4"
                value={String(dataSource.authenticationType)}
                onValueChange={(value) =>
                  update({
                    authenticationType: Number(
                      value
                    ) as DataSource_AuthenticationType,
                  })
                }
              >
                {supportedAuthenticationTypes.map((item) => (
                  <RadioGroupItem
                    key={item.value}
                    value={String(item.value)}
                    disabled={!allowEdit}
                  >
                    {item.label}
                  </RadioGroupItem>
                ))}
              </RadioGroup>
              <CredentialSourceForm
                dataSource={dataSource}
                engine={basicInfo.engine}
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
              <FormField
                className="sm:col-span-3 sm:col-start-1"
                title={<>{t("data-source.ssh.private-key")}</>}
              >
                <div className="flex gap-x-2 text-sm">
                  <span className="textinfolabel">
                    {t("data-source.snowflake-keypair-tip")}
                  </span>
                  <LearnMoreLink
                    href="https://docs.snowflake.com/en/user-guide/key-pair-auth"
                    className="text-sm text-accent"
                  />
                </div>
                <textarea
                  value={dataSource.authenticationPrivateKey ?? ""}
                  disabled={!allowEdit}
                  className="w-full h-32 whitespace-pre-wrap rounded-sm border border-control-border p-2 text-sm font-mono"
                  placeholder={`-----BEGIN PRIVATE KEY-----\nMIIEvQ...\n-----END PRIVATE KEY-----`}
                  onChange={(e) =>
                    update({ authenticationPrivateKey: e.target.value })
                  }
                />
              </FormField>
              <FormField
                className="sm:col-span-3 sm:col-start-1"
                title={<>{t("data-source.private-key-passphrase")}</>}
                description={<>{t("data-source.private-key-passphrase-tip")}</>}
              >
                <Input
                  value={dataSource.authenticationPrivateKeyPassphrase ?? ""}
                  type="password"
                  className="w-full"
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
              </FormField>
            </>
          )}

          {/* Databricks */}
          {basicInfo.engine === Engine.DATABRICKS && (
            <>
              <FormField
                title={
                  <>
                    Warehouse ID <span className="text-error">*</span>
                  </>
                }
              >
                <Input
                  value={dataSource.warehouseId ?? ""}
                  disabled={!allowEdit}
                  onChange={(e) => update({ warehouseId: e.target.value })}
                />
              </FormField>
              <FormField
                title={
                  <>
                    Token <span className="text-error">*</span>
                  </>
                }
              >
                <Input
                  type="password"
                  value={dataSource.updatedToken}
                  className="w-full"
                  autoComplete="off"
                  disabled={!allowEdit}
                  placeholder={
                    isCreating
                      ? "personal access token"
                      : t("instance.token-write-only")
                  }
                  onChange={(e) =>
                    update({ updatedToken: e.target.value.trim() })
                  }
                />
              </FormField>
            </>
          )}

          {/* MongoDB authentication database */}
          {showAuthenticationDatabase && (
            <FormField
              className="sm:col-span-3 sm:col-start-1"
              title={<>{t("instance.authentication-database")}</>}
            >
              <Input
                className="w-full"
                autoComplete="off"
                placeholder="admin"
                disabled={!allowEdit}
                value={dataSource.authenticationDatabase ?? ""}
                onChange={(e) =>
                  update({ authenticationDatabase: e.target.value.trim() })
                }
              />
            </FormField>
          )}

          {/* Read-only replica host/port */}
          {dataSource.type === DataSourceType.READ_ONLY &&
            (hasReadonlyReplicaHost || hasReadonlyReplicaPort) && (
              <>
                {hasReadonlyReplicaHost && (
                  <FormField
                    className="sm:col-span-3 sm:col-start-1"
                    title={<>{t("data-source.read-replica-host")}</>}
                  >
                    <Input
                      className="w-full"
                      autoComplete="off"
                      value={dataSource.host}
                      disabled={!allowEdit}
                      onChange={(e) => handleHostInput(e.target.value)}
                    />
                  </FormField>
                )}
                {hasReadonlyReplicaPort && (
                  <FormField
                    className="sm:col-span-3 sm:col-start-1"
                    title={<>{t("data-source.read-replica-port")}</>}
                  >
                    <Input
                      className="w-full"
                      autoComplete="off"
                      value={dataSource.port}
                      disabled={!allowEdit}
                      onChange={(e) => {
                        if (e.target.value && !onlyAllowNumber(e.target.value))
                          return;
                        handlePortInput(e.target.value);
                      }}
                    />
                  </FormField>
                )}
              </>
            )}

          {/* Database field */}
          {showDatabase && (
            <FormField
              className="sm:col-span-3 sm:col-start-1"
              title={<>{t("common.database")}</>}
            >
              <Input
                value={dataSource.database ?? ""}
                className="w-full"
                disabled={!allowEdit}
                placeholder={t("common.database")}
                onChange={(e) => update({ database: e.target.value })}
              />
            </FormField>
          )}
        </>
      )}

      {/* Connection options (SSL, SSH, Extra params) */}
      {!hideOptions && (
        <>
          {/* SSL */}
          {showSSL && isPasswordAuth && (
            <FormField
              className="sm:col-span-3 sm:col-start-1"
              title={
                <span className="flex items-center justify-start gap-x-2">
                  {t("data-source.ssl.connection-security")}
                  {onOpenInfoPanel && hasSslInfo && (
                    <button
                      type="button"
                      className="inline-flex items-center gap-x-0.5 text-accent text-xs"
                      onClick={() => onOpenInfoPanel("ssl")}
                    >
                      <Info className="size-3.5" />
                    </button>
                  )}
                </span>
              }
            >
              <SslCertificateForm
                useSsl={dataSource.useSsl}
                onUseSslChange={(useSsl) => {
                  if (useSsl) {
                    update({
                      useSsl: true,
                      updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                        useSsl: true,
                      }),
                    });
                    return;
                  }
                  setLocalTlsCaSource("SYSTEM_TRUST");
                  setLocalTlsClientCertSource("NONE");
                  update({ ...disableLocalTls(dataSource), updateSsl: true });
                }}
                posture={localTlsPosture}
                onPostureChange={onLocalTlsPostureChange}
                isSaaSMode={isSaaSMode}
                caSource={localTlsCaSource}
                onCaSourceChange={(source) => {
                  setLocalTlsCaSource(source);
                  update({
                    ...applyLocalTlsCaSource(dataSource, source),
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      ca: true,
                    }),
                  });
                }}
                clientCertSource={localTlsClientCertSource}
                onClientCertSourceChange={(source) => {
                  setLocalTlsClientCertSource(source);
                  update({
                    ...applyLocalTlsClientCertSource(dataSource, source),
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      clientCert: true,
                    }),
                  });
                }}
                verify={dataSource.verifyTlsCertificate}
                onVerifyChange={(val) => update({ verifyTlsCertificate: val })}
                ca={dataSource.sslCa}
                onCaChange={(val) =>
                  update({
                    sslCa: val,
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      ca: true,
                    }),
                  })
                }
                caPath={dataSource.sslCaPath}
                hasCa={dataSource.sslCaSet}
                hasCaPath={dataSource.sslCaPathSet}
                onCaPathChange={(val) =>
                  update({
                    sslCaPath: val,
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      ca: true,
                    }),
                  })
                }
                cert={dataSource.sslCert}
                hasCert={dataSource.sslCertSet}
                onCertChange={(val) =>
                  update({
                    sslCert: val,
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      clientCert: true,
                    }),
                  })
                }
                certPath={dataSource.sslCertPath}
                hasCertPath={dataSource.sslCertPathSet}
                onCertPathChange={(val) =>
                  update({
                    sslCertPath: val,
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      clientCert: true,
                    }),
                  })
                }
                sslKey={dataSource.sslKey}
                hasKey={dataSource.sslKeySet}
                onKeyChange={(val) =>
                  update({
                    sslKey: val,
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      clientCert: true,
                    }),
                  })
                }
                keyPath={dataSource.sslKeyPath}
                hasKeyPath={dataSource.sslKeyPathSet}
                onKeyPathChange={(val) =>
                  update({
                    sslKeyPath: val,
                    updateSsl: mergeTlsUpdateState(dataSource.updateSsl, {
                      clientCert: true,
                    }),
                  })
                }
                engineType={basicInfo.engine}
                disabled={!allowEdit}
              />
            </FormField>
          )}

          {/* SSH */}
          {!hideAdvancedFeatures && showSSH && isPasswordAuth && (
            <FormField
              className="sm:col-span-3 sm:col-start-1"
              title={
                <span className="flex flex-row items-center gap-x-1">
                  {t("data-source.ssh-connection")}
                  {onOpenInfoPanel && hasSshInfo && (
                    <button
                      type="button"
                      className="inline-flex items-center gap-x-0.5 text-accent text-xs"
                      onClick={() => onOpenInfoPanel("ssh")}
                    >
                      <Info className="size-3.5" />
                    </button>
                  )}
                </span>
              }
            >
              <SshConnectionForm
                value={dataSource}
                instance={instance}
                disabled={!allowEdit}
                onChange={handleSSHChange}
              />
            </FormField>
          )}

          {/* Extra connection parameters */}
          {hasExtraParameters && (
            <FormField
              className="sm:col-span-3 sm:col-start-1"
              title={t("data-source.extra-params.self")}
              description={t("data-source.extra-params.description")}
            >
              <FormControlGroup className="mt-2">
                {allowEdit && (
                  <FormControlRow>
                    <Input
                      value={newParamKey}
                      className="min-w-0 flex-1"
                      placeholder={t("instance.parameter-name-placeholder")}
                      onChange={(e) => setNewParamKey(e.target.value)}
                    />
                    <Input
                      value={newParamValue}
                      className="min-w-0 flex-1"
                      placeholder={t("instance.parameter-value-placeholder")}
                      onChange={(e) => setNewParamValue(e.target.value)}
                    />
                    <Button
                      variant="outline"
                      className="shrink-0"
                      disabled={!newParamKey.trim()}
                      onClick={addNewParameter}
                    >
                      Add
                    </Button>
                  </FormControlRow>
                )}

                {extraConnectionParamsList.map((param, index) => (
                  <FormControlRow key={param.key}>
                    <Input
                      className="min-w-0 flex-1"
                      value={param.key}
                      disabled={!allowEdit}
                      placeholder="Parameter name"
                      onChange={(e) =>
                        updateExtraConnectionParamKey(index, e.target.value)
                      }
                    />
                    <Input
                      className="min-w-0 flex-1"
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
                        className="shrink-0"
                        onClick={() => removeExtraConnectionParam(index)}
                      >
                        Remove
                      </Button>
                    )}
                  </FormControlRow>
                ))}
              </FormControlGroup>

              {extraConnectionParamsList.length === 0 && (
                <div className="textinfolabel text-sm italic mt-2">
                  {allowEdit
                    ? t("instance.no-params-yet-add-above")
                    : t("instance.no-extra-params-configured")}
                </div>
              )}
            </FormField>
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
      <RadioGroup
        className="textlabel mb-2 gap-x-4"
        value={mode}
        onValueChange={(value) =>
          handleModeChange(value as "sid" | "serviceName")
        }
      >
        <RadioGroupItem value="sid" disabled={!allowEdit}>
          SID
        </RadioGroupItem>
        <RadioGroupItem value="serviceName" disabled={!allowEdit}>
          Service Name
        </RadioGroupItem>
      </RadioGroup>
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
