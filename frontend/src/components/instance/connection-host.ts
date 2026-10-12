import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSource_AuthenticationType } from "@/types/proto-es/v1/instance_service_pb";

type HostFields = {
  host: string;
  sshHost?: string;
  authenticationType?: DataSource_AuthenticationType;
};

// Reduces what users type into the host field to the hostname or IP: a bare
// host, a host:port pair, or an endpoint URL.
const dialedHost = (raw: string): string => {
  const host = raw.trim().toLowerCase();
  if (host.includes("://")) {
    try {
      return new URL(host).hostname.replace(/^\[|\]$/g, "");
    } catch {
      return host;
    }
  }
  const unbracketed = host.replace(/^\[|\]$/g, "");
  if (isIPv6(unbracketed)) return unbracketed;
  const hostPort = /^(\[[^\]]+\]|[^:]+):\d+$/.exec(host);
  return hostPort ? hostPort[1].replace(/^\[|\]$/g, "") : host;
};

const parseIPv4 = (host: string): number[] | undefined => {
  const parts = host.split(".");
  if (parts.length !== 4 || !parts.every((part) => /^\d{1,3}$/.test(part))) {
    return undefined;
  }
  const octets = parts.map(Number);
  return octets.every((octet) => octet <= 255) ? octets : undefined;
};

const isIPv6 = (host: string): boolean =>
  /^[0-9a-f:.]+$/.test(host) &&
  (host.includes("::") || /^([0-9a-f]{1,4}:){7}[0-9a-f]{1,4}$/.test(host));

const isUnroutableIPv4 = ([a, b]: number[]): boolean =>
  a === 0 ||
  a === 10 ||
  a === 127 ||
  (a === 100 && b >= 64 && b <= 127) ||
  (a === 169 && b === 254) ||
  (a === 172 && b >= 16 && b <= 31) ||
  (a === 192 && b === 168);

const isUnroutableAddress = (host: string): boolean => {
  const ipv4 = parseIPv4(host);
  if (ipv4) return isUnroutableIPv4(ipv4);
  if (!isIPv6(host)) return false;
  const mapped = /^::ffff:(\d+\.\d+\.\d+\.\d+)$/.exec(host);
  if (mapped) {
    const mappedIPv4 = parseIPv4(mapped[1]);
    return !!mappedIPv4 && isUnroutableIPv4(mappedIPv4);
  }
  // Loopback, unspecified, unique local (fc00::/7), link-local (fe80::/10).
  return host === "::1" || host === "::" || /^f[cd]|^fe[89ab]/.test(host);
};

const localNameSuffixes = [".localhost", ".local", ".internal", ".home.arpa"];

const isLocalName = (raw: string): boolean => {
  const host = raw.replace(/\.$/, "");
  if (!host) return false;
  if (host === "localhost" || host === "host.docker.internal") return true;
  if (localNameSuffixes.some((suffix) => host.endsWith(suffix))) return true;
  return !/[.:/]/.test(host);
};

// The host is an account locator, a Cloud SQL connection name, or absent on
// these, not a name Bytebase resolves.
const dialsHostname = (engine: Engine, dataSource: HostFields): boolean =>
  ![
    Engine.SNOWFLAKE,
    Engine.BIGQUERY,
    Engine.SPANNER,
    Engine.DYNAMODB,
  ].includes(engine) &&
  dataSource.authenticationType !==
    DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;

// Mirrors isPrivateOrLocalHost in backend/api/v1/instance_service.go, which
// decides the connection test on Bytebase Cloud. Keep the two test tables in
// step when either rule changes.
export const isPrivateOrLocalHost = (
  engine: Engine,
  dataSource: HostFields
): boolean => {
  if (dataSource.sshHost) return false;
  const host = dialedHost(dataSource.host);
  if (!host) return false;
  if (isUnroutableAddress(host)) return true;
  return dialsHostname(engine, dataSource) && isLocalName(host);
};
