import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSourceExternalSecret_SecretType,
  DataSourceExternalSecretSchema,
  DataSourceSchema,
  DataSourceType,
  Vendor,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "./common";
import {
  applyParsedConnectionString,
  detectConnectionStringEngine,
  parseConnectionString,
  supportsConnectionString,
} from "./connection-string";

const makeDraft = (
  overrides: Partial<EditDataSource> = {}
): EditDataSource => ({
  ...create(DataSourceSchema, {
    id: "admin",
    type: DataSourceType.ADMIN,
    authenticationType: DataSource_AuthenticationType.PASSWORD,
    username: "bytebase",
  }),
  pendingCreate: true,
  updatedPassword: "",
  updatedMasterPassword: "",
  updatedToken: "",
  useEmptyPassword: false,
  useEmptyMasterPassword: false,
  extraConnectionParameters: {},
  ...overrides,
});

describe("supportsConnectionString", () => {
  it("enables connection string input for MySQL, PostgreSQL, MSSQL, and Oracle only", () => {
    expect(supportsConnectionString(Engine.MYSQL)).toBe(true);
    expect(supportsConnectionString(Engine.POSTGRES)).toBe(true);
    expect(supportsConnectionString(Engine.MSSQL)).toBe(true);
    expect(supportsConnectionString(Engine.ORACLE)).toBe(true);
    expect(supportsConnectionString(Engine.MARIADB)).toBe(false);
    expect(supportsConnectionString(Engine.SNOWFLAKE)).toBe(false);
    expect(supportsConnectionString(Engine.MONGODB)).toBe(false);
  });
});

describe("parseConnectionString - PostgreSQL", () => {
  it("parses standard postgresql:// and postgres:// URIs", () => {
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "postgresql://app_user:p%40ss%3Aword@db.example.com:5433/storefront?sslmode=verify-full&pgbouncer=true"
      )
    ).toEqual({
      host: "db.example.com",
      port: "5433",
      username: "app_user",
      password: "p@ss:word",
      database: "storefront",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "psql 'postgres://postgres:[YOUR-PASSWORD]@db.example.supabase.co/postgres?sslmode=require'"
      )
    ).toEqual({
      host: "db.example.supabase.co",
      port: "5432",
      username: "postgres",
      password: undefined,
      database: "postgres",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });
  });

  it("supports IPv6 hosts and query-supplied host/dbname in PostgreSQL URIs", () => {
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "postgresql://admin:secret@[2001:db8::1]:5432/appdb?sslmode=disable"
      )
    ).toEqual({
      host: "2001:db8::1",
      port: "5432",
      username: "admin",
      password: "secret",
      database: "appdb",
      tls: { useSsl: false },
    });

    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "postgresql:///analytics?host=pg.example.com&port=6432&user=reader"
      )
    ).toEqual({
      host: "pg.example.com",
      port: "6432",
      username: "reader",
      password: undefined,
      database: "analytics",
      tls: undefined,
    });
  });

  it("parses libpq keyword=value strings with single-quoted escapes and semicolons", () => {
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "host=mydb.postgres.database.azure.com port=5432 dbname=appdb user=myadmin password='it\\'s \\\\ a;b' sslmode=require"
      )
    ).toEqual({
      host: "mydb.postgres.database.azure.com",
      port: "5432",
      username: "myadmin",
      password: "it's \\ a;b",
      database: "appdb",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });
  });

  it("rejects non-PostgreSQL connection strings and bare hostnames", () => {
    expect(
      parseConnectionString(Engine.POSTGRES, "db.example.com")
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "mysql://root:pass@db.example.com:3306/app"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "Server=tcp:db.example.com,1433;Database=app;"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "host=db.example.com;port=5432;dbname=app"
      )
    ).toBeUndefined();
  });

  it("keeps verified TLS for Neon sslmode=require, honors no-verify, and rejects plaintext TLS modes", () => {
    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "psql 'postgresql://app_owner:p%40ss@ep-example-123456.us-east-2.aws.neon.tech/storefront?sslmode=require&channel_binding=require'",
        Vendor.NEON
      )
    ).toEqual({
      host: "ep-example-123456.us-east-2.aws.neon.tech",
      port: "5432",
      username: "app_owner",
      password: "p@ss",
      database: "storefront",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "postgresql://app_owner:secret@ep-example-123456.us-east-2.aws.neon.tech/storefront?sslmode=no-verify",
        Vendor.NEON
      )
    ).toMatchObject({
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "postgresql://app_owner:secret@ep-example-123456.us-east-2.aws.neon.tech/storefront?sslmode=disable",
        Vendor.NEON
      )
    ).toBeUndefined();

    expect(
      parseConnectionString(
        Engine.POSTGRES,
        "host=ep-example-123456.us-east-2.aws.neon.tech dbname=storefront user=app_owner sslmode=disable",
        Vendor.NEON
      )
    ).toBeUndefined();

    const applied = applyParsedConnectionString(
      makeDraft({ useSsl: false, verifyTlsCertificate: false }),
      Engine.POSTGRES,
      {
        host: "ep-example-123456.us-east-2.aws.neon.tech",
        port: "5432",
        username: "app_owner",
        database: "storefront",
      },
      Vendor.NEON
    );
    expect(applied).toMatchObject({
      useSsl: true,
      verifyTlsCertificate: true,
    });
  });
});

describe("parseConnectionString - MySQL", () => {
  it("parses mysql:// URIs, maps TLS options, and ignores the database path", () => {
    expect(
      parseConnectionString(
        Engine.MYSQL,
        "mysql://admin:s3cr3t@mysql.example.com:25060/defaultdb?ssl-mode=REQUIRED"
      )
    ).toEqual({
      host: "mysql.example.com",
      port: "25060",
      username: "admin",
      password: "s3cr3t",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.MYSQL,
        'mysql://user:pass@aws.connect.psdb.cloud/mydb?ssl={"rejectUnauthorized":true}'
      )
    ).toEqual({
      host: "aws.connect.psdb.cloud",
      port: "3306",
      username: "user",
      password: "pass",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(Engine.MYSQL, "mysql://root:@127.0.0.1/localdb")
    ).toEqual({
      host: "127.0.0.1",
      port: "3306",
      username: "root",
      password: "",
      tls: undefined,
    });
  });

  it("rejects non-MySQL strings and invalid ports", () => {
    expect(
      parseConnectionString(
        Engine.MYSQL,
        "postgresql://user:pass@db.example.com/mydb"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.MYSQL,
        "mysql://user:pass@db.example.com:99999/mydb"
      )
    ).toBeUndefined();
  });
});

describe("parseConnectionString - SQL Server", () => {
  it("parses Azure SQL ADO.NET strings and strips password placeholders", () => {
    expect(
      parseConnectionString(
        Engine.MSSQL,
        "Server=tcp:myserver.database.windows.net,1433;Initial Catalog=orders;Persist Security Info=False;User ID=sqladmin;Password={your_password};MultipleActiveResultSets=False;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;"
      )
    ).toEqual({
      host: "myserver.database.windows.net",
      port: "1433",
      username: "sqladmin",
      password: undefined,
      database: "orders",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });
  });

  it("preserves named instances and brace-escaped passwords in ADO.NET strings", () => {
    expect(
      parseConnectionString(
        Engine.MSSQL,
        "Data Source=sql.example.com\\SQLEXPRESS,1434;Database=erp;Uid=sa;Pwd={p}}a;ss};Encrypt=yes;TrustServerCertificate=yes"
      )
    ).toEqual({
      host: "sql.example.com\\SQLEXPRESS",
      port: "1434",
      username: "sa",
      password: "p}a;ss",
      database: "erp",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });
  });

  it("parses RFC and semicolon-style sqlserver:// URIs", () => {
    expect(
      parseConnectionString(
        Engine.MSSQL,
        "sqlserver://sa:p%40ss@sql.example.com/SQLEXPRESS?database=appdb&encrypt=false"
      )
    ).toEqual({
      host: "sql.example.com\\SQLEXPRESS",
      port: "1433",
      username: "sa",
      password: "p@ss",
      database: "appdb",
      tls: { useSsl: false },
    });

    expect(
      parseConnectionString(
        Engine.MSSQL,
        "sqlserver://sql.example.com:1433;database=appdb;user=sa;password=secret;encrypt=true;trustServerCertificate=true"
      )
    ).toEqual({
      host: "sql.example.com",
      port: "1433",
      username: "sa",
      password: "secret",
      database: "appdb",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.MSSQL,
        "sqlserver://db.example.com:1433;user=sa;password={abc#def};database=app;encrypt=true"
      )
    ).toEqual({
      host: "db.example.com",
      port: "1433",
      username: "sa",
      password: "abc#def",
      database: "app",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.MSSQL,
        "sqlserver://db.example.com:1433;user=sa;password=abc#def;database=app;encrypt=true"
      )
    ).toEqual({
      host: "db.example.com",
      port: "1433",
      username: "sa",
      password: "abc#def",
      database: "app",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });
  });

  it("rejects non-MSSQL schemes and non-TCP transports", () => {
    expect(
      parseConnectionString(
        Engine.MSSQL,
        "postgresql://user:pass@db.example.com/mydb"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.MSSQL,
        "Server=np:\\\\.\\pipe\\sql\\query;Database=mydb;"
      )
    ).toBeUndefined();
  });
});

describe("parseConnectionString - Oracle", () => {
  it("parses Easy Connect and Easy Connect Plus strings", () => {
    expect(
      parseConnectionString(
        Engine.ORACLE,
        "tcps://adb.us-ashburn-1.oraclecloud.com:1522/g123_mydb_high.adb.oraclecloud.com?ssl_server_dn_match=yes"
      )
    ).toEqual({
      host: "adb.us-ashburn-1.oraclecloud.com",
      port: "1522",
      username: undefined,
      password: undefined,
      serviceName: "g123_mydb_high.adb.oraclecloud.com",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.ORACLE,
        "sqlplus system/manager@//ora.example.com:1521/XEPDB1:dedicated/inst1"
      )
    ).toEqual({
      host: "ora.example.com",
      port: "1521",
      username: "system",
      password: "manager",
      serviceName: "XEPDB1",
      tls: undefined,
    });
  });

  it("parses oracle:// URIs with either service name or SID", () => {
    expect(
      parseConnectionString(
        Engine.ORACLE,
        "oracle://system:p%40ss@ora.example.com:1521/ORCLPDB1?SSL=true&SSL%20Verify=false"
      )
    ).toEqual({
      host: "ora.example.com",
      port: "1521",
      username: "system",
      password: "p@ss",
      serviceName: "ORCLPDB1",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.ORACLE,
        "oracle://system:pass@ora.example.com:1521?SID=XE"
      )
    ).toEqual({
      host: "ora.example.com",
      port: "1521",
      username: "system",
      password: "pass",
      sid: "XE",
      tls: undefined,
    });
  });

  it("rejects strings missing a service name or SID and other engine schemes", () => {
    expect(
      parseConnectionString(Engine.ORACLE, "ora.example.com:1521")
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.ORACLE,
        "mysql://root:pass@ora.example.com:3306/app"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.ORACLE,
        "jdbc:oracle:thin:@//ora.example.com:1521/ORCL"
      )
    ).toBeUndefined();
  });
});

describe("applyParsedConnectionString", () => {
  it("resets IAM/external secrets, preserves default username when omitted, and clears stale passwords without marking empty", () => {
    const initial = makeDraft({
      authenticationType: DataSource_AuthenticationType.AWS_RDS_IAM,
      externalSecret: create(DataSourceExternalSecretSchema, {
        secretType: DataSourceExternalSecret_SecretType.VAULT_KV_V2,
      }),
      password: "old",
      updatedPassword: "old",
      updatedSecretFields: ["password"],
    });

    const updated = applyParsedConnectionString(initial, Engine.ORACLE, {
      host: "adb.us-ashburn-1.oraclecloud.com",
      port: "1522",
      serviceName: "mydb_high.adb.oraclecloud.com",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(updated).toMatchObject({
      authenticationType: DataSource_AuthenticationType.PASSWORD,
      externalSecret: undefined,
      host: "adb.us-ashburn-1.oraclecloud.com",
      port: "1522",
      username: "bytebase",
      password: "",
      updatedPassword: "",
      useEmptyPassword: false,
      serviceName: "mydb_high.adb.oraclecloud.com",
      sid: "",
      useSsl: true,
      verifyTlsCertificate: true,
      updateSsl: undefined,
    });
    expect(updated.updatedSecretFields).toEqual([]);
  });

  it("marks explicit empty passwords and switches Oracle between SID and service name", () => {
    const mysqlDraft = applyParsedConnectionString(makeDraft(), Engine.MYSQL, {
      host: "127.0.0.1",
      port: "3306",
      username: "root",
      password: "",
    });
    expect(mysqlDraft.useEmptyPassword).toBe(true);
    expect(mysqlDraft.database).toBe("");

    const withService = applyParsedConnectionString(
      makeDraft(),
      Engine.ORACLE,
      {
        host: "ora.example.com",
        serviceName: "ORCLPDB1",
      }
    );
    const withSid = applyParsedConnectionString(withService, Engine.ORACLE, {
      host: "ora.example.com",
      sid: "XE",
    });
    expect(withSid.sid).toBe("XE");
    expect(withSid.serviceName).toBe("");
  });
});

describe("detectConnectionStringEngine", () => {
  it("identifies unambiguous strings belonging to another supported engine", () => {
    expect(
      detectConnectionStringEngine(
        "postgresql://user:pass@pg.example.com:5432/appdb",
        Engine.MYSQL
      )
    ).toBe(Engine.POSTGRES);
    expect(
      detectConnectionStringEngine(
        "host=pg.example.com port=5432 dbname=appdb user=postgres",
        Engine.MYSQL
      )
    ).toBe(Engine.POSTGRES);
    expect(
      detectConnectionStringEngine(
        "mysql://root:pass@mysql.example.com:3306/app",
        Engine.POSTGRES
      )
    ).toBe(Engine.MYSQL);
    expect(
      detectConnectionStringEngine(
        "Server=tcp:sql.example.com,1433;Database=orders;User ID=sa;Password=secret;",
        Engine.MYSQL
      )
    ).toBe(Engine.MSSQL);
    expect(
      detectConnectionStringEngine(
        "tcps://system:secret@ora.example.com:1522/orclpdb1",
        Engine.MYSQL
      )
    ).toBe(Engine.ORACLE);
  });

  it("ignores excluded engines and schemeless host:port/db typos", () => {
    expect(
      detectConnectionStringEngine(
        "mysql://root:pass@mysql.example.com:3306/app",
        Engine.MYSQL
      )
    ).toBeUndefined();
    expect(
      detectConnectionStringEngine("db.example.com:5432/mydb", Engine.POSTGRES)
    ).toBeUndefined();
  });
});
