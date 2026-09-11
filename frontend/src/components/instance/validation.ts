import { isEqual } from "lodash-es";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType as Auth,
  type DataSource,
  DataSource_RedisType,
  DataSourceExternalSecret_AuthType as SecretAuth,
  DataSourceExternalSecret_SecretType as SecretType,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";

export type ValidationErrors = Record<string, string>;

// This catches malformed envelopes and base64 before a request. Certificate
// parsing, key matching, and trust verification remain authoritative on the server.
function hasPem(value: string, kind: "certificate" | "key"): boolean {
  const pattern =
    /-----BEGIN ([A-Z ]+)-----\s+([A-Za-z0-9+/=\s]+?)\s+-----END \1-----/g;
  for (const match of value.matchAll(pattern)) {
    if (
      kind === "certificate"
        ? match[1] !== "CERTIFICATE"
        : !["PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY"].includes(
            match[1]
          )
    )
      continue;
    try {
      const decoded = atob(match[2].replace(/\s/g, ""));
      if (decoded.length > 2 && decoded.charCodeAt(0) === 0x30) return true;
    } catch {
      // Try another PEM block in a certificate bundle.
    }
  }
  return false;
}

export function validateDataSource(
  ds: EditDataSource,
  {
    engine,
    isSaaSMode,
    stored,
    keytabResupply = false,
  }: {
    engine: Engine;
    isSaaSMode: boolean;
    stored?: DataSource;
    keytabResupply?: boolean;
  }
): ValidationErrors {
  const errors: ValidationErrors = {};
  const require = (field: string, value: string | undefined) => {
    if (!value?.trim()) errors[field] = "required";
  };
  const gcpEngine = engine === Engine.BIGQUERY || engine === Engine.SPANNER;
  if (gcpEngine) {
    if (!/^[a-z\d.:-]+$/.test(ds.projectId)) errors.projectId = "gcp-project";
    if (engine === Engine.SPANNER && !/^[a-z\d-]+$/.test(ds.instanceId))
      errors.instanceId = "gcp-instance";
  } else if (engine !== Engine.DYNAMODB) {
    require("host", ds.host);
  }
  if (
    ds.authenticationType === Auth.GOOGLE_CLOUD_SQL_IAM &&
    !gcpEngine &&
    !/.+:.+:.+/.test(ds.host)
  )
    errors.host = "cloud-sql-host";
  if (
    ds.authenticationType === Auth.AWS_RDS_IAM &&
    (engine !== Engine.DYNAMODB || ds.iamExtension?.case === "awsCredential")
  )
    require("region", ds.region);
  if (engine === Engine.ORACLE && !ds.sid && !ds.serviceName)
    errors.serviceName = "oracle-service";
  if (engine === Engine.DATABRICKS) {
    require("warehouseId", ds.warehouseId);
    if (ds.pendingCreate) require("updatedToken", ds.updatedToken);
  }
  if (engine === Engine.REDIS && ds.redisType === DataSource_RedisType.SENTINEL)
    require("masterName", ds.masterName);

  const krb =
    ds.saslConfig?.mechanism.case === "krbConfig"
      ? ds.saslConfig.mechanism.value
      : undefined;
  if (krb) {
    for (const field of ["primary", "realm", "kdcHost"] as const)
      require(`saslConfig.${field}`, krb[field]);
    if (ds.pendingCreate && !krb.keytab.length)
      errors["saslConfig.keytab"] = "required";
    if (keytabResupply) errors["saslConfig.keytab"] = "keytab-resupply";
  }

  const iam = ds.iamExtension;
  // Credential update masks replace the complete credential. Only the unchanged
  // redacted value can inherit the stored secrets.
  const keepsCredential =
    !ds.pendingCreate &&
    stored?.iamExtension.case &&
    stored.authenticationType === ds.authenticationType &&
    isEqual(stored.iamExtension, iam);
  if (
    isSaaSMode &&
    [Auth.GOOGLE_CLOUD_SQL_IAM, Auth.AWS_RDS_IAM, Auth.AZURE_IAM].includes(
      ds.authenticationType
    ) &&
    !keepsCredential
  ) {
    if (!iam?.case) errors.iamExtension = "specific-credential";
    if (
      iam?.case === "awsCredential" &&
      !iam.value.accessKeyId &&
      !iam.value.roleArn
    ) {
      errors["iamExtension.accessKeyId"] = "aws-credential";
      errors["iamExtension.roleArn"] = "aws-credential";
    }
    if (iam?.case === "gcpCredential")
      require("iamExtension.content", iam.value.content);
    if (iam?.case === "azureCredential") {
      for (const field of ["tenantId", "clientId", "clientSecret"] as const)
        require(`iamExtension.${field}`, iam.value[field]);
    }
  }

  const secret = ds.externalSecret;
  if (secret) {
    if (
      [SecretType.VAULT_KV_V2, SecretType.AZURE_KEY_VAULT].includes(
        secret.secretType
      )
    )
      require("externalSecret.url", secret.url);
    if (secret.secretType === SecretType.VAULT_KV_V2)
      require("externalSecret.engineName", secret.engineName);
    require("externalSecret.secretName", secret.secretName);
    if (
      [SecretType.VAULT_KV_V2, SecretType.AWS_SECRETS_MANAGER].includes(
        secret.secretType
      )
    )
      require("externalSecret.passwordKeyName", secret.passwordKeyName);
    if (secret.authType === SecretAuth.TOKEN)
      require("externalSecret.token", secret.authOption.case === "token"
        ? secret.authOption.value
        : "");
    if (secret.authType === SecretAuth.VAULT_APP_ROLE) {
      const role =
        secret.authOption.case === "appRole"
          ? secret.authOption.value
          : undefined;
      require("externalSecret.roleId", role?.roleId);
      require("externalSecret.secretId", role?.secretId);
    }
  }

  if (ds.useSsl) {
    const caChanged =
      ds.updateSsl === true ||
      (typeof ds.updateSsl === "object" && ds.updateSsl.ca);
    const clientChanged =
      ds.updateSsl === true ||
      (typeof ds.updateSsl === "object" && ds.updateSsl.clientCert);
    const retained = (
      field:
        | "sslCaSet"
        | "sslCaPathSet"
        | "sslCertSet"
        | "sslCertPathSet"
        | "sslKeySet"
        | "sslKeyPathSet"
    ) =>
      !(field === "sslCaSet" || field === "sslCaPathSet"
        ? caChanged
        : clientChanged) && ds[field];
    for (const [inline, path, set, pathSet] of [
      ["sslCa", "sslCaPath", "sslCaSet", "sslCaPathSet"],
      ["sslCert", "sslCertPath", "sslCertSet", "sslCertPathSet"],
      ["sslKey", "sslKeyPath", "sslKeySet", "sslKeyPathSet"],
    ] as const) {
      if ((ds[inline] || retained(set)) && (ds[path] || retained(pathSet)))
        errors[path] = "tls-source-conflict";
      if (ds[path] && !ds[path].startsWith("/")) errors[path] = "absolute-path";
      if (
        ds[inline] &&
        !hasPem(ds[inline], inline === "sslKey" ? "key" : "certificate")
      )
        errors[inline] = inline === "sslKey" ? "pem-key" : "pem-certificate";
    }
    const cert = !!(
      ds.sslCert ||
      ds.sslCertPath ||
      retained("sslCertSet") ||
      retained("sslCertPathSet")
    );
    const key = !!(
      ds.sslKey ||
      ds.sslKeyPath ||
      retained("sslKeySet") ||
      retained("sslKeyPathSet")
    );
    if (cert !== key) {
      errors.sslCert =
        errors.sslCertPath =
        errors.sslKey =
        errors.sslKeyPath =
          "tls-pair";
    }
  }
  if (
    [Engine.MYSQL, Engine.MARIADB, Engine.OCEANBASE, Engine.TIDB].includes(
      engine
    )
  ) {
    for (const key of Object.keys(ds.extraConnectionParameters ?? {})) {
      if (key.trim().toLowerCase() === "allowallfiles")
        errors[`extraConnectionParameters.${key}`] = "forbidden-parameter";
    }
  }
  return errors;
}
