import { cloneDeep } from "lodash-es";
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
    | "hasSslCaPath"
    | "hasSslCertPath"
    | "hasSslKeyPath"
  >
>;

const clearLocalTlsCaFields = (ds: DataSource): void => {
  ds.sslCa = "";
  ds.sslCaPath = "";
  ds.hasSslCaPath = false;
};

const clearLocalTlsClientCertFields = (ds: DataSource): void => {
  ds.sslCert = "";
  ds.sslKey = "";
  ds.sslCertPath = "";
  ds.sslKeyPath = "";
  ds.hasSslCertPath = false;
  ds.hasSslKeyPath = false;
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
    ds.hasSslCaPath ||
    ds.hasSslCertPath ||
    ds.hasSslKeyPath
  );
};

export const getLocalTlsCaSource = (
  ds: LocalTlsDataSource | undefined
): LocalTlsCaSource => {
  if (!ds?.useSsl) return LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST;
  if (ds.sslCaPath || ds.hasSslCaPath) {
    return LOCAL_TLS_CA_SOURCE_FILE_PATH;
  }
  if (ds.sslCa) {
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
    ds.hasSslCertPath ||
    ds.hasSslKeyPath
  ) {
    return LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH;
  }
  if (ds.sslCert || ds.sslKey) {
    return LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM;
  }
  return LOCAL_TLS_CLIENT_CERT_SOURCE_NONE;
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
      next.hasSslCaPath = false;
      break;
    case LOCAL_TLS_CA_SOURCE_FILE_PATH:
      next.useSsl = true;
      next.sslCa = "";
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
      next.hasSslCertPath = false;
      next.hasSslKeyPath = false;
      break;
    case LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH:
      next.useSsl = true;
      next.sslCert = "";
      next.sslKey = "";
      break;
  }
  return next;
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
  next.hasSslCaPath = false;
  next.hasSslCertPath = false;
  next.hasSslKeyPath = false;
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
    ds.hasSslCaPath ||
    ds.hasSslCertPath ||
    ds.hasSslKeyPath
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
      next.hasSslCaPath = false;
      next.hasSslCertPath = false;
      next.hasSslKeyPath = false;
      break;
    case LOCAL_TLS_SOURCE_FILE_PATH:
      next.useSsl = true;
      next.sslCa = "";
      next.sslCert = "";
      next.sslKey = "";
      break;
  }
  return next;
};

export type { LocalTlsDataSource };
