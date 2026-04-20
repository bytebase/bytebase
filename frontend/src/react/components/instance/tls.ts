import { cloneDeep } from "lodash-es";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";

export const LOCAL_TLS_SOURCE_DISABLED = "DISABLED" as const;
export const LOCAL_TLS_SOURCE_INLINE_PEM = "INLINE_PEM" as const;
export const LOCAL_TLS_SOURCE_FILE_PATH = "FILE_PATH" as const;

export type LocalTlsSource =
  | typeof LOCAL_TLS_SOURCE_DISABLED
  | typeof LOCAL_TLS_SOURCE_INLINE_PEM
  | typeof LOCAL_TLS_SOURCE_FILE_PATH;

export const SSL_UPDATE_MASK_FIELDS = [
  "use_ssl",
  "ssl_ca",
  "ssl_cert",
  "ssl_key",
  "ssl_ca_path",
  "ssl_cert_path",
  "ssl_key_path",
] as const;

type LocalTlsDataSource = Pick<
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
>;

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
      break;
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
