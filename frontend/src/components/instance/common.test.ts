import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSourceSchema,
  KerberosConfigSchema,
  SASLConfigSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { movesKeytabToNewDestination } from "./common";
import {
  applyLocalTlsCaSource,
  applyLocalTlsClientCertSource,
  applyLocalTlsPosture,
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  getLocalTlsPosture,
  isLocalTlsClientIdentitySupported,
  SSL_UPDATE_MASK_FIELDS,
} from "./tls";

describe("TLS update mask fields", () => {
  test("includes the SSL path fields alongside inline material", () => {
    expect(SSL_UPDATE_MASK_FIELDS).toEqual([
      "use_ssl",
      "ssl_ca",
      "ssl_cert",
      "ssl_key",
      "ssl_ca_path",
      "ssl_cert_path",
      "ssl_key_path",
    ]);
  });
});

describe("TLS local source helpers", () => {
  test("treats empty CA material with SSL enabled as system trust", () => {
    expect(getLocalTlsCaSource({ useSsl: true })).toBe("SYSTEM_TRUST");
  });

  test("clears only CA fields when selecting system trust", () => {
    const next = applyLocalTlsCaSource(
      {
        useSsl: true,
        sslCa: "inline-ca",
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKeyPath: "/tmp/key.pem",
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never,
      "SYSTEM_TRUST"
    );
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("inline-cert");
    expect(next.sslKeyPath).toBe("/tmp/key.pem");
    expect(next.sslCertPathSet).toBe(true);
    expect(next.sslKeyPathSet).toBe(true);
  });

  test("clears only client cert fields when selecting none", () => {
    const next = applyLocalTlsClientCertSource(
      {
        useSsl: true,
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKey: "inline-key",
        sslCertPath: "/tmp/cert.pem",
        sslKeyPath: "/tmp/key.pem",
        sslCaPathSet: true,
      } as never,
      "NONE"
    );
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.sslCaPathSet).toBe(true);
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
  });

  test("infers client certificate source from path presence flags", () => {
    expect(
      getLocalTlsClientCertSource({
        useSsl: true,
        sslCertPathSet: true,
      } as never)
    ).toBe("FILE_PATH");
  });

  test("infers client certificate source from inline presence flags", () => {
    expect(
      getLocalTlsClientCertSource({
        useSsl: true,
        sslCertSet: true,
      } as never)
    ).toBe("INLINE_PEM");
  });

  test("infers CA source from inline presence flag", () => {
    expect(
      getLocalTlsCaSource({
        useSsl: true,
        sslCaSet: true,
      } as never)
    ).toBe("INLINE_PEM");
  });
});

describe("TLS posture helpers", () => {
  test("infers disabled posture when SSL is off", () => {
    expect(getLocalTlsPosture({ useSsl: false })).toBe("DISABLED");
  });

  test("infers TLS posture when SSL is on without client identity", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCaPathSet: true,
      } as never)
    ).toBe("TLS");
  });

  test("infers mutual TLS posture from inline client material", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCertSet: true,
        sslKeySet: true,
      } as never)
    ).toBe("MUTUAL_TLS");
  });

  test("infers mutual TLS posture from file path client material", () => {
    expect(
      getLocalTlsPosture({
        useSsl: true,
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never)
    ).toBe("MUTUAL_TLS");
  });

  test("switching posture to TLS clears only client identity fields", () => {
    const next = applyLocalTlsPosture(
      {
        useSsl: true,
        sslCaPath: "/tmp/ca.pem",
        sslCaPathSet: true,
        sslCert: "inline-cert",
        sslKey: "inline-key",
        sslCertPath: "/tmp/cert.pem",
        sslKeyPath: "/tmp/key.pem",
        sslCertSet: true,
        sslKeySet: true,
        sslCertPathSet: true,
        sslKeyPathSet: true,
      } as never,
      "TLS"
    );

    expect(next.useSsl).toBe(true);
    expect(next.sslCaPath).toBe("/tmp/ca.pem");
    expect(next.sslCaPathSet).toBe(true);
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
    expect(next.sslCertPath).toBe("");
    expect(next.sslKeyPath).toBe("");
    expect(next.sslCertSet).toBe(false);
    expect(next.sslKeySet).toBe(false);
    expect(next.sslCertPathSet).toBe(false);
    expect(next.sslKeyPathSet).toBe(false);
  });

  test("switching posture to disabled clears all TLS material", () => {
    const next = applyLocalTlsPosture(
      {
        useSsl: true,
        sslCa: "inline-ca",
        sslCaPath: "/tmp/ca.pem",
        sslCert: "inline-cert",
        sslKey: "inline-key",
      } as never,
      "DISABLED"
    );

    expect(next.useSsl).toBe(false);
    expect(next.sslCa).toBe("");
    expect(next.sslCaPath).toBe("");
    expect(next.sslCert).toBe("");
    expect(next.sslKey).toBe("");
  });

  test("MSSQL does not support client identity in this form", () => {
    expect(isLocalTlsClientIdentitySupported(Engine.MSSQL)).toBe(false);
    expect(isLocalTlsClientIdentitySupported(Engine.POSTGRES)).toBe(true);
  });
});

const KEYTAB = new Uint8Array([0x05, 0x02, 0x00, 0x00]);

const kerberosDataSource = (
  overrides: MessageInitShape<typeof DataSourceSchema> = {},
  krbOverrides: MessageInitShape<typeof KerberosConfigSchema> = {}
): DataSource =>
  create(DataSourceSchema, {
    id: "admin",
    host: "hive.example.com",
    port: "10000",
    username: "bytebase",
    ...overrides,
    saslConfig: create(SASLConfigSchema, {
      mechanism: {
        case: "krbConfig",
        value: create(KerberosConfigSchema, {
          primary: "bytebase",
          realm: "EXAMPLE.COM",
          kdcHost: "kdc.example.com",
          kdcPort: "88",
          kdcTransportProtocol: "tcp",
          ...krbOverrides,
        }),
      },
    }),
  });

describe("Kerberos keytab resupply", () => {
  const stored = kerberosDataSource();

  test("an edit that keeps the destination inherits the stored keytab", () => {
    expect(
      movesKeytabToNewDestination(
        kerberosDataSource({ username: "hive" }),
        stored
      )
    ).toBe(false);
  });

  test.each([
    ["host", kerberosDataSource({ host: "hive-2.example.com" })],
    ["port", kerberosDataSource({ port: "10001" })],
    [
      "additional_addresses",
      kerberosDataSource({
        additionalAddresses: [{ host: "hive-2.example.com", port: "10000" }],
      }),
    ],
    ["ssh_host", kerberosDataSource({ sshHost: "bastion.example.com" })],
    ["ssh_port", kerberosDataSource({ sshPort: "22" })],
    [
      "extra_connection_parameters",
      kerberosDataSource({
        extraConnectionParameters: { host: "hive-2.example.com" },
      }),
    ],
    ["kdc_host", kerberosDataSource({}, { kdcHost: "kdc-2.example.com" })],
    ["kdc_port", kerberosDataSource({}, { kdcPort: "8888" })],
  ])("moving %s requires the keytab again", (_field, editing) => {
    expect(movesKeytabToNewDestination(editing, stored)).toBe(true);
  });

  test.each([
    ["realm", kerberosDataSource({}, { realm: "OTHER.EXAMPLE.COM" })],
    ["primary", kerberosDataSource({}, { primary: "hive" })],
    [
      "kdc_transport_protocol",
      kerberosDataSource({}, { kdcTransportProtocol: "udp" }),
    ],
    ["database", kerberosDataSource({ database: "warehouse" })],
  ])("changing %s leaves the destination alone", (_field, editing) => {
    expect(movesKeytabToNewDestination(editing, stored)).toBe(false);
  });

  test("a re-uploaded keytab clears the requirement", () => {
    expect(
      movesKeytabToNewDestination(
        kerberosDataSource({ host: "hive-2.example.com" }, { keytab: KEYTAB }),
        stored
      )
    ).toBe(false);
  });

  test("a data source that did not authenticate with Kerberos has no stored keytab", () => {
    const passwordDataSource = create(DataSourceSchema, {
      id: "admin",
      host: "hive.example.com",
      port: "10000",
    });
    expect(
      movesKeytabToNewDestination(
        kerberosDataSource({ host: "hive-2.example.com" }),
        passwordDataSource
      )
    ).toBe(false);
    expect(movesKeytabToNewDestination(passwordDataSource, stored)).toBe(false);
  });

  test("a data source with no stored counterpart is a create", () => {
    expect(
      movesKeytabToNewDestination(
        kerberosDataSource({ host: "hive-2.example.com" }),
        undefined
      )
    ).toBe(false);
  });
});

// Mirrors TestDataSourceDestinationClassifiesEveryField in
// backend/api/v1/instance_service_converter_test.go. dataSourceDestination is
// a projection, so a field added later is silently treated as "not a
// destination" — and if it carries an address, this copy stops seeing a move
// the server still refuses, which puts the refusal back in the connection
// dialog. This fails until the new field is named in one list or the other.
describe("Kerberos keytab resupply field classification", () => {
  const classify = (
    schema: { fields: readonly { name: string }[] },
    inDestination: string[],
    notDestination: string[]
  ) => {
    const listed = [...inDestination, ...notDestination];
    expect(new Set(listed).size).toBe(listed.length);
    const declared = schema.fields.map((field) => field.name);
    return {
      unclassified: declared.filter((name) => !listed.includes(name)),
      stale: listed.filter((name) => !declared.includes(name)),
    };
  };

  test("every DataSource field is either an address or explained", () => {
    const { unclassified, stale } = classify(
      DataSourceSchema,
      [
        "host",
        "port",
        "additional_addresses",
        "ssh_host",
        "ssh_port",
        "extra_connection_parameters",
        "sasl_config", // kdc_host / kdc_port only; classified again below
      ],
      // Out for the reasons dataSourceDestination records in Go: they name a
      // target inside a server already reached, select which cloud API serves
      // it, follow a peer the operator's own server advertises, or carry a
      // secret rather than an address.
      [
        "id",
        "type",
        "username",
        "password",
        "use_ssl",
        "verify_tls_certificate",
        "ssl_ca",
        "ssl_cert",
        "ssl_key",
        "ssl_ca_path",
        "ssl_cert_path",
        "ssl_key_path",
        "ssl_ca_set",
        "ssl_cert_set",
        "ssl_key_set",
        "ssl_ca_path_set",
        "ssl_cert_path_set",
        "ssl_key_path_set",
        "database",
        "srv",
        "authentication_database",
        "replica_set",
        "sid",
        "service_name",
        "ssh_user",
        "ssh_password",
        "ssh_private_key",
        "authentication_private_key",
        "authentication_private_key_passphrase",
        "external_secret",
        "authentication_type",
        "cloud_sql_ip_type",
        "azure_credential",
        "aws_credential",
        "gcp_credential",
        "direct_connection",
        "region",
        "warehouse_id",
        "master_name",
        "master_username",
        "master_password",
        "redis_type",
        "project_id",
        "instance_id",
      ]
    );
    expect(unclassified).toEqual([]);
    expect(stale).toEqual([]);
  });

  // sasl_config is the one field the projection picks apart rather than
  // copying whole, so the walk has to go a level down with it.
  test("every KerberosConfig field is either an address or explained", () => {
    const { unclassified, stale } = classify(
      KerberosConfigSchema,
      ["kdc_host", "kdc_port"],
      // primary, instance and realm choose the principal kinit claims rather
      // than where it sends the claim; kdc_transport_protocol reaches the same
      // kdc_host:kdc_port either way; keytab is the credential itself.
      ["primary", "instance", "realm", "keytab", "kdc_transport_protocol"]
    );
    expect(unclassified).toEqual([]);
    expect(stale).toEqual([]);
  });

  // krbConfigOf reads one mechanism, so a second one would carry its own
  // endpoint past both walks above: sasl_config is already classified in.
  test("Kerberos is still the only SASL mechanism", () => {
    expect(SASLConfigSchema.oneofs.map((oneof) => oneof.name)).toEqual([
      "mechanism",
    ]);
    expect(
      SASLConfigSchema.oneofs[0].fields.map((field) => field.name)
    ).toEqual(["krb_config"]);
  });
});
