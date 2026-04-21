import { cloneDeep } from "lodash-es";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";

export const LOCAL_TLS_SOURCE_DISABLED = "DISABLED" as const;
export const LOCAL_TLS_SOURCE_INLINE_PEM = "INLINE_PEM" as const;
export const LOCAL_TLS_SOURCE_FILE_PATH = "FILE_PATH" as const;

export type LocalTlsSource =
  | typeof LOCAL_TLS_SOURCE_DISABLED
  | typeof LOCAL_TLS_SOURCE_INLINE_PEM
  | typeof LOCAL_TLS_SOURCE_FILE_PATH;

export const LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST = "SYSTEM_TRUST" as const;
export const LOCAL_TLS_CA_SOURCE_INLINE_PEM = "INLINE_PEM" as const;
export const LOCAL_TLS_CA_SOURCE_FILE_PATH = "FILE_PATH" as const;

export const LOCAL_TLS_CLIENT_CERT_SOURCE_NONE = "NONE" as const;
export const LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM = "INLINE_PEM" as const;
export const LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH = "FILE_PATH" as const;

export type LocalTlsCaSource =
  | typeof LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST
  | typeof LOCAL_TLS_CA_SOURCE_INLINE_PEM
  | typeof LOCAL_TLS_CA_SOURCE_FILE_PATH;

export type LocalTlsClientCertSource =
  | typeof LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
  | typeof LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM
  | typeof LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH;

export const LOCAL_TLS_POSTURE_DISABLED = "DISABLED" as const;
export const LOCAL_TLS_POSTURE_TLS = "TLS" as const;
export const LOCAL_TLS_POSTURE_MUTUAL_TLS = "MUTUAL_TLS" as const;

export type LocalTlsPosture =
  | typeof LOCAL_TLS_POSTURE_DISABLED
  | typeof LOCAL_TLS_POSTURE_TLS
  | typeof LOCAL_TLS_POSTURE_MUTUAL_TLS;

export const SSL_UPDATE_MASK_FIELDS = [
  "use_ssl",
  "ssl_ca",
  "ssl_cert",
  "ssl_key",
  "ssl_ca_path",
  "ssl_cert_path",
  "ssl_key_path",
] as const;

type LocalTlsDataSource = Partial<
  Pick<
    DataSource,
    | "useSsl"
    | "sslCa"
    | "sslCert"
    | "sslKey"
    | "sslCaPath"
    | "sslCertPath"
    | "sslKeyPath"
    | "sslCaSet"
    | "sslCertSet"
    | "sslKeySet"
    | "sslCaPathSet"
    | "sslCertPathSet"
    | "sslKeyPathSet"
  >
>;

const clearLocalTlsCaFields = (ds: DataSource): void => {
  ds.sslCa = "";
  ds.sslCaPath = "";
  ds.sslCaSet = false;
  ds.sslCaPathSet = false;
};

const clearLocalTlsClientCertFields = (ds: DataSource): void => {
  ds.sslCert = "";
  ds.sslKey = "";
  ds.sslCertPath = "";
  ds.sslKeyPath = "";
  ds.sslCertSet = false;
  ds.sslKeySet = false;
  ds.sslCertPathSet = false;
  ds.sslKeyPathSet = false;
};

export const hasSslConfig = (ds: LocalTlsDataSource | undefined): boolean => {
  if (!ds) return false;
  return !!(
    ds.useSsl ||
    ds.sslCa ||
    ds.sslCert ||
    ds.sslKey ||
    ds.sslCaPath ||
    ds.sslCertPath ||
    ds.sslKeyPath ||
    ds.sslCaSet ||
    ds.sslCertSet ||
    ds.sslKeySet ||
    ds.sslCaPathSet ||
    ds.sslCertPathSet ||
    ds.sslKeyPathSet
  );
};

export const getLocalTlsCaSource = (
  ds: LocalTlsDataSource | undefined
): LocalTlsCaSource => {
  if (!ds?.useSsl) return LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST;
  if (ds.sslCaPath || ds.sslCaPathSet) {
    return LOCAL_TLS_CA_SOURCE_FILE_PATH;
  }
  if (ds.sslCa || ds.sslCaSet) {
    return LOCAL_TLS_CA_SOURCE_INLINE_PEM;
  }
  return LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST;
};

export const getLocalTlsClientCertSource = (
  ds: LocalTlsDataSource | undefined
): LocalTlsClientCertSource => {
  if (!ds?.useSsl) return LOCAL_TLS_CLIENT_CERT_SOURCE_NONE;
  if (
    ds.sslCertPath ||
    ds.sslKeyPath ||
    ds.sslCertPathSet ||
    ds.sslKeyPathSet
  ) {
    return LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH;
  }
  if (ds.sslCert || ds.sslKey || ds.sslCertSet || ds.sslKeySet) {
    return LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM;
  }
  return LOCAL_TLS_CLIENT_CERT_SOURCE_NONE;
};

export const getLocalTlsPosture = (
  ds: LocalTlsDataSource | undefined
): LocalTlsPosture => {
  if (!ds?.useSsl) {
    return LOCAL_TLS_POSTURE_DISABLED;
  }
  return getLocalTlsClientCertSource(ds) === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
    ? LOCAL_TLS_POSTURE_TLS
    : LOCAL_TLS_POSTURE_MUTUAL_TLS;
};

export const isLocalTlsClientIdentitySupported = (engine: Engine): boolean => {
  return engine !== Engine.MSSQL;
};

export const applyLocalTlsCaSource = (
  ds: DataSource,
  source: LocalTlsCaSource
): DataSource => {
  const next = cloneDeep(ds);
  switch (source) {
    case LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST:
      next.useSsl = true;
      clearLocalTlsCaFields(next);
      break;
    case LOCAL_TLS_CA_SOURCE_INLINE_PEM:
      next.useSsl = true;
      next.sslCaPath = "";
      next.sslCaPathSet = false;
      break;
    case LOCAL_TLS_CA_SOURCE_FILE_PATH:
      next.useSsl = true;
      next.sslCa = "";
      next.sslCaSet = false;
      break;
  }
  return next;
};

export const applyLocalTlsClientCertSource = (
  ds: DataSource,
  source: LocalTlsClientCertSource
): DataSource => {
  const next = cloneDeep(ds);
  switch (source) {
    case LOCAL_TLS_CLIENT_CERT_SOURCE_NONE:
      next.useSsl = true;
      clearLocalTlsClientCertFields(next);
      break;
    case LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM:
      next.useSsl = true;
      next.sslCertPath = "";
      next.sslKeyPath = "";
      next.sslCertPathSet = false;
      next.sslKeyPathSet = false;
      break;
    case LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH:
      next.useSsl = true;
      next.sslCert = "";
      next.sslKey = "";
      next.sslCertSet = false;
      next.sslKeySet = false;
      break;
  }
  return next;
};

export const applyLocalTlsPosture = (
  ds: DataSource,
  posture: LocalTlsPosture
): DataSource => {
  const next = cloneDeep(ds);
  switch (posture) {
    case LOCAL_TLS_POSTURE_DISABLED:
      return disableLocalTls(next);
    case LOCAL_TLS_POSTURE_TLS:
      next.useSsl = true;
      clearLocalTlsClientCertFields(next);
      return next;
    case LOCAL_TLS_POSTURE_MUTUAL_TLS:
      next.useSsl = true;
      return next;
  }
};

export const disableLocalTls = (ds: DataSource): DataSource => {
  const next = cloneDeep(ds);
  next.useSsl = false;
  next.sslCa = "";
  next.sslCert = "";
  next.sslKey = "";
  next.sslCaPath = "";
  next.sslCertPath = "";
  next.sslKeyPath = "";
  next.sslCaSet = false;
  next.sslCertSet = false;
  next.sslKeySet = false;
  next.sslCaPathSet = false;
  next.sslCertPathSet = false;
  next.sslKeyPathSet = false;
  return next;
};

export const getLocalTlsSource = (
  ds: LocalTlsDataSource | undefined
): LocalTlsSource => {
  if (!ds?.useSsl) return LOCAL_TLS_SOURCE_DISABLED;
  if (
    ds.sslCaPath ||
    ds.sslCertPath ||
    ds.sslKeyPath ||
    ds.sslCaPathSet ||
    ds.sslCertPathSet ||
    ds.sslKeyPathSet
  ) {
    return LOCAL_TLS_SOURCE_FILE_PATH;
  }
  return LOCAL_TLS_SOURCE_INLINE_PEM;
};

export const applyLocalTlsSource = (
  ds: DataSource,
  source: LocalTlsSource
): DataSource => {
  const next = cloneDeep(ds);
  switch (source) {
    case LOCAL_TLS_SOURCE_DISABLED:
      return disableLocalTls(next);
    case LOCAL_TLS_SOURCE_INLINE_PEM:
      next.useSsl = true;
      next.sslCaPath = "";
      next.sslCertPath = "";
      next.sslKeyPath = "";
      next.sslCaPathSet = false;
      next.sslCertPathSet = false;
      next.sslKeyPathSet = false;
      break;
    case LOCAL_TLS_SOURCE_FILE_PATH:
      next.useSsl = true;
      next.sslCa = "";
      next.sslCert = "";
      next.sslKey = "";
      next.sslCaSet = false;
      next.sslCertSet = false;
      next.sslKeySet = false;
      break;
  }
  return next;
};

export type { LocalTlsDataSource };
