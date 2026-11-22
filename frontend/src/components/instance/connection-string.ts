import { create } from "@bufbuild/protobuf";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AddressSchema,
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceType,
  Vendor,
} from "@/types/proto-es/v1/instance_service_pb";
import { type EditDataSource, updateDataSourceSecret } from "./common";
import { defaultPortForEngine } from "./constants";
import { hasConnectionDatabase } from "./specs";
import {
  applyLocalTlsPosture,
  disableLocalTls,
  LOCAL_TLS_POSTURE_TLS,
} from "./tls";

export type ConnectionStringEngine =
  | Engine.MYSQL
  | Engine.POSTGRES
  | Engine.MSSQL
  | Engine.ORACLE
  | Engine.MARIADB
  | Engine.TIDB
  | Engine.OCEANBASE
  | Engine.STARROCKS
  | Engine.DORIS
  | Engine.COCKROACHDB
  | Engine.REDSHIFT
  | Engine.MONGODB
  | Engine.REDIS
  | Engine.CASSANDRA
  | Engine.ELASTICSEARCH
  | Engine.CLICKHOUSE
  | Engine.SNOWFLAKE
  | Engine.TRINO
  | Engine.HIVE
  | Engine.DATABRICKS;

export type ParsedAddress = {
  host: string;
  port: string;
};

export type ParsedTls =
  | { useSsl: false }
  | { useSsl: true; verifyTlsCertificate: boolean };

export type ParsedConnectionString = {
  host: string;
  port?: string;
  username?: string;
  password?: string;
  database?: string;
  sid?: string;
  serviceName?: string;
  tls?: ParsedTls;
  additionalAddresses?: ParsedAddress[];
  srv?: boolean;
  replicaSet?: string;
  directConnection?: boolean;
  authenticationDatabase?: string;
  redisType?: DataSource_RedisType;
  masterName?: string;
  masterUsername?: string;
  masterPassword?: string;
  warehouseId?: string;
  token?: string;
};

const CONNECTION_STRING_ENGINES = new Set<Engine>([
  Engine.MYSQL,
  Engine.POSTGRES,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.MARIADB,
  Engine.TIDB,
  Engine.OCEANBASE,
  Engine.STARROCKS,
  Engine.DORIS,
  Engine.COCKROACHDB,
  Engine.REDSHIFT,
  Engine.MONGODB,
  Engine.REDIS,
  Engine.CASSANDRA,
  Engine.ELASTICSEARCH,
  Engine.CLICKHOUSE,
  Engine.SNOWFLAKE,
  Engine.TRINO,
  Engine.HIVE,
  Engine.DATABRICKS,
]);

// Cloud consoles copy template tokens in place of real secrets (e.g. Azure
// `{your_password}`, Supabase `[YOUR-PASSWORD]`, MongoDB `<db_password>`,
// CockroachDB `<ENTER-SQL-USER-PASSWORD>`, Databricks `<personal-access-token>`,
// or masked bullets). Treating them as omitted avoids submitting placeholder text.
const PASSWORD_PLACEHOLDER_PATTERN =
  /^(?:\{your[_ -]?password(?:[_ -]?here)?\}|\[your[_ -]?password\]|<(?:(?:your|db)[_ -]?)?password>|<enter[_ -]?sql[_ -]?user[_ -]?password>|<(?:(?:your|personal[_ -]?access|access)[_ -]?)?token>|\u2022+)$/i;

const SCHEME_PREFIX_PATTERN = /^([a-zA-Z][a-zA-Z0-9+.-]*):\/\//;

export function supportsConnectionString(
  engine: Engine
): engine is ConnectionStringEngine {
  return CONNECTION_STRING_ENGINES.has(engine);
}

const decodePercent = (value: string): string => {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
};

const normalizePassword = (value: string | undefined): string | undefined => {
  if (value === undefined) return undefined;
  if (PASSWORD_PLACEHOLDER_PATTERN.test(value.trim())) return undefined;
  return value;
};

const stripSurroundingQuotes = (value: string): string => {
  if (
    (value.startsWith("'") && value.endsWith("'")) ||
    (value.startsWith('"') && value.endsWith('"'))
  ) {
    return value.slice(1, -1).trim();
  }
  return value;
};

const validatePort = (value: string): string | undefined => {
  if (!/^\d+$/.test(value)) return undefined;
  const numeric = Number(value);
  if (!Number.isInteger(numeric) || numeric < 1 || numeric > 65535) {
    return undefined;
  }
  return String(numeric);
};

function parseBooleanValue(value: string | undefined): boolean | undefined {
  if (value === undefined) return undefined;
  switch (value.trim().toLowerCase()) {
    case "true":
    case "1":
    case "yes":
    case "on":
    case "enable":
    case "enabled":
      return true;
    case "false":
    case "0":
    case "no":
    case "off":
    case "disable":
    case "disabled":
      return false;
    default:
      return undefined;
  }
}

function tokenizeCliArgs(input: string): string[] | undefined {
  const tokens: string[] = [];
  let index = 0;
  const length = input.length;

  while (index < length) {
    while (index < length && /\s/.test(input[index])) index++;
    if (index >= length) break;

    let token = "";
    let quote: "'" | '"' | null = null;
    while (index < length) {
      const ch = input[index];
      if (quote === "'") {
        index++;
        if (ch === "'") {
          quote = null;
        } else {
          token += ch;
        }
      } else if (quote === '"') {
        index++;
        if (ch === "\\") {
          if (index >= length) return undefined;
          const next = input[index++];
          token += next === '"' || next === "\\" ? next : `\\${next}`;
        } else if (ch === '"') {
          quote = null;
        } else {
          token += ch;
        }
      } else {
        if (/\s/.test(ch)) break;
        index++;
        if (ch === "'" || ch === '"') {
          quote = ch;
        } else {
          token += ch;
        }
      }
    }
    if (quote !== null) return undefined;
    tokens.push(token);
  }

  return tokens;
}

// Parses `host[:port]` or `[ipv6][:port]`, returning the unbracketed host so
// Go drivers can pass it to `net.JoinHostPort` without double-bracketing.
function parseHostPort(
  input: string
): { host: string; port?: string } | undefined {
  if (!input || /\s/.test(input)) return undefined;
  if (input.startsWith("[")) {
    const closeIndex = input.indexOf("]");
    if (closeIndex <= 1) return undefined;
    const host = input.slice(1, closeIndex);
    if (!host.includes(":")) return undefined;
    const rest = input.slice(closeIndex + 1);
    if (!rest) return { host };
    if (!rest.startsWith(":")) return undefined;
    const port = validatePort(rest.slice(1));
    return port ? { host, port } : undefined;
  }
  if (input.includes("[") || input.includes("]")) return undefined;
  const colonIndex = input.indexOf(":");
  if (colonIndex === -1) {
    return { host: input };
  }
  const host = input.slice(0, colonIndex);
  const portPart = input.slice(colonIndex + 1);
  if (!host || portPart.includes(":")) return undefined;
  const port = validatePort(portPart);
  return port ? { host, port } : undefined;
}

type ParsedUriParts = {
  scheme: string;
  username?: string;
  hasExplicitEmptyUser: boolean;
  password?: string;
  host?: string;
  port?: string;
  hosts: { host: string; port?: string }[];
  path: string;
  query: Map<string, string>;
};

// Custom database schemes (`postgresql:`, `mysql:`, `sqlserver:`, `oracle:`, etc.)
// are non-special in WHATWG URL, so `new URL()` treats their authority as an
// opaque path when credentials contain `@` or `#` or when host is omitted.
function parseStandardUri(
  input: string,
  options?: { allowMultipleHosts?: boolean }
): ParsedUriParts | undefined {
  const schemeMatch = input.match(SCHEME_PREFIX_PATTERN);
  if (!schemeMatch) return undefined;
  const scheme = schemeMatch[1].toLowerCase();
  const rest = input.slice(schemeMatch[0].length);
  const slashOrQueryIndex = rest.search(/[/?]/);
  const beforePathOrQuery =
    slashOrQueryIndex === -1 ? rest : rest.slice(0, slashOrQueryIndex);
  const authorityAtIndex = beforePathOrQuery.lastIndexOf("@");
  const hashSearchStart = authorityAtIndex === -1 ? 0 : authorityAtIndex + 1;
  const hashIndex = rest.indexOf("#", hashSearchStart);
  const withoutFragment = hashIndex === -1 ? rest : rest.slice(0, hashIndex);

  const queryIndex = withoutFragment.indexOf("?");
  const authorityAndPath =
    queryIndex === -1 ? withoutFragment : withoutFragment.slice(0, queryIndex);
  const queryString =
    queryIndex === -1 ? "" : withoutFragment.slice(queryIndex + 1);

  const slashIndex = authorityAndPath.indexOf("/");
  const authority =
    slashIndex === -1
      ? authorityAndPath
      : authorityAndPath.slice(0, slashIndex);
  const path = slashIndex === -1 ? "" : authorityAndPath.slice(slashIndex + 1);

  let username: string | undefined;
  let hasExplicitEmptyUser = false;
  let password: string | undefined;
  let hostPortPart = authority;

  const atIndex = authority.lastIndexOf("@");
  if (atIndex !== -1) {
    const userInfo = authority.slice(0, atIndex);
    hostPortPart = authority.slice(atIndex + 1);
    const colonIndex = userInfo.indexOf(":");
    if (colonIndex === -1) {
      if (userInfo) username = decodePercent(userInfo);
    } else {
      const rawUser = userInfo.slice(0, colonIndex);
      if (rawUser) {
        username = decodePercent(rawUser);
      } else {
        hasExplicitEmptyUser = true;
      }
      password = decodePercent(userInfo.slice(colonIndex + 1));
    }
  }

  const hosts: { host: string; port?: string }[] = [];
  if (hostPortPart) {
    if (options?.allowMultipleHosts) {
      for (const segment of hostPortPart.split(",")) {
        const parsedHostPort = parseHostPort(segment);
        if (!parsedHostPort) return undefined;
        hosts.push(parsedHostPort);
      }
    } else {
      const parsedHostPort = parseHostPort(hostPortPart);
      if (!parsedHostPort) return undefined;
      hosts.push(parsedHostPort);
    }
  }

  const query = new Map<string, string>();
  for (const [key, value] of new URLSearchParams(queryString)) {
    query.set(key.toLowerCase(), value);
  }

  return {
    scheme,
    username,
    hasExplicitEmptyUser,
    password,
    host: hosts[0]?.host,
    port: hosts[0]?.port,
    hosts,
    path,
    query,
  };
}

function parseUriWithJdbcParams(
  input: string
): { uri: ParsedUriParts; params: Map<string, string> } | undefined {
  const schemeMatch = input.match(SCHEME_PREFIX_PATTERN);
  if (!schemeMatch) return undefined;
  const rest = input.slice(schemeMatch[0].length);
  const slashOrQueryIndex = rest.search(/[/?]/);
  const beforePathOrQuery =
    slashOrQueryIndex === -1 ? rest : rest.slice(0, slashOrQueryIndex);
  const authorityAtIndex = beforePathOrQuery.lastIndexOf("@");
  const afterAuthStart = authorityAtIndex === -1 ? 0 : authorityAtIndex + 1;
  const firstQuery = rest.indexOf("?", afterAuthStart);
  const beforeQuery = firstQuery === -1 ? rest : rest.slice(0, firstQuery);
  const semiIndex = beforeQuery.indexOf(";", afterAuthStart);

  if (semiIndex === -1) {
    const uri = parseStandardUri(input);
    return uri ? { uri, params: new Map() } : undefined;
  }

  const queryPart =
    firstQuery === -1 ? "" : rest.slice(firstQuery).split("#", 1)[0];
  const uri = parseStandardUri(
    `${schemeMatch[0]}${rest.slice(0, semiIndex)}${queryPart}`
  );
  if (!uri) return undefined;

  const semiSegment = beforeQuery.slice(semiIndex + 1);
  if (!semiSegment.trim()) {
    return { uri, params: new Map() };
  }
  const params = parseAdoNetKeyValue(semiSegment);
  return params ? { uri, params } : undefined;
}

function parseTlsSkipVerify(
  query: Map<string, string>,
  extraSkip = false
): boolean {
  return (
    extraSkip ||
    parseBooleanValue(query.get("insecure")) === true ||
    parseBooleanValue(query.get("tls_insecure")) === true ||
    parseBooleanValue(query.get("skip_verify")) === true ||
    parseBooleanValue(query.get("tls_skip_verify")) === true ||
    parseBooleanValue(query.get("ssl_verify")) === false ||
    parseBooleanValue(query.get("verify")) === false
  );
}

function parseSchemeOrQueryTls(
  scheme: string,
  query: Map<string, string>,
  {
    tlsSchemes = [],
    plainSchemes = [],
    queryKeys = ["tls", "ssl"],
    skipVerify = false,
  }: {
    tlsSchemes?: string[];
    plainSchemes?: string[];
    queryKeys?: string[];
    skipVerify?: boolean;
  }
): ParsedTls | undefined {
  if (tlsSchemes.includes(scheme)) {
    return { useSsl: true, verifyTlsCertificate: !skipVerify };
  }
  if (plainSchemes.includes(scheme)) {
    return { useSsl: false };
  }
  for (const key of queryKeys) {
    const raw = query.get(key);
    if (raw === undefined) continue;
    const enabled = parseBooleanValue(raw);
    if (enabled === false) return { useSsl: false };
    if (enabled === true) {
      return { useSsl: true, verifyTlsCertificate: !skipVerify };
    }
  }
  return undefined;
}

function parsePostgresTls(
  sslMode: string | undefined,
  ssl?: string,
  verifiedByDefault = false
): ParsedTls | undefined {
  if (sslMode !== undefined) {
    switch (sslMode.trim().toLowerCase()) {
      case "disable":
        return { useSsl: false };
      case "no-verify":
        return { useSsl: true, verifyTlsCertificate: false };
      case "allow":
      case "prefer":
      case "require":
        return { useSsl: true, verifyTlsCertificate: verifiedByDefault };
      case "verify-ca":
      case "verify-full":
        return { useSsl: true, verifyTlsCertificate: true };
      default:
        return undefined;
    }
  }
  if (ssl !== undefined) {
    switch (ssl.trim().toLowerCase()) {
      case "false":
      case "0":
        return { useSsl: false };
      case "true":
      case "1":
        return { useSsl: true, verifyTlsCertificate: verifiedByDefault };
    }
  }
  return undefined;
}

function parseLibpqKeyValue(input: string): Map<string, string> | undefined {
  if (SCHEME_PREFIX_PATTERN.test(input) || /^jdbc:/i.test(input)) {
    return undefined;
  }
  const pairs = new Map<string, string>();
  let index = 0;
  const length = input.length;

  while (index < length) {
    while (index < length && /\s/.test(input[index])) index++;
    if (index >= length) break;

    const keyStart = index;
    while (index < length && /[a-zA-Z0-9_]/.test(input[index])) index++;
    const key = input.slice(keyStart, index);
    if (!/^[a-z][a-z0-9_]*$/.test(key)) return undefined;

    while (index < length && /\s/.test(input[index])) index++;
    if (input[index] !== "=") return undefined;
    index++;
    while (index < length && /\s/.test(input[index])) index++;

    let value = "";
    if (input[index] === "'") {
      index++;
      let closed = false;
      while (index < length) {
        const ch = input[index++];
        if (ch === "\\") {
          if (index >= length) return undefined;
          value += input[index++];
        } else if (ch === "'") {
          closed = true;
          break;
        } else {
          value += ch;
        }
      }
      if (!closed) return undefined;
      if (index < length && !/\s/.test(input[index])) return undefined;
    } else {
      const valueStart = index;
      while (index < length && !/\s/.test(input[index])) {
        if (input[index] === "'") return undefined;
        index++;
      }
      value = input.slice(valueStart, index);
    }
    pairs.set(key, value);
  }

  return pairs.size > 0 ? pairs : undefined;
}

function extractCockroachCliUrl(raw: string): string | undefined {
  const tokens = tokenizeCliArgs(raw);
  if (
    !tokens ||
    tokens[0]?.toLowerCase() !== "cockroach" ||
    tokens[1]?.toLowerCase() !== "sql"
  ) {
    return undefined;
  }
  for (let i = 2; i < tokens.length; i++) {
    const token = tokens[i];
    if (token === "--url") {
      return tokens[i + 1];
    }
    if (token.startsWith("--url=")) {
      return token.slice("--url=".length);
    }
  }
  return undefined;
}

function parsePostgresFamilyConnectionString(
  raw: string,
  {
    schemes,
    defaultPort,
    includeTls,
    requireTls = false,
    verifiedTlsByDefault = false,
    allowJdbc = false,
    allowCockroachCli = false,
  }: {
    schemes: string[];
    defaultPort: string;
    includeTls: boolean;
    requireTls?: boolean;
    verifiedTlsByDefault?: boolean;
    allowJdbc?: boolean;
    allowCockroachCli?: boolean;
  }
): ParsedConnectionString | undefined {
  const trimmed = raw.trim();
  let candidate = trimmed;

  if (allowCockroachCli && /^cockroach\s+sql\b/i.test(trimmed)) {
    const urlArg = extractCockroachCliUrl(trimmed);
    if (!urlArg) return undefined;
    candidate = urlArg;
  } else {
    candidate = stripSurroundingQuotes(trimmed.replace(/^psql\s+/i, "").trim());
  }
  if (!candidate) return undefined;

  if (allowJdbc && /^jdbc:redshift:\/\//i.test(candidate)) {
    candidate = candidate.slice("jdbc:".length);
  }

  const schemeMatch = candidate.match(SCHEME_PREFIX_PATTERN);
  if (schemeMatch) {
    const scheme = schemeMatch[1].toLowerCase();
    if (!schemes.includes(scheme)) return undefined;
    const uri = parseStandardUri(candidate);
    if (!uri || uri.path.includes("/")) return undefined;
    const rawHost =
      uri.host ?? uri.query.get("host") ?? uri.query.get("hostaddr");
    if (!rawHost || /\s/.test(rawHost)) return undefined;
    const host =
      rawHost.startsWith("[") && rawHost.endsWith("]")
        ? rawHost.slice(1, -1)
        : rawHost;
    let port = uri.port;
    if (!port && uri.query.has("port")) {
      port = validatePort(uri.query.get("port") ?? "");
      if (!port) return undefined;
    }
    const database = uri.path
      ? decodePercent(uri.path)
      : uri.query.get("dbname");
    const username =
      uri.username ??
      uri.query.get("user") ??
      (allowJdbc ? uri.query.get("uid") : undefined);
    const password = normalizePassword(
      uri.password ??
        uri.query.get("password") ??
        (allowJdbc ? uri.query.get("pwd") : undefined)
    );
    const tls = includeTls
      ? parsePostgresTls(
          uri.query.get("sslmode"),
          uri.query.get("ssl"),
          verifiedTlsByDefault
        )
      : undefined;
    if (requireTls && tls?.useSsl === false) return undefined;
    return {
      host,
      port: port ?? defaultPort,
      username: username || undefined,
      password,
      database: database || undefined,
      tls,
    };
  }

  const kv = parseLibpqKeyValue(candidate);
  if (!kv) return undefined;
  const rawHost = kv.get("host") || kv.get("hostaddr");
  if (!rawHost || /[\s;=]/.test(rawHost)) return undefined;
  const host =
    rawHost.startsWith("[") && rawHost.endsWith("]")
      ? rawHost.slice(1, -1)
      : rawHost;
  let port = defaultPort;
  if (kv.has("port") && kv.get("port") !== "") {
    const validated = validatePort(kv.get("port") ?? "");
    if (!validated) return undefined;
    port = validated;
  }
  const tls = includeTls
    ? parsePostgresTls(kv.get("sslmode"), kv.get("ssl"), verifiedTlsByDefault)
    : undefined;
  if (requireTls && tls?.useSsl === false) return undefined;
  return {
    host,
    port,
    username: kv.get("user") || undefined,
    password: normalizePassword(kv.get("password")),
    database: kv.get("dbname") || undefined,
    tls,
  };
}

function parseMysqlTls(query: Map<string, string>): ParsedTls | undefined {
  const sslMode = query.get("ssl-mode") ?? query.get("sslmode");
  if (sslMode !== undefined) {
    switch (sslMode.trim().toLowerCase()) {
      case "disabled":
      case "disable":
      case "false":
      case "0":
        return { useSsl: false };
      case "preferred":
      case "prefer":
      case "required":
      case "require":
        return { useSsl: true, verifyTlsCertificate: false };
      case "verify_ca":
      case "verify-ca":
      case "verify_identity":
      case "verify-identity":
      case "verify-full":
        return { useSsl: true, verifyTlsCertificate: true };
    }
  }
  const sslAccept = query.get("sslaccept");
  if (sslAccept !== undefined) {
    switch (sslAccept.trim().toLowerCase()) {
      case "strict":
        return { useSsl: true, verifyTlsCertificate: true };
      case "accept_invalid_certs":
        return { useSsl: true, verifyTlsCertificate: false };
    }
  }
  const tls = query.get("tls");
  if (tls !== undefined) {
    switch (tls.trim().toLowerCase()) {
      case "false":
      case "0":
        return { useSsl: false };
      case "skip-verify":
      case "preferred":
        return { useSsl: true, verifyTlsCertificate: false };
      case "true":
      case "1":
        return { useSsl: true, verifyTlsCertificate: true };
    }
  }
  const ssl = query.get("ssl");
  if (ssl !== undefined) {
    const trimmed = ssl.trim();
    switch (trimmed.toLowerCase()) {
      case "false":
      case "0":
        return { useSsl: false };
      case "true":
      case "1":
        return { useSsl: true, verifyTlsCertificate: false };
    }
    if (trimmed.startsWith("{")) {
      try {
        const parsed: unknown = JSON.parse(trimmed);
        if (
          parsed &&
          typeof parsed === "object" &&
          "rejectUnauthorized" in parsed
        ) {
          return {
            useSsl: true,
            verifyTlsCertificate: parsed.rejectUnauthorized !== false,
          };
        }
      } catch {
        // Ignore non-JSON `ssl` query values.
      }
    }
  }
  return undefined;
}

function parseMysqlFamilyConnectionString(
  raw: string,
  { schemes, defaultPort }: { schemes: string[]; defaultPort: string }
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const schemeMatch = unwrapped.match(SCHEME_PREFIX_PATTERN);
  if (!schemeMatch || !schemes.includes(schemeMatch[1].toLowerCase())) {
    return undefined;
  }
  const uri = parseStandardUri(unwrapped);
  if (!uri || !uri.host || uri.path.includes("/")) return undefined;
  return {
    host: uri.host,
    port: uri.port ?? defaultPort,
    username: uri.username || undefined,
    password: normalizePassword(uri.password),
    tls: parseMysqlTls(uri.query),
  };
}

function parseAdoNetKeyValue(input: string): Map<string, string> | undefined {
  const pairs = new Map<string, string>();
  let index = 0;
  const length = input.length;

  while (index < length) {
    while (
      index < length &&
      (/\s/.test(input[index]) || input[index] === ";")
    ) {
      index++;
    }
    if (index >= length) break;

    const eqIndex = input.indexOf("=", index);
    if (eqIndex === -1) return undefined;
    const rawKey = input.slice(index, eqIndex);
    if (rawKey.includes(";")) return undefined;
    const key = rawKey.trim().toLowerCase().replace(/\s+/g, " ");
    if (!key) return undefined;

    index = eqIndex + 1;
    while (index < length && /\s/.test(input[index])) index++;

    let value = "";
    const quote = input[index];
    if (quote === '"' || quote === "'" || quote === "{") {
      const closing = quote === "{" ? "}" : quote;
      index++;
      let closed = false;
      while (index < length) {
        const ch = input[index++];
        if (ch === closing) {
          if (input[index] === closing) {
            value += closing;
            index++;
          } else {
            closed = true;
            break;
          }
        } else {
          value += ch;
        }
      }
      if (!closed) return undefined;
      while (index < length && /\s/.test(input[index])) index++;
      if (index < length && input[index] !== ";") return undefined;
      if (quote === "{") {
        // Preserve `{your_password}` placeholder recognition before stripping braces.
        const rawBraced = `{${value}}`;
        if (PASSWORD_PLACEHOLDER_PATTERN.test(rawBraced)) {
          value = rawBraced;
        }
      }
    } else {
      const valueStart = index;
      while (index < length && input[index] !== ";") index++;
      value = input.slice(valueStart, index).trim();
    }

    pairs.set(key, value);
    if (input[index] === ";") index++;
  }

  return pairs.size > 0 ? pairs : undefined;
}

function parseMssqlServerValue(
  rawServer: string
): { host: string; port?: string } | undefined {
  const trimmed = rawServer.trim();
  if (!trimmed || /\s/.test(trimmed)) return undefined;
  if (/^(?:np|lpc|via):/i.test(trimmed)) return undefined;
  const withoutProtocol = trimmed.replace(/^tcp:/i, "");
  if (!withoutProtocol) return undefined;

  let port: string | undefined;
  let hostAndInstance = withoutProtocol;
  const commaIndex = withoutProtocol.lastIndexOf(",");
  if (commaIndex !== -1) {
    const before = withoutProtocol.slice(0, commaIndex);
    const after = withoutProtocol.slice(commaIndex + 1);
    const backslashInAfter = after.indexOf("\\");
    const portText =
      backslashInAfter === -1 ? after : after.slice(0, backslashInAfter);
    const trailingInstance =
      backslashInAfter === -1 ? "" : after.slice(backslashInAfter);
    const validated = validatePort(portText);
    if (!validated) return undefined;
    port = validated;
    hostAndInstance = `${before}${trailingInstance}`;
  }

  const backslashIndex = hostAndInstance.indexOf("\\");
  const rawHost =
    backslashIndex === -1
      ? hostAndInstance
      : hostAndInstance.slice(0, backslashIndex);
  const instanceName =
    backslashIndex === -1 ? "" : hostAndInstance.slice(backslashIndex + 1);
  if (!rawHost || (backslashIndex !== -1 && !instanceName)) return undefined;
  if (instanceName.includes("\\")) return undefined;

  let host = rawHost;
  if (rawHost.startsWith("[")) {
    if (!rawHost.endsWith("]") || !rawHost.includes(":")) return undefined;
    host = rawHost.slice(1, -1);
  } else if (rawHost.includes(":")) {
    return undefined;
  }

  return {
    host: instanceName ? `${host}\\${instanceName}` : host,
    port,
  };
}

function parseMssqlTls(
  encrypt: string | undefined,
  trustServerCertificate: string | undefined
): ParsedTls | undefined {
  if (encrypt === undefined) return undefined;
  const normalizedEncrypt = encrypt.trim().toLowerCase();
  if (["false", "no", "optional", "0"].includes(normalizedEncrypt)) {
    return { useSsl: false };
  }
  if (["true", "yes", "mandatory", "strict", "1"].includes(normalizedEncrypt)) {
    const normalizedTrust = trustServerCertificate?.trim().toLowerCase();
    const trust =
      normalizedTrust !== undefined &&
      ["true", "yes", "1"].includes(normalizedTrust);
    return { useSsl: true, verifyTlsCertificate: !trust };
  }
  return undefined;
}

function extractMssqlFromKeyMap(
  pairs: Map<string, string>,
  fallback?: {
    host: string;
    port?: string;
    username?: string;
    password?: string;
  }
): ParsedConnectionString | undefined {
  const serverRaw =
    pairs.get("server") ??
    pairs.get("data source") ??
    pairs.get("datasource") ??
    pairs.get("addr") ??
    pairs.get("address") ??
    pairs.get("network address");
  const server = serverRaw
    ? parseMssqlServerValue(serverRaw)
    : fallback
      ? { host: fallback.host, port: fallback.port }
      : undefined;
  if (!server) return undefined;

  let port = server.port;
  if (!port && pairs.has("port") && pairs.get("port") !== "") {
    port = validatePort(pairs.get("port") ?? "");
    if (!port) return undefined;
  }

  const username =
    pairs.get("user id") ??
    pairs.get("userid") ??
    pairs.get("uid") ??
    pairs.get("user") ??
    fallback?.username;
  const rawPassword =
    pairs.get("password") ?? pairs.get("pwd") ?? fallback?.password;
  const database = pairs.get("initial catalog") ?? pairs.get("database");
  const tls = parseMssqlTls(
    pairs.get("encrypt"),
    pairs.get("trustservercertificate") ?? pairs.get("trust server certificate")
  );

  return {
    host: server.host,
    port: port ?? "1433",
    username: username || undefined,
    password: normalizePassword(rawPassword),
    database: database || undefined,
    tls,
  };
}

function parseMssqlHostWithOptionalInstance(input: string):
  | {
      host: string;
      port?: string;
      username?: string;
      password?: string;
    }
  | undefined {
  let username: string | undefined;
  let password: string | undefined;
  let hostInput = input;
  const atIndex = input.lastIndexOf("@");
  if (atIndex !== -1) {
    const userInfo = input.slice(0, atIndex);
    hostInput = input.slice(atIndex + 1);
    const colonIndex = userInfo.indexOf(":");
    if (colonIndex === -1) {
      if (userInfo) username = decodePercent(userInfo);
    } else {
      const rawUser = userInfo.slice(0, colonIndex);
      if (rawUser) username = decodePercent(rawUser);
      password = decodePercent(userInfo.slice(colonIndex + 1));
    }
  }
  const backslashIndex = hostInput.indexOf("\\");
  if (backslashIndex === -1) {
    const parsedHost = parseHostPort(hostInput);
    return parsedHost ? { ...parsedHost, username, password } : undefined;
  }
  const hostPart = hostInput.slice(0, backslashIndex);
  const rest = hostInput.slice(backslashIndex + 1);
  if (!rest || rest.includes("\\")) return undefined;
  const colonIndex = rest.indexOf(":");
  const instance = colonIndex === -1 ? rest : rest.slice(0, colonIndex);
  const portText = colonIndex === -1 ? undefined : rest.slice(colonIndex + 1);
  if (!instance) return undefined;
  const parsedHost = parseHostPort(
    portText !== undefined ? `${hostPart}:${portText}` : hostPart
  );
  if (!parsedHost) return undefined;
  return {
    host: `${parsedHost.host}\\${instance}`,
    port: parsedHost.port,
    username,
    password,
  };
}

function parseMssqlConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  if (!unwrapped || /^jdbc:/i.test(unwrapped)) return undefined;

  if (/^sqlserver:\/\//i.test(unwrapped)) {
    const afterScheme = unwrapped.slice("sqlserver://".length);
    const semicolonIndex = afterScheme.indexOf(";");
    const queryIndex = afterScheme.indexOf("?");
    const hashIndex = afterScheme.indexOf("#");
    // Prisma's `sqlserver://host:1433;database=mydb;user=sa;password=...`
    // embeds ADO.NET properties directly after the host instead of `?query`.
    if (
      semicolonIndex !== -1 &&
      (queryIndex === -1 || semicolonIndex < queryIndex) &&
      (hashIndex === -1 || semicolonIndex < hashIndex)
    ) {
      const hostSegment = afterScheme.slice(0, semicolonIndex);
      const kvSegment = afterScheme.slice(semicolonIndex + 1);
      const hostPort = hostSegment
        ? parseMssqlHostWithOptionalInstance(hostSegment)
        : undefined;
      if (hostSegment && !hostPort) return undefined;
      const pairs = parseAdoNetKeyValue(kvSegment);
      if (!pairs) return undefined;
      return extractMssqlFromKeyMap(pairs, hostPort);
    }

    const uri = parseStandardUri(unwrapped);
    if (!uri || !uri.host || uri.path.includes("/")) return undefined;
    const instanceName = uri.path ? decodePercent(uri.path) : "";
    let port = uri.port;
    if (!port && uri.query.has("port")) {
      port = validatePort(uri.query.get("port") ?? "");
      if (!port) return undefined;
    }
    const username =
      uri.username ??
      uri.query.get("user id") ??
      uri.query.get("userid") ??
      uri.query.get("user");
    const password = normalizePassword(
      uri.password ?? uri.query.get("password") ?? uri.query.get("pwd")
    );
    const database =
      uri.query.get("database") ?? uri.query.get("initial catalog");
    const tls = parseMssqlTls(
      uri.query.get("encrypt"),
      uri.query.get("trustservercertificate")
    );
    return {
      host: instanceName ? `${uri.host}\\${instanceName}` : uri.host,
      port: port ?? "1433",
      username: username || undefined,
      password,
      database: database || undefined,
      tls,
    };
  }

  if (SCHEME_PREFIX_PATTERN.test(unwrapped) || !unwrapped.includes(";")) {
    return undefined;
  }
  const pairs = parseAdoNetKeyValue(unwrapped);
  if (!pairs) return undefined;
  return extractMssqlFromKeyMap(pairs);
}

function parseOracleServiceSpec(spec: string): string | undefined {
  const match = spec.match(
    /^([^/:?#\s]+)(?::(?:dedicated|shared|pooled))?(?:\/[^/:?#\s]+)?$/i
  );
  return match ? match[1] : undefined;
}

function parseOracleUrlTls(query: Map<string, string>): ParsedTls | undefined {
  const ssl = query.get("ssl");
  if (ssl === undefined) return undefined;
  switch (ssl.trim().toLowerCase()) {
    case "false":
    case "disable":
    case "disabled":
    case "0":
      return { useSsl: false };
    case "true":
    case "enable":
    case "enabled":
    case "1": {
      const verifyRaw = (
        query.get("ssl verify") ?? query.get("ssl_verify")
      )?.trim();
      const skipVerify =
        verifyRaw !== undefined &&
        ["false", "disable", "disabled", "0", "no"].includes(
          verifyRaw.toLowerCase()
        );
      return { useSsl: true, verifyTlsCertificate: !skipVerify };
    }
    default:
      return undefined;
  }
}

function parseOracleConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(
    raw
      .trim()
      .replace(/^sqlplus\s+(?:-[ls]\s+)*/i, "")
      .trim()
  );
  if (!unwrapped || unwrapped.includes(";") || /^jdbc:/i.test(unwrapped)) {
    return undefined;
  }

  if (/^oracle:\/\//i.test(unwrapped)) {
    const uri = parseStandardUri(unwrapped);
    if (!uri || !uri.host) return undefined;
    const sid = uri.query.get("sid")?.trim() || undefined;
    let serviceName: string | undefined;
    if (uri.path) {
      serviceName = parseOracleServiceSpec(decodePercent(uri.path));
      if (!serviceName) return undefined;
    } else if (!sid) {
      serviceName = uri.query.get("service_name")?.trim() || undefined;
    }
    if (!sid && !serviceName) return undefined;
    return {
      host: uri.host,
      port: uri.port ?? "1521",
      username: uri.username || undefined,
      password: normalizePassword(uri.password),
      ...(sid ? { sid } : { serviceName }),
      tls: parseOracleUrlTls(uri.query),
    };
  }

  const leadingScheme = unwrapped.match(SCHEME_PREFIX_PATTERN);
  if (
    leadingScheme &&
    !["tcp", "tcps"].includes(leadingScheme[1].toLowerCase())
  ) {
    return undefined;
  }

  const withoutFragment = unwrapped.split("#", 1)[0];
  const queryIndex = withoutFragment.indexOf("?");
  const target =
    queryIndex === -1 ? withoutFragment : withoutFragment.slice(0, queryIndex);
  if (/\s/.test(target)) return undefined;
  const queryString =
    queryIndex === -1 ? "" : withoutFragment.slice(queryIndex + 1);

  let protocol: "tcp" | "tcps" | undefined;
  let username: string | undefined;
  let password: string | undefined;
  let endpoint = target;

  const parseEasyConnectUserInfo = (
    userInfo: string,
    percentEncoded: boolean
  ) => {
    const slashSep = userInfo.indexOf("/");
    const colonSep = userInfo.indexOf(":");
    const useSlash =
      slashSep !== -1 && (colonSep === -1 || slashSep < colonSep);
    const sepIndex = useSlash ? slashSep : colonSep;
    const decodePart = (part: string) =>
      percentEncoded ? decodePercent(part) : part;
    if (sepIndex === -1) {
      if (userInfo) username = decodePart(userInfo);
    } else {
      const rawUser = userInfo.slice(0, sepIndex);
      if (rawUser) username = decodePart(rawUser);
      password = decodePart(userInfo.slice(sepIndex + 1));
    }
  };

  if (leadingScheme) {
    protocol = leadingScheme[1].toLowerCase() as "tcp" | "tcps";
    endpoint = target.slice(leadingScheme[0].length);
    const atIndex = endpoint.lastIndexOf("@");
    if (atIndex !== -1) {
      parseEasyConnectUserInfo(endpoint.slice(0, atIndex), true);
      endpoint = endpoint.slice(atIndex + 1);
    }
  } else {
    const atIndex = endpoint.lastIndexOf("@");
    if (atIndex !== -1) {
      parseEasyConnectUserInfo(endpoint.slice(0, atIndex), false);
      endpoint = endpoint.slice(atIndex + 1);
    }
    const postAuthScheme = endpoint.match(SCHEME_PREFIX_PATTERN);
    if (postAuthScheme) {
      const scheme = postAuthScheme[1].toLowerCase();
      if (scheme !== "tcp" && scheme !== "tcps") return undefined;
      protocol = scheme;
      endpoint = endpoint.slice(postAuthScheme[0].length);
    } else if (endpoint.startsWith("//")) {
      endpoint = endpoint.slice(2);
    }
  }

  const slashIndex = endpoint.indexOf("/");
  const hostPortPart =
    slashIndex === -1 ? endpoint : endpoint.slice(0, slashIndex);
  const servicePart = slashIndex === -1 ? "" : endpoint.slice(slashIndex + 1);

  const parsedHostPort = parseHostPort(hostPortPart);
  if (!parsedHostPort) return undefined;

  const query = new Map<string, string>();
  for (const [key, value] of new URLSearchParams(queryString)) {
    query.set(key.toLowerCase(), value);
  }

  const sid = query.get("sid")?.trim() || undefined;
  let serviceName: string | undefined;
  if (servicePart) {
    serviceName = parseOracleServiceSpec(servicePart);
    if (!serviceName) return undefined;
  } else if (!sid) {
    serviceName = query.get("service_name")?.trim() || undefined;
  }
  if (!sid && !serviceName) return undefined;

  let tls: ParsedTls | undefined;
  if (protocol === "tcps") {
    const dnMatch = query.get("ssl_server_dn_match")?.trim().toLowerCase();
    const verify =
      dnMatch === undefined || !["no", "false", "0"].includes(dnMatch);
    tls = { useSsl: true, verifyTlsCertificate: verify };
  } else if (protocol === "tcp") {
    tls = { useSsl: false };
  } else {
    tls = parseOracleUrlTls(query);
  }

  return {
    host: parsedHostPort.host,
    port: parsedHostPort.port ?? "1521",
    username: username || undefined,
    password: normalizePassword(password),
    ...(sid ? { sid } : { serviceName }),
    tls,
  };
}

function readFlagValue(
  tokens: string[],
  index: number,
  allowStandaloneMissing = false
): { value: string | undefined; nextIndex: number } {
  const current = tokens[index];
  const eqIndex = current.indexOf("=");
  if (eqIndex !== -1) {
    return { value: current.slice(eqIndex + 1), nextIndex: index };
  }
  const next = tokens[index + 1];
  if (
    next === undefined ||
    (allowStandaloneMissing &&
      (next.startsWith("-") ||
        /^(?:mongodb(?:\+srv)?|jdbc:|hive2?:)/i.test(next)))
  ) {
    return { value: undefined, nextIndex: index };
  }
  return { value: next, nextIndex: index + 1 };
}

function parseMongoConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const trimmed = raw.trim();
  let uriInput = stripSurroundingQuotes(trimmed);
  let cliUsername: string | undefined;
  let cliPassword: string | undefined;
  let cliAuthDb: string | undefined;

  if (/^mongosh\b/i.test(trimmed)) {
    const tokens = tokenizeCliArgs(trimmed);
    if (!tokens || tokens[0]?.toLowerCase() !== "mongosh") return undefined;
    let foundUri: string | undefined;
    for (let i = 1; i < tokens.length; i++) {
      const token = tokens[i];
      const lower = token.toLowerCase();
      const flag = lower.split("=", 1)[0];
      if (flag === "--username" || flag === "-u") {
        const res = readFlagValue(tokens, i);
        cliUsername = res.value;
        i = res.nextIndex;
      } else if (flag === "--password" || flag === "-p") {
        const res = readFlagValue(tokens, i, true);
        cliPassword = res.value;
        i = res.nextIndex;
      } else if (flag === "--authenticationdatabase") {
        const res = readFlagValue(tokens, i);
        cliAuthDb = res.value;
        i = res.nextIndex;
      } else if (flag === "--apiversion") {
        const res = readFlagValue(tokens, i);
        i = res.nextIndex;
      } else if (/^mongodb(?:\+srv)?:\/\//i.test(token)) {
        foundUri = token;
      }
    }
    if (!foundUri) return undefined;
    uriInput = foundUri;
  }

  if (!/^mongodb(?:\+srv)?:\/\//i.test(uriInput)) return undefined;
  const uri = parseStandardUri(uriInput, { allowMultipleHosts: true });
  if (!uri || uri.hosts.length === 0 || uri.path.includes("/")) {
    return undefined;
  }

  const isSrv = uri.scheme === "mongodb+srv";
  if (isSrv && (uri.hosts.length !== 1 || uri.hosts[0].port !== undefined)) {
    return undefined;
  }

  const username = (cliUsername ?? uri.username)?.trim() || undefined;
  const password = normalizePassword(cliPassword ?? uri.password);
  // Standard `mongodb://` URIs default `authSource` to the URI path database,
  // whereas `mongodb+srv://` URIs let DNS TXT records (e.g. Atlas's
  // `authSource=admin`) take precedence over the path; only forward explicit
  // CLI/query `authSource` on SRV URIs.
  const authenticationDatabase =
    (
      cliAuthDb ??
      uri.query.get("authsource") ??
      (!isSrv && uri.path ? decodePercent(uri.path) : undefined)
    )?.trim() || undefined;

  // Leave `tls` unset for `mongodb+srv://` unless `?tls`/`?ssl` or an explicit
  // verification override is present, since the MongoDB driver already enables
  // verified TLS by default for SRV URIs.
  const allowInvalid =
    parseBooleanValue(uri.query.get("tlsallowinvalidcertificates")) === true ||
    parseBooleanValue(uri.query.get("tlsallowinvalidhostnames")) === true ||
    parseBooleanValue(uri.query.get("tlsinsecure")) === true ||
    parseBooleanValue(uri.query.get("sslinsecure")) === true;
  const tls =
    parseSchemeOrQueryTls(uri.scheme, uri.query, {
      skipVerify: allowInvalid,
    }) ??
    (isSrv && allowInvalid
      ? { useSsl: true, verifyTlsCertificate: false }
      : undefined);

  if (isSrv) {
    // `DataSource.useSsl = false` on SRV preserves the MongoDB driver's native
    // SRV TLS default rather than forcing plaintext, so reject explicit opt-out.
    if (tls?.useSsl === false) {
      return undefined;
    }
    return {
      host: uri.hosts[0].host,
      port: "",
      username,
      password,
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase,
      tls,
    };
  }

  const primary = uri.hosts[0];
  const additionalAddresses = uri.hosts.slice(1).map((addr) => ({
    host: addr.host,
    port: addr.port ?? "27017",
  }));
  const replicaSet = uri.query.get("replicaset")?.trim() || undefined;
  const directConnection =
    additionalAddresses.length === 0 &&
    parseBooleanValue(uri.query.get("directconnection")) === true;

  return {
    host: primary.host,
    port: primary.port ?? "27017",
    username,
    password,
    srv: false,
    additionalAddresses,
    replicaSet,
    directConnection,
    authenticationDatabase,
    tls,
  };
}

function parseRedisConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const trimmed = raw.trim();
  let uriInput = stripSurroundingQuotes(trimmed);
  let cliTls = false;
  let cliInsecure = false;

  if (/^redis-cli\b/i.test(trimmed)) {
    const tokens = tokenizeCliArgs(trimmed);
    if (!tokens || tokens[0]?.toLowerCase() !== "redis-cli") return undefined;
    let foundUri: string | undefined;
    for (let i = 1; i < tokens.length; i++) {
      const token = tokens[i];
      const lower = token.toLowerCase();
      const flag = lower.split("=", 1)[0];
      if (lower === "--tls") {
        cliTls = true;
      } else if (lower === "--insecure") {
        cliInsecure = true;
      } else if (flag === "-u" || flag === "--uri") {
        const res = readFlagValue(tokens, i);
        foundUri = res.value;
        i = res.nextIndex;
      }
    }
    if (!foundUri) return undefined;
    uriInput = foundUri;
  }

  const schemeMatch = uriInput.match(SCHEME_PREFIX_PATTERN);
  if (
    !schemeMatch ||
    !["redis", "rediss", "redis-sentinel", "rediss-sentinel"].includes(
      schemeMatch[1].toLowerCase()
    )
  ) {
    return undefined;
  }

  const uri = parseStandardUri(uriInput, { allowMultipleHosts: true });
  if (!uri || uri.hosts.length === 0 || uri.path.includes("/")) {
    return undefined;
  }

  const username =
    uri.username ??
    (uri.query.get("user") || undefined) ??
    (uri.hasExplicitEmptyUser ? "" : undefined);
  const password = normalizePassword(uri.password ?? uri.query.get("password"));
  const masterName =
    (
      uri.query.get("master") ??
      uri.query.get("master_name") ??
      uri.query.get("sentinel_master") ??
      uri.query.get("sentinelmasterid") ??
      uri.query.get("service_name")
    )?.trim() || undefined;
  const isSentinel =
    uri.scheme === "redis-sentinel" ||
    uri.scheme === "rediss-sentinel" ||
    masterName !== undefined;

  let redisType: DataSource_RedisType;
  let defaultPort: string;
  if (isSentinel) {
    redisType = DataSource_RedisType.SENTINEL;
    defaultPort = "26379";
  } else if (
    uri.hosts.length > 1 ||
    parseBooleanValue(uri.query.get("cluster")) === true
  ) {
    redisType = DataSource_RedisType.CLUSTER;
    defaultPort = "6379";
  } else {
    redisType = DataSource_RedisType.STANDALONE;
    defaultPort = "6379";
  }

  const primary = uri.hosts[0];
  const additionalAddresses = uri.hosts.slice(1).map((addr) => ({
    host: addr.host,
    port: addr.port ?? defaultPort,
  }));

  const skipVerify = parseTlsSkipVerify(
    uri.query,
    cliInsecure ||
      uri.query.get("ssl_cert_reqs")?.trim().toLowerCase() === "none"
  );
  const tls = parseSchemeOrQueryTls(uri.scheme, uri.query, {
    tlsSchemes: cliTls
      ? ["redis", "rediss", "redis-sentinel", "rediss-sentinel"]
      : ["rediss", "rediss-sentinel"],
    skipVerify,
  });

  return {
    host: primary.host,
    port: primary.port ?? defaultPort,
    username,
    password,
    redisType,
    additionalAddresses,
    masterName,
    masterUsername:
      (
        uri.query.get("master_username") ?? uri.query.get("master_user")
      )?.trim() || undefined,
    masterPassword: normalizePassword(uri.query.get("master_password")),
    tls,
  };
}

function parseClickHouseConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const uri = parseStandardUri(unwrapped);
  if (
    !uri ||
    !["clickhouse", "clickhouses", "clickhouse+native"].includes(uri.scheme) ||
    !uri.host ||
    uri.path.includes("/")
  ) {
    return undefined;
  }

  const defaultPort = uri.scheme === "clickhouses" ? "9440" : "9000";
  const skipVerify = parseTlsSkipVerify(uri.query);
  const tls = parseSchemeOrQueryTls(uri.scheme, uri.query, {
    tlsSchemes: ["clickhouses"],
    queryKeys: ["secure", "tls", "ssl"],
    skipVerify,
  });

  return {
    host: uri.host,
    port: uri.port ?? defaultPort,
    username: uri.username ?? (uri.query.get("user") || undefined),
    password: normalizePassword(uri.password ?? uri.query.get("password")),
    tls,
  };
}

function parseSnowflakeConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const normalized = /^jdbc:snowflake:\/\//i.test(unwrapped)
    ? unwrapped.slice("jdbc:".length)
    : unwrapped;
  const uri = parseStandardUri(normalized);
  if (!uri || !uri.host) return undefined;
  if (uri.scheme === "https") {
    if (!/\.snowflakecomputing\.com$/i.test(uri.host)) return undefined;
  } else if (uri.scheme !== "snowflake") {
    return undefined;
  }
  if (/^snowflakecomputing\.com$/i.test(uri.host)) return undefined;

  const host = uri.host.replace(/\.snowflakecomputing\.com$/i, "");
  if (!host) return undefined;

  return {
    host,
    port: "",
    username: uri.username || uri.query.get("user") || undefined,
    password: normalizePassword(uri.password ?? uri.query.get("password")),
  };
}

function parseElasticsearchConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const uri = parseStandardUri(unwrapped);
  if (
    !uri ||
    !["https", "http", "elasticsearch", "opensearch"].includes(uri.scheme) ||
    !uri.host ||
    uri.path !== "" ||
    /^(?:.+\.)?snowflakecomputing\.com$/i.test(uri.host)
  ) {
    return undefined;
  }

  const defaultPort =
    uri.scheme === "https" ? "443" : uri.scheme === "http" ? "80" : "9200";
  const skipVerify = parseTlsSkipVerify(uri.query);
  const tls = parseSchemeOrQueryTls(uri.scheme, uri.query, {
    tlsSchemes: ["https"],
    plainSchemes: ["http"],
    queryKeys: ["ssl", "tls"],
    skipVerify,
  });

  return {
    host: uri.host,
    port: uri.port ?? defaultPort,
    username: uri.username || undefined,
    password: normalizePassword(uri.password),
    tls,
  };
}

function parseCassandraConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const uri = parseStandardUri(unwrapped, { allowMultipleHosts: true });
  if (
    !uri ||
    uri.scheme !== "cassandra" ||
    uri.hosts.length === 0 ||
    uri.path.includes("/")
  ) {
    return undefined;
  }

  const primary = uri.hosts[0];
  const additionalAddresses = uri.hosts.slice(1).map((addr) => ({
    host: addr.host,
    port: addr.port ?? "9042",
  }));
  const skipVerify = parseTlsSkipVerify(uri.query);
  const tls = parseSchemeOrQueryTls(uri.scheme, uri.query, {
    queryKeys: ["ssl", "tls"],
    skipVerify,
  });

  return {
    host: primary.host,
    port: primary.port ?? "9042",
    username: uri.username ?? (uri.query.get("user") || undefined),
    password: normalizePassword(uri.password ?? uri.query.get("password")),
    additionalAddresses,
    tls,
  };
}

function parseTrinoConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const uri = parseStandardUri(unwrapped);
  if (
    !uri ||
    !["trino", "https", "http"].includes(uri.scheme) ||
    !uri.host ||
    /^(?:.+\.)?snowflakecomputing\.com$/i.test(uri.host)
  ) {
    return undefined;
  }

  const trimmedPath = uri.path.replace(/\/+$/, "");
  const segments = trimmedPath ? trimmedPath.split("/") : [];
  if (segments.length > 2 || segments.some((segment) => !segment)) {
    return undefined;
  }

  const defaultPort =
    uri.scheme === "https" ? "443" : uri.scheme === "http" ? "80" : "8080";
  const sslVerification = uri.query
    .get("sslverification")
    ?.trim()
    .toLowerCase();
  const skipVerify = parseTlsSkipVerify(uri.query, sslVerification === "none");
  let tls = parseSchemeOrQueryTls(uri.scheme, uri.query, {
    tlsSchemes: ["https"],
    plainSchemes: ["http"],
    queryKeys: ["ssl", "tls"],
    skipVerify,
  });
  if (!tls && uri.scheme === "trino") {
    if (sslVerification === "none") {
      tls = { useSsl: true, verifyTlsCertificate: false };
    } else if (sslVerification === "ca" || sslVerification === "full") {
      tls = { useSsl: true, verifyTlsCertificate: true };
    }
  }

  return {
    host: uri.host,
    port: uri.port ?? defaultPort,
    username: uri.username ?? (uri.query.get("user") || undefined),
    password: normalizePassword(uri.password ?? uri.query.get("password")),
    tls,
  };
}

function parseHiveConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const trimmed = raw.trim();
  let uriInput = stripSurroundingQuotes(trimmed);
  let cliUser: string | undefined;
  let cliPassword: string | undefined;

  if (/^beeline\b/i.test(trimmed)) {
    const tokens = tokenizeCliArgs(trimmed);
    if (!tokens || tokens[0]?.toLowerCase() !== "beeline") return undefined;
    let foundUri: string | undefined;
    for (let i = 1; i < tokens.length; i++) {
      const flag = tokens[i].toLowerCase().split("=", 1)[0];
      if (flag === "-u" || flag === "--url") {
        const res = readFlagValue(tokens, i);
        foundUri = res.value;
        i = res.nextIndex;
      } else if (flag === "-n" || flag === "--username") {
        const res = readFlagValue(tokens, i);
        cliUser = res.value;
        i = res.nextIndex;
      } else if (flag === "-p" || flag === "--password") {
        const res = readFlagValue(tokens, i, true);
        cliPassword = res.value;
        i = res.nextIndex;
      }
    }
    if (!foundUri) return undefined;
    uriInput = foundUri;
  }

  const normalized = /^jdbc:hive2:\/\//i.test(uriInput)
    ? uriInput.slice("jdbc:".length)
    : uriInput;
  const schemeMatch = normalized.match(SCHEME_PREFIX_PATTERN);
  if (
    !schemeMatch ||
    !["hive2", "hive"].includes(schemeMatch[1].toLowerCase())
  ) {
    return undefined;
  }

  const parsed = parseUriWithJdbcParams(normalized);
  if (!parsed) return undefined;
  const { uri, params } = parsed;
  if (!uri.host || uri.path.includes("/")) return undefined;

  const getHiveParam = (...keys: string[]): string | undefined => {
    for (const key of keys) {
      const value = params.get(key)?.trim();
      if (value) return value;
    }
    for (const key of keys) {
      const value = uri.query.get(key)?.trim();
      if (value) return value;
    }
    return undefined;
  };

  if (getHiveParam("servicediscoverymode")?.toLowerCase() === "zookeeper") {
    return undefined;
  }

  // The Hive instance form exposes neither TLS (`instanceV1HasSSL`) nor extra
  // connection parameters (`instanceV1HasExtraParameters`), and URL import
  // configures password auth (gohive's default binary SASL PLAIN transport).
  if (parseBooleanValue(getHiveParam("ssl", "tls")) === true) {
    return undefined;
  }

  const transportMode = getHiveParam(
    "transportmode",
    "transport_mode",
    "transport"
  )?.toLowerCase();
  if (transportMode && transportMode !== "binary") {
    return undefined;
  }
  if (getHiveParam("httppath", "http_path")) {
    return undefined;
  }

  if (getHiveParam("principal")) {
    return undefined;
  }
  const auth = getHiveParam("auth")?.toLowerCase();
  if (auth && !["none", "plain", "ldap", "custom"].includes(auth)) {
    return undefined;
  }

  const username =
    (
      cliUser ||
      uri.username ||
      params.get("user") ||
      params.get("uid") ||
      uri.query.get("user") ||
      uri.query.get("uid")
    )?.trim() || undefined;
  const rawPassword =
    cliPassword ??
    uri.password ??
    params.get("password") ??
    params.get("pwd") ??
    uri.query.get("password") ??
    uri.query.get("pwd");

  return {
    host: uri.host,
    port: uri.port ?? "10000",
    username,
    password: normalizePassword(rawPassword),
  };
}

function extractDatabricksWarehouseId(
  httpPath: string | undefined
): string | undefined {
  if (!httpPath) return undefined;
  const match = httpPath
    .trim()
    .match(/^\/?sql\/(?:1\.0\/)?warehouses\/([^/?;#\s]+)\/?$/i);
  return match?.[1];
}

function parseDatabricksConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  const normalized = /^jdbc:databricks:\/\//i.test(unwrapped)
    ? unwrapped.slice("jdbc:".length)
    : unwrapped;
  const parsed = parseUriWithJdbcParams(normalized);
  if (!parsed) return undefined;
  const { uri, params } = parsed;
  if (!uri.host) return undefined;
  if (uri.scheme !== "databricks" && uri.scheme !== "https") {
    return undefined;
  }

  const pathWarehouseId = extractDatabricksWarehouseId(uri.path);
  const httpPath =
    params.get("httppath") ??
    uri.query.get("http_path") ??
    uri.query.get("httppath");
  const warehouseId =
    pathWarehouseId ??
    extractDatabricksWarehouseId(httpPath) ??
    (
      params.get("warehouseid") ??
      uri.query.get("warehouse_id") ??
      uri.query.get("warehouseid")
    )?.trim() ??
    undefined;

  if (uri.scheme === "https" && !pathWarehouseId) {
    return undefined;
  }

  const rawToken =
    uri.password ??
    (uri.scheme === "databricks" &&
    uri.username &&
    uri.username.toLowerCase() !== "token"
      ? uri.username
      : undefined) ??
    params.get("pwd") ??
    params.get("password") ??
    params.get("token") ??
    params.get("access_token") ??
    uri.query.get("pwd") ??
    uri.query.get("password") ??
    uri.query.get("token") ??
    uri.query.get("access_token");
  const normalizedToken = normalizePassword(rawToken);
  const token = normalizedToken || undefined;

  const formattedHost = uri.host.includes(":") ? `[${uri.host}]` : uri.host;
  const portSuffix = uri.port && uri.port !== "443" ? `:${uri.port}` : "";

  return {
    host: `https://${formattedHost}${portSuffix}`,
    port: "",
    warehouseId: warehouseId || undefined,
    token,
  };
}

export function parseConnectionString(
  engine: Engine,
  value: string,
  vendor: Vendor = Vendor.VENDOR_UNSPECIFIED
): ParsedConnectionString | undefined {
  switch (engine) {
    case Engine.MYSQL:
      return parseMysqlFamilyConnectionString(value, {
        schemes: ["mysql"],
        defaultPort: "3306",
      });
    case Engine.MARIADB:
      return parseMysqlFamilyConnectionString(value, {
        schemes: ["mariadb", "mysql"],
        defaultPort: "3306",
      });
    case Engine.TIDB:
      return parseMysqlFamilyConnectionString(value, {
        schemes: ["mysql", "tidb"],
        defaultPort: "4000",
      });
    case Engine.OCEANBASE:
      return parseMysqlFamilyConnectionString(value, {
        schemes: ["mysql", "oceanbase"],
        defaultPort: "2883",
      });
    case Engine.STARROCKS:
      return parseMysqlFamilyConnectionString(value, {
        schemes: ["starrocks", "mysql"],
        defaultPort: "9030",
      });
    case Engine.DORIS:
      return parseMysqlFamilyConnectionString(value, {
        schemes: ["doris", "mysql"],
        defaultPort: "9030",
      });
    case Engine.POSTGRES: {
      const isNeon = vendor === Vendor.NEON;
      return parsePostgresFamilyConnectionString(value, {
        schemes: ["postgres", "postgresql"],
        defaultPort: "5432",
        includeTls: true,
        // Neon requires TLS and its console copies `sslmode=require` against
        // publicly trusted endpoints, so keep certificate verification enabled
        // unless `sslmode=no-verify` explicitly opts out.
        requireTls: isNeon,
        verifiedTlsByDefault: isNeon,
      });
    }
    case Engine.COCKROACHDB:
      return parsePostgresFamilyConnectionString(value, {
        schemes: ["postgres", "postgresql", "cockroachdb"],
        defaultPort: "26257",
        includeTls: true,
        allowCockroachCli: true,
      });
    case Engine.REDSHIFT:
      return parsePostgresFamilyConnectionString(value, {
        schemes: ["redshift", "postgres", "postgresql"],
        defaultPort: "5439",
        includeTls: false,
        allowJdbc: true,
      });
    case Engine.MSSQL:
      return parseMssqlConnectionString(value);
    case Engine.ORACLE:
      return parseOracleConnectionString(value);
    case Engine.MONGODB:
      return parseMongoConnectionString(value);
    case Engine.REDIS:
      return parseRedisConnectionString(value);
    case Engine.CASSANDRA:
      return parseCassandraConnectionString(value);
    case Engine.ELASTICSEARCH:
      return parseElasticsearchConnectionString(value);
    case Engine.CLICKHOUSE:
      return parseClickHouseConnectionString(value);
    case Engine.SNOWFLAKE:
      return parseSnowflakeConnectionString(value);
    case Engine.TRINO:
      return parseTrinoConnectionString(value);
    case Engine.HIVE:
      return parseHiveConnectionString(value);
    case Engine.DATABRICKS:
      return parseDatabricksConnectionString(value);
    default:
      return undefined;
  }
}

const ENGINE_DETECTORS: ReadonlyArray<{
  engine: ConnectionStringEngine;
  matches: (unwrapped: string) => boolean;
}> = [
  { engine: Engine.MARIADB, matches: (s) => /^mariadb:\/\//i.test(s) },
  { engine: Engine.TIDB, matches: (s) => /^tidb:\/\//i.test(s) },
  { engine: Engine.OCEANBASE, matches: (s) => /^oceanbase:\/\//i.test(s) },
  { engine: Engine.STARROCKS, matches: (s) => /^starrocks:\/\//i.test(s) },
  { engine: Engine.DORIS, matches: (s) => /^doris:\/\//i.test(s) },
  { engine: Engine.MYSQL, matches: (s) => /^mysql:\/\//i.test(s) },
  {
    engine: Engine.COCKROACHDB,
    matches: (s) =>
      /^cockroachdb:\/\//i.test(s) || /^cockroach\s+sql\b/i.test(s),
  },
  {
    engine: Engine.REDSHIFT,
    matches: (s) => /^(?:jdbc:)?redshift:\/\//i.test(s),
  },
  {
    engine: Engine.POSTGRES,
    matches: (s) =>
      /^(?:postgres(?:ql)?:\/\/|psql\s+)/i.test(s) ||
      parseLibpqKeyValue(s) !== undefined,
  },
  {
    engine: Engine.MONGODB,
    matches: (s) => /^mongodb(?:\+srv)?:\/\//i.test(s) || /^mongosh\b/i.test(s),
  },
  // Check Redis before Oracle so `redis-sentinel://...?service_name=` never
  // triggers Oracle's query-key heuristic.
  {
    engine: Engine.REDIS,
    matches: (s) =>
      /^rediss?(?:-sentinel)?:\/\//i.test(s) || /^redis-cli\b/i.test(s),
  },
  {
    engine: Engine.CLICKHOUSE,
    matches: (s) => /^clickhouse(?:s|\+native)?:\/\//i.test(s),
  },
  {
    engine: Engine.SNOWFLAKE,
    matches: (s) =>
      /^(?:jdbc:)?snowflake:\/\//i.test(s) ||
      /^https:\/\/[^/?#]+\.snowflakecomputing\.com(?:[/?#]|$)/i.test(s),
  },
  {
    engine: Engine.ELASTICSEARCH,
    matches: (s) => /^(?:elasticsearch|opensearch):\/\//i.test(s),
  },
  { engine: Engine.CASSANDRA, matches: (s) => /^cassandra:\/\//i.test(s) },
  { engine: Engine.TRINO, matches: (s) => /^trino:\/\//i.test(s) },
  {
    engine: Engine.HIVE,
    matches: (s) =>
      /^(?:jdbc:hive2|hive2?):\/\//i.test(s) || /^beeline\b/i.test(s),
  },
  {
    engine: Engine.DATABRICKS,
    matches: (s) =>
      /^(?:jdbc:)?databricks:\/\//i.test(s) ||
      /^https:\/\/[^/?#]+\/sql\/(?:1\.0\/)?warehouses\//i.test(s),
  },
  {
    engine: Engine.MSSQL,
    matches: (s) =>
      /^sqlserver:\/\//i.test(s) || (!/^jdbc:/i.test(s) && s.includes(";")),
  },
  // Oracle requires an explicit Oracle scheme, prefix, or query key here so a
  // schemeless `host:port/db` typo isn't misclassified as Easy Connect.
  {
    engine: Engine.ORACLE,
    matches: (s) =>
      /^(?:oracle|tcps?):\/\//i.test(s) ||
      /^sqlplus\s+/i.test(s) ||
      /[?&](?:sid|service_name)=/i.test(s),
  },
];

export function detectConnectionStringEngine(
  value: string,
  excludeEngine?: Engine
): ConnectionStringEngine | undefined {
  const unwrapped = stripSurroundingQuotes(value.trim());
  if (!unwrapped) return undefined;

  for (const detector of ENGINE_DETECTORS) {
    if (detector.engine === excludeEngine) continue;
    if (
      detector.matches(unwrapped) &&
      parseConnectionString(detector.engine, value)
    ) {
      return detector.engine;
    }
  }

  return undefined;
}

function clearPasswordField(ds: EditDataSource): EditDataSource {
  return {
    ...ds,
    password: "",
    updatedPassword: "",
    useEmptyPassword: false,
    updatedSecretFields: (ds.updatedSecretFields ?? []).filter(
      (field) => field !== "password"
    ),
  };
}

function clearTokenField(ds: EditDataSource): EditDataSource {
  return {
    ...ds,
    authenticationPrivateKey: "",
    updatedToken: "",
    updatedSecretFields: (ds.updatedSecretFields ?? []).filter(
      (field) => field !== "authenticationPrivateKey"
    ),
  };
}

function clearSnowflakeKeyPairFields(ds: EditDataSource): EditDataSource {
  return {
    ...ds,
    authenticationPrivateKey: "",
    updatedToken: "",
    authenticationPrivateKeyPassphrase: "",
    updatedSecretFields: (ds.updatedSecretFields ?? []).filter(
      (field) =>
        field !== "authenticationPrivateKey" &&
        field !== "authenticationPrivateKeyPassphrase"
    ),
  };
}

function clearMasterPasswordField(ds: EditDataSource): EditDataSource {
  return {
    ...ds,
    masterPassword: "",
    updatedMasterPassword: "",
    useEmptyMasterPassword: false,
    updatedSecretFields: (ds.updatedSecretFields ?? []).filter(
      (field) => field !== "masterPassword"
    ),
  };
}

export function applyParsedConnectionString(
  ds: EditDataSource,
  engine: Engine,
  parsed: ParsedConnectionString,
  vendor: Vendor = Vendor.VENDOR_UNSPECIFIED
): EditDataSource {
  const requireTls = engine === Engine.POSTGRES && vendor === Vendor.NEON;
  let next: EditDataSource = {
    ...ds,
    authenticationType: DataSource_AuthenticationType.PASSWORD,
    externalSecret: undefined,
    saslConfig: undefined,
    host: parsed.host,
    port:
      engine === Engine.MONGODB && parsed.srv
        ? ""
        : (parsed.port ?? defaultPortForEngine(engine)),
  };

  if (engine === Engine.DATABRICKS) {
    next.warehouseId = parsed.warehouseId ?? "";
    next = clearPasswordField(next);
    next =
      parsed.token !== undefined
        ? updateDataSourceSecret(next, "authenticationPrivateKey", parsed.token)
        : clearTokenField(next);
  } else {
    if (parsed.username !== undefined) {
      next.username = parsed.username;
    }
    next =
      parsed.password !== undefined
        ? updateDataSourceSecret(next, "password", parsed.password)
        : clearPasswordField(next);
  }

  next.database = hasConnectionDatabase(engine, DataSourceType.ADMIN)
    ? (parsed.database ?? "")
    : "";

  if (engine === Engine.ORACLE) {
    next.sid = parsed.sid ?? "";
    next.serviceName = parsed.serviceName ?? "";
  }

  if (engine === Engine.MONGODB) {
    next.srv = parsed.srv ?? false;
    next.additionalAddresses = (parsed.additionalAddresses ?? []).map((addr) =>
      create(DataSource_AddressSchema, addr)
    );
    next.replicaSet = parsed.replicaSet ?? "";
    next.directConnection = parsed.directConnection ?? false;
    next.authenticationDatabase = parsed.authenticationDatabase ?? "";
  }

  if (engine === Engine.REDIS) {
    next.redisType = parsed.redisType ?? DataSource_RedisType.STANDALONE;
    next.additionalAddresses = (parsed.additionalAddresses ?? []).map((addr) =>
      create(DataSource_AddressSchema, addr)
    );
    next.masterName = parsed.masterName ?? "";
    next.masterUsername = parsed.masterUsername ?? "";
    next =
      parsed.masterPassword !== undefined
        ? updateDataSourceSecret(next, "masterPassword", parsed.masterPassword)
        : clearMasterPasswordField(next);
  }

  if (engine === Engine.CASSANDRA) {
    next.additionalAddresses = (parsed.additionalAddresses ?? []).map((addr) =>
      create(DataSource_AddressSchema, addr)
    );
  }

  if (engine === Engine.SNOWFLAKE) {
    next = clearSnowflakeKeyPairFields(next);
  }

  if (parsed.tls !== undefined) {
    if (!parsed.tls.useSsl && !requireTls) {
      next = { ...next, ...disableLocalTls(next), updateSsl: undefined };
    } else if (parsed.tls.useSsl || requireTls) {
      const withTls = next.useSsl
        ? next
        : applyLocalTlsPosture(next, LOCAL_TLS_POSTURE_TLS);
      next = {
        ...next,
        ...withTls,
        useSsl: true,
        verifyTlsCertificate: parsed.tls.useSsl
          ? parsed.tls.verifyTlsCertificate
          : true,
        updateSsl: undefined,
      };
    }
  } else if (requireTls && !next.useSsl) {
    next = {
      ...next,
      ...applyLocalTlsPosture(next, LOCAL_TLS_POSTURE_TLS),
      useSsl: true,
      verifyTlsCertificate: true,
      updateSsl: undefined,
    };
  }

  return next;
}
