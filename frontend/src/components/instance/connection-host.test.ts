import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSource_AuthenticationType } from "@/types/proto-es/v1/instance_service_pb";
import { isPrivateOrLocalHost } from "./connection-host";

// The same rows as TestIsPrivateOrLocalHost in
// backend/api/v1/instance_service_test.go.
const cases: [
  name: string,
  dataSource: Parameters<typeof isPrivateOrLocalHost>[1],
  want: boolean,
  engine?: Engine,
][] = [
  ["10/8", { host: "10.0.0.5" }, true],
  ["172.16/12 lower bound", { host: "172.16.0.1" }, true],
  ["172.16/12 upper bound", { host: "172.31.255.254" }, true],
  ["just outside 172.16/12", { host: "172.32.0.1" }, false],
  ["192.168/16", { host: "192.168.1.10" }, true],
  ["loopback", { host: "127.1.2.3" }, true],
  ["unspecified", { host: "0.0.0.0" }, true],
  ["link-local", { host: "169.254.169.254" }, true],
  ["carrier-grade NAT", { host: "100.64.0.7" }, true],
  ["just outside carrier-grade NAT", { host: "100.128.0.7" }, false],
  ["IPv6 loopback", { host: "::1" }, true],
  ["bracketed IPv6 loopback", { host: "[::1]" }, true],
  ["IPv6 unique local", { host: "fd12:3456:789a::1" }, true],
  ["IPv6 link-local", { host: "fe80::1" }, true],
  ["IPv4-mapped private", { host: "::ffff:10.0.0.5" }, true],
  ["public IPv4", { host: "203.0.113.5" }, false],
  ["public IPv6", { host: "2001:db8::5" }, false],
  ["private IP with a port", { host: "10.0.0.5:5432" }, true],
  [
    "private IP in an endpoint URL",
    { host: "https://10.0.0.5:9200" },
    true,
    Engine.ELASTICSEARCH,
  ],
  ["surrounding spaces and case", { host: "  LocalHost " }, true],

  ["localhost", { host: "localhost" }, true],
  ["Docker host alias", { host: "host.docker.internal" }, true],
  ["single-label name", { host: "postgres" }, true],
  ["reserved .internal suffix", { host: "ip-10-0-0-5.ec2.internal" }, true],
  ["reserved .local suffix", { host: "db.local" }, true],
  ["trailing dot", { host: "db.local." }, true],
  ["public hostname", { host: "db.example.com" }, false],
  [
    "public hostname that only contains a reserved word",
    { host: "internal.example.com" },
    false,
  ],
  ["socket path", { host: "/var/run/postgresql" }, false],
  ["empty host", { host: "" }, false],

  [
    "SSH tunnel reaches a private IP",
    { host: "10.0.0.5", sshHost: "bastion.example.com" },
    false,
  ],
  [
    "SSH tunnel reaches a local name",
    { host: "postgres", sshHost: "bastion.example.com" },
    false,
  ],

  ["Snowflake account locator", { host: "xy12345" }, false, Engine.SNOWFLAKE],
  [
    "Cloud SQL connection name",
    {
      host: "project:region:instance",
      authenticationType: DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
    },
    false,
  ],
  ["BigQuery without an endpoint", { host: "" }, false, Engine.BIGQUERY],
  [
    "DynamoDB local endpoint IP",
    { host: "http://127.0.0.1:8000" },
    true,
    Engine.DYNAMODB,
  ],
];

describe("isPrivateOrLocalHost", () => {
  test.each(cases)("%s", (_name, dataSource, want, engine) => {
    expect(isPrivateOrLocalHost(engine ?? Engine.POSTGRES, dataSource)).toBe(
      want
    );
  });
});
