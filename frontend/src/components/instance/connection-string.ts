import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSource_AuthenticationType } from "@/types/proto-es/v1/instance_service_pb";
import { type EditDataSource, updateDataSourceSecret } from "./common";
import { defaultPortForEngine } from "./constants";
import {
  applyLocalTlsPosture,
  disableLocalTls,
  LOCAL_TLS_POSTURE_TLS,
} from "./tls";

export type ConnectionStringEngine =
  | Engine.MYSQL
  | Engine.POSTGRES
  | Engine.MSSQL
  | Engine.ORACLE;

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
};

// Cloud consoles copy template tokens in place of real secrets (e.g. Azure
// `{your_password}`, Supabase `[YOUR-PASSWORD]`, or masked bullets). Treating
// them as omitted avoids submitting literal placeholder text as a password.
const PASSWORD_PLACEHOLDER_PATTERN =
  /^(?:\{your[_ -]?password(?:[_ -]?here)?\}|\[your[_ -]?password\]|<(?:your[_ -]?)?password>|\u2022+)$/i;

const SCHEME_PREFIX_PATTERN = /^([a-zA-Z][a-zA-Z0-9+.-]*):\/\//;

export function supportsConnectionString(
  engine: Engine
): engine is ConnectionStringEngine {
  return (
    engine === Engine.MYSQL ||
    engine === Engine.POSTGRES ||
    engine === Engine.MSSQL ||
    engine === Engine.ORACLE
  );
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
  password?: string;
  host?: string;
  port?: string;
  path: string;
  query: Map<string, string>;
};

// Custom database schemes (`postgresql:`, `mysql:`, `sqlserver:`, `oracle:`)
// are non-special in WHATWG URL, so `new URL()` treats their authority as an
// opaque path when credentials contain `@` or `#` or when host is omitted.
function parseStandardUri(input: string): ParsedUriParts | undefined {
  const schemeMatch = input.match(SCHEME_PREFIX_PATTERN);
  if (!schemeMatch) return undefined;
  const scheme = schemeMatch[1].toLowerCase();
  const withoutFragment = input.slice(schemeMatch[0].length).split("#", 1)[0];
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
      if (rawUser) username = decodePercent(rawUser);
      password = decodePercent(userInfo.slice(colonIndex + 1));
    }
  }

  let host: string | undefined;
  let port: string | undefined;
  if (hostPortPart) {
    const parsedHostPort = parseHostPort(hostPortPart);
    if (!parsedHostPort) return undefined;
    host = parsedHostPort.host;
    port = parsedHostPort.port;
  }

  const query = new Map<string, string>();
  for (const [key, value] of new URLSearchParams(queryString)) {
    query.set(key.toLowerCase(), value);
  }

  return { scheme, username, password, host, port, path, query };
}

function parsePostgresTls(
  sslMode: string | undefined,
  ssl?: string
): ParsedTls | undefined {
  if (sslMode !== undefined) {
    switch (sslMode.trim().toLowerCase()) {
      case "disable":
        return { useSsl: false };
      case "allow":
      case "prefer":
      case "require":
      case "no-verify":
        return { useSsl: true, verifyTlsCertificate: false };
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
        return { useSsl: true, verifyTlsCertificate: false };
    }
  }
  return undefined;
}

function parseLibpqKeyValue(input: string): Map<string, string> | undefined {
  if (input.includes(";") || SCHEME_PREFIX_PATTERN.test(input)) {
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

function parsePostgresConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(
    raw
      .trim()
      .replace(/^psql\s+/, "")
      .trim()
  );
  if (!unwrapped) return undefined;

  if (/^postgres(?:ql)?:\/\//i.test(unwrapped)) {
    const uri = parseStandardUri(unwrapped);
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
    const username = uri.username ?? uri.query.get("user");
    const password = normalizePassword(
      uri.password ?? uri.query.get("password")
    );
    const tls = parsePostgresTls(
      uri.query.get("sslmode"),
      uri.query.get("ssl")
    );
    return {
      host,
      port: port ?? "5432",
      username: username || undefined,
      password,
      database: database || undefined,
      tls,
    };
  }

  const kv = parseLibpqKeyValue(unwrapped);
  if (!kv) return undefined;
  const rawHost = kv.get("host") || kv.get("hostaddr");
  if (!rawHost || /\s/.test(rawHost)) return undefined;
  const host =
    rawHost.startsWith("[") && rawHost.endsWith("]")
      ? rawHost.slice(1, -1)
      : rawHost;
  let port = "5432";
  if (kv.has("port") && kv.get("port") !== "") {
    const validated = validatePort(kv.get("port") ?? "");
    if (!validated) return undefined;
    port = validated;
  }
  return {
    host,
    port,
    username: kv.get("user") || undefined,
    password: normalizePassword(kv.get("password")),
    database: kv.get("dbname") || undefined,
    tls: parsePostgresTls(kv.get("sslmode"), kv.get("ssl")),
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

function parseMysqlConnectionString(
  raw: string
): ParsedConnectionString | undefined {
  const unwrapped = stripSurroundingQuotes(raw.trim());
  if (!/^mysql:\/\//i.test(unwrapped)) return undefined;
  const uri = parseStandardUri(unwrapped);
  if (!uri || !uri.host || uri.path.includes("/")) return undefined;
  return {
    host: uri.host,
    port: uri.port ?? "3306",
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
  if (!unwrapped) return undefined;

  if (/^sqlserver:\/\//i.test(unwrapped)) {
    const afterScheme = unwrapped.slice("sqlserver://".length).split("#", 1)[0];
    const semicolonIndex = afterScheme.indexOf(";");
    const queryIndex = afterScheme.indexOf("?");
    // Prisma's `sqlserver://host:1433;database=mydb;user=sa;password=...`
    // embeds ADO.NET properties directly after the host instead of `?query`.
    if (
      semicolonIndex !== -1 &&
      (queryIndex === -1 || semicolonIndex < queryIndex)
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
  if (!unwrapped || unwrapped.includes(";")) {
    return undefined;
  }
  if (/^jdbc:/i.test(unwrapped)) return undefined;

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

export function parseConnectionString(
  engine: Engine,
  value: string
): ParsedConnectionString | undefined {
  switch (engine) {
    case Engine.POSTGRES:
      return parsePostgresConnectionString(value);
    case Engine.MYSQL:
      return parseMysqlConnectionString(value);
    case Engine.MSSQL:
      return parseMssqlConnectionString(value);
    case Engine.ORACLE:
      return parseOracleConnectionString(value);
    default:
      return undefined;
  }
}

export function applyParsedConnectionString(
  ds: EditDataSource,
  engine: Engine,
  parsed: ParsedConnectionString
): EditDataSource {
  let next: EditDataSource = {
    ...ds,
    authenticationType: DataSource_AuthenticationType.PASSWORD,
    externalSecret: undefined,
    host: parsed.host,
    port: parsed.port ?? defaultPortForEngine(engine),
  };

  if (parsed.username !== undefined) {
    next.username = parsed.username;
  }

  if (parsed.password !== undefined) {
    next = updateDataSourceSecret(next, "password", parsed.password);
  } else {
    next = {
      ...next,
      password: "",
      updatedPassword: "",
      useEmptyPassword: false,
      updatedSecretFields: (next.updatedSecretFields ?? []).filter(
        (field) => field !== "password"
      ),
    };
  }

  if (engine === Engine.POSTGRES || engine === Engine.MSSQL) {
    next.database = parsed.database ?? "";
  } else {
    next.database = "";
  }

  if (engine === Engine.ORACLE) {
    next.sid = parsed.sid ?? "";
    next.serviceName = parsed.serviceName ?? "";
  }

  if (parsed.tls !== undefined) {
    if (!parsed.tls.useSsl) {
      next = { ...next, ...disableLocalTls(next), updateSsl: undefined };
    } else {
      const withTls = next.useSsl
        ? next
        : applyLocalTlsPosture(next, LOCAL_TLS_POSTURE_TLS);
      next = {
        ...next,
        ...withTls,
        useSsl: true,
        verifyTlsCertificate: parsed.tls.verifyTlsCertificate,
        updateSsl: undefined,
      };
    }
  }

  return next;
}
