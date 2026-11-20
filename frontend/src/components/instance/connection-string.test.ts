import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceExternalSecret_SecretType,
  DataSourceExternalSecretSchema,
  DataSourceSchema,
  DataSourceType,
  KerberosConfigSchema,
  SASLConfigSchema,
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
  it("enables connection string input for the 20 credential/endpoint engines and excludes cloud-IAM-only engines", () => {
    const supported = [
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
    ];
    for (const engine of supported) {
      expect(supportsConnectionString(engine)).toBe(true);
    }

    const excluded = [
      Engine.SPANNER,
      Engine.BIGQUERY,
      Engine.DYNAMODB,
      Engine.COSMOSDB,
    ];
    for (const engine of excluded) {
      expect(supportsConnectionString(engine)).toBe(false);
    }
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
    expect(
      detectConnectionStringEngine(
        "https://search.example.com:9200",
        Engine.MYSQL
      )
    ).toBeUndefined();
    expect(
      detectConnectionStringEngine(
        "http://trino.example.com:8080/catalog",
        Engine.MYSQL
      )
    ).toBeUndefined();
  });

  it("identifies unambiguous strings for newly supported engines", () => {
    expect(
      detectConnectionStringEngine(
        "mariadb://admin:secret@db.example.com:3306/app",
        Engine.MYSQL
      )
    ).toBe(Engine.MARIADB);
    expect(
      detectConnectionStringEngine(
        "cockroachdb://root@roach.example.com:26257/defaultdb?sslmode=verify-full",
        Engine.POSTGRES
      )
    ).toBe(Engine.COCKROACHDB);
    expect(
      detectConnectionStringEngine(
        'cockroach sql --url "postgresql://user:secret@roach.example.com:26257/defaultdb"',
        Engine.MYSQL
      )
    ).toBe(Engine.COCKROACHDB);
    expect(
      detectConnectionStringEngine(
        "jdbc:redshift://cluster.example.com:5439/dev?user=admin&password=secret",
        Engine.POSTGRES
      )
    ).toBe(Engine.REDSHIFT);
    expect(
      detectConnectionStringEngine(
        "mongodb+srv://user:secret@cluster0.example.mongodb.net/mydb",
        Engine.MYSQL
      )
    ).toBe(Engine.MONGODB);
    expect(
      detectConnectionStringEngine(
        "redis-sentinel://sentinel.example.com:26379?service_name=mymaster",
        Engine.MYSQL
      )
    ).toBe(Engine.REDIS);
    expect(
      detectConnectionStringEngine(
        "clickhouses://default:secret@ch.example.com:9440/default",
        Engine.MYSQL
      )
    ).toBe(Engine.CLICKHOUSE);
    expect(
      detectConnectionStringEngine(
        "https://xy12345.us-east-1.snowflakecomputing.com",
        Engine.MYSQL
      )
    ).toBe(Engine.SNOWFLAKE);
    expect(
      detectConnectionStringEngine(
        "opensearch://admin:secret@search.example.com:9200",
        Engine.MYSQL
      )
    ).toBe(Engine.ELASTICSEARCH);
    expect(
      detectConnectionStringEngine(
        "cassandra://cass:secret@n1.example.com:9042,n2.example.com:9042",
        Engine.MYSQL
      )
    ).toBe(Engine.CASSANDRA);
    expect(
      detectConnectionStringEngine(
        "trino://user@trino.example.com:8080/hive/default",
        Engine.MYSQL
      )
    ).toBe(Engine.TRINO);
    expect(
      detectConnectionStringEngine(
        "jdbc:hive2://hive.example.com:10000/default;user=hive;password=secret",
        Engine.MSSQL
      )
    ).toBe(Engine.HIVE);
    expect(
      detectConnectionStringEngine(
        "https://dbc-12345.cloud.databricks.com/sql/1.0/warehouses/wh123",
        Engine.MYSQL
      )
    ).toBe(Engine.DATABRICKS);
    expect(
      detectConnectionStringEngine(
        "jdbc:databricks://dbc.example.com:443/default;httpPath=/sql/1.0/warehouses/wh1;PWD=dapi123",
        Engine.POSTGRES
      )
    ).toBe(Engine.DATABRICKS);
  });
});

describe("parseConnectionString - MySQL-family engines", () => {
  it("parses MariaDB, TiDB, OceanBase, StarRocks, and Doris URLs with engine default ports", () => {
    expect(
      parseConnectionString(
        Engine.MARIADB,
        "mariadb://admin:s3cr3t@maria.example.com/appdb?ssl-mode=REQUIRED"
      )
    ).toEqual({
      host: "maria.example.com",
      port: "3306",
      username: "admin",
      password: "s3cr3t",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.TIDB,
        "mysql://user:pass@tidb.example.com/appdb?ssl-mode=VERIFY_IDENTITY"
      )
    ).toEqual({
      host: "tidb.example.com",
      port: "4000",
      username: "user",
      password: "pass",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.OCEANBASE,
        "mysql://root@sys#obcluster:s3cr3t@ob.example.com/test#frag"
      )
    ).toEqual({
      host: "ob.example.com",
      port: "2883",
      username: "root@sys#obcluster",
      password: "s3cr3t",
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.STARROCKS,
        "starrocks://sr:secret@sr.example.com/analytics"
      )
    ).toEqual({
      host: "sr.example.com",
      port: "9030",
      username: "sr",
      password: "secret",
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.DORIS,
        "doris://admin:pass@doris.example.com:9031/db"
      )
    ).toEqual({
      host: "doris.example.com",
      port: "9031",
      username: "admin",
      password: "pass",
      tls: undefined,
    });
  });
});

describe("parseConnectionString - CockroachDB and Redshift", () => {
  it("parses CockroachDB URIs, cockroach sql CLI, placeholder passwords, and libpq", () => {
    expect(
      parseConnectionString(
        Engine.COCKROACHDB,
        'cockroach sql --url "postgresql://app_user:<ENTER-SQL-USER-PASSWORD>@roach.example.com:26257/defaultdb?sslmode=verify-full"'
      )
    ).toEqual({
      host: "roach.example.com",
      port: "26257",
      username: "app_user",
      password: undefined,
      database: "defaultdb",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.COCKROACHDB,
        "cockroachdb://root:pass@roach.example.com/bank?sslmode=require"
      )
    ).toEqual({
      host: "roach.example.com",
      port: "26257",
      username: "root",
      password: "pass",
      database: "bank",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.COCKROACHDB,
        "host=roach.example.com dbname=bank user=root sslmode=verify-ca"
      )
    ).toEqual({
      host: "roach.example.com",
      port: "26257",
      username: "root",
      password: undefined,
      database: "bank",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });
  });

  it("parses Redshift URIs and JDBC URLs while omitting TLS controls", () => {
    expect(
      parseConnectionString(
        Engine.REDSHIFT,
        "redshift://admin:s3cr3t@rs.example.com/dev?sslmode=require"
      )
    ).toEqual({
      host: "rs.example.com",
      port: "5439",
      username: "admin",
      password: "s3cr3t",
      database: "dev",
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.REDSHIFT,
        "jdbc:redshift://rs.example.com:5439/analytics?UID=analyst&PWD=secret&ssl=true"
      )
    ).toEqual({
      host: "rs.example.com",
      port: "5439",
      username: "analyst",
      password: "secret",
      database: "analytics",
      tls: undefined,
    });
  });
});

describe("parseConnectionString - MongoDB", () => {
  it("parses mongodb+srv:// without forcing useSsl or promoting the path to authSource", () => {
    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:<db_password>@cluster0.example.mongodb.net/storefront?retryWrites=true&w=majority"
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: undefined,
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: undefined,
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:<db_password>@cluster0.example.mongodb.net/?retryWrites=true"
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: undefined,
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: undefined,
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/storefront?tls=true&tlsAllowInvalidCertificates=true&authSource=admin"
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: "pass",
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: "admin",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        'mongosh "mongodb+srv://cluster0.example.mongodb.net/storefront" --authenticationDatabase custom -u app -p pass'
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: "pass",
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: "custom",
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/?tlsInsecure=true"
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: "pass",
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: undefined,
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/?tlsAllowInvalidCertificates=true"
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: "pass",
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: undefined,
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/?tlsAllowInvalidHostnames=true"
      )
    ).toEqual({
      host: "cluster0.example.mongodb.net",
      port: "",
      username: "app",
      password: "pass",
      srv: true,
      additionalAddresses: [],
      directConnection: false,
      authenticationDatabase: undefined,
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/?tls=false&tlsInsecure=true"
      )
    ).toBeUndefined();

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/?tls=false"
      )
    ).toBeUndefined();

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://app:pass@cluster0.example.mongodb.net/?ssl=false"
      )
    ).toBeUndefined();

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb://app:pass@db.example.com/?tls=false"
      )
    ).toEqual({
      host: "db.example.com",
      port: "27017",
      username: "app",
      password: "pass",
      srv: false,
      additionalAddresses: [],
      replicaSet: undefined,
      directConnection: false,
      authenticationDatabase: undefined,
      tls: { useSsl: false },
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb://app:pass@db.example.com/?tlsInsecure=true"
      )
    ).toEqual({
      host: "db.example.com",
      port: "27017",
      username: "app",
      password: "pass",
      srv: false,
      additionalAddresses: [],
      replicaSet: undefined,
      directConnection: false,
      authenticationDatabase: undefined,
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb://app:secret@db.example.com/my%20authdb"
      )
    ).toEqual({
      host: "db.example.com",
      port: "27017",
      username: "app",
      password: "secret",
      srv: false,
      additionalAddresses: [],
      replicaSet: undefined,
      directConnection: false,
      authenticationDatabase: "my authdb",
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb://user:secret@[2001:db8::1]:27017,[2001:db8::2]:27018/storefront"
      )
    ).toEqual({
      host: "2001:db8::1",
      port: "27017",
      username: "user",
      password: "secret",
      srv: false,
      additionalAddresses: [{ host: "2001:db8::2", port: "27018" }],
      replicaSet: undefined,
      directConnection: false,
      authenticationDatabase: "storefront",
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://cluster0.example.mongodb.net:27017/db"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb+srv://n1.example.net,n2.example.net/db"
      )
    ).toBeUndefined();
  });

  it("parses multi-host mongodb:// URIs and mongosh commands", () => {
    expect(
      parseConnectionString(
        Engine.MONGODB,
        "mongodb://admin:pass@n1.example.com,n2.example.com:27018/appdb?replicaSet=rs0&authSource=admin&tls=true"
      )
    ).toEqual({
      host: "n1.example.com",
      port: "27017",
      username: "admin",
      password: "pass",
      srv: false,
      additionalAddresses: [{ host: "n2.example.com", port: "27018" }],
      replicaSet: "rs0",
      directConnection: false,
      authenticationDatabase: "admin",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.MONGODB,
        'mongosh "mongodb://localhost:27017/mydb?directConnection=true" --apiVersion 1 --username admin --password secret --authenticationDatabase admin'
      )
    ).toEqual({
      host: "localhost",
      port: "27017",
      username: "admin",
      password: "secret",
      srv: false,
      additionalAddresses: [],
      replicaSet: undefined,
      directConnection: true,
      authenticationDatabase: "admin",
      tls: undefined,
    });
  });
});

describe("parseConnectionString - Redis", () => {
  it("parses standalone, password-only, redis-cli, cluster, and sentinel strings", () => {
    expect(
      parseConnectionString(Engine.REDIS, "redis://:s3cr3t@redis.example.com")
    ).toEqual({
      host: "redis.example.com",
      port: "6379",
      username: "",
      password: "s3cr3t",
      redisType: DataSource_RedisType.STANDALONE,
      additionalAddresses: [],
      masterName: undefined,
      masterUsername: undefined,
      masterPassword: undefined,
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.REDIS,
        "redis-cli --tls -u rediss://default:pass@redis.example.com:6380/0?ssl_cert_reqs=none"
      )
    ).toEqual({
      host: "redis.example.com",
      port: "6380",
      username: "default",
      password: "pass",
      redisType: DataSource_RedisType.STANDALONE,
      additionalAddresses: [],
      masterName: undefined,
      masterUsername: undefined,
      masterPassword: undefined,
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.REDIS,
        "redis://n1.example.com,n2.example.com:6380"
      )
    ).toEqual({
      host: "n1.example.com",
      port: "6379",
      username: undefined,
      password: undefined,
      redisType: DataSource_RedisType.CLUSTER,
      additionalAddresses: [{ host: "n2.example.com", port: "6380" }],
      masterName: undefined,
      masterUsername: undefined,
      masterPassword: undefined,
      tls: undefined,
    });

    expect(
      parseConnectionString(
        Engine.REDIS,
        "rediss-sentinel://sentuser:sentpass@s1.example.com,s2.example.com:26380?master=mymaster&master_username=muser&master_password=mpass"
      )
    ).toEqual({
      host: "s1.example.com",
      port: "26379",
      username: "sentuser",
      password: "sentpass",
      redisType: DataSource_RedisType.SENTINEL,
      additionalAddresses: [{ host: "s2.example.com", port: "26380" }],
      masterName: "mymaster",
      masterUsername: "muser",
      masterPassword: "mpass",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });
  });
});

describe("parseConnectionString - ClickHouse, Snowflake, Elasticsearch, Cassandra, Trino, Hive, Databricks", () => {
  it("parses ClickHouse native schemes and rejects https://", () => {
    expect(
      parseConnectionString(
        Engine.CLICKHOUSE,
        "clickhouses://default:secret@ch.example.com/analytics"
      )
    ).toEqual({
      host: "ch.example.com",
      port: "9440",
      username: "default",
      password: "secret",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.CLICKHOUSE,
        "clickhouse+native://user:pass@ch.example.com:9001?secure=true&skip_verify=true"
      )
    ).toEqual({
      host: "ch.example.com",
      port: "9001",
      username: "user",
      password: "pass",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(Engine.CLICKHOUSE, "https://ch.example.com:8443")
    ).toBeUndefined();
  });

  it("parses Snowflake URLs, strips .snowflakecomputing.com, and rejects bare domain", () => {
    expect(
      parseConnectionString(
        Engine.SNOWFLAKE,
        "snowflake://bytebase:secret@org-account.snowflakecomputing.com/ANALYTICS/PUBLIC"
      )
    ).toEqual({
      host: "org-account",
      port: "",
      username: "bytebase",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.SNOWFLAKE,
        "snowflake://bytebase@org-account.snowflakecomputing.com"
      )
    ).toEqual({
      host: "org-account",
      port: "",
      username: "bytebase",
      password: undefined,
    });

    expect(
      parseConnectionString(
        Engine.SNOWFLAKE,
        "snowflake://:secret@org-account.snowflakecomputing.com/?user=bytebase"
      )
    ).toEqual({
      host: "org-account",
      port: "",
      username: "bytebase",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.SNOWFLAKE,
        "jdbc:snowflake://xy12345.us-east-1.snowflakecomputing.com:443/?user=bytebase&password=secret"
      )
    ).toEqual({
      host: "xy12345.us-east-1",
      port: "",
      username: "bytebase",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.SNOWFLAKE,
        "https://xy12345.us-east-1.snowflakecomputing.com/"
      )
    ).toEqual({
      host: "xy12345.us-east-1",
      port: "",
      username: undefined,
      password: undefined,
    });

    expect(
      parseConnectionString(Engine.SNOWFLAKE, "https://snowflakecomputing.com")
    ).toBeUndefined();
  });

  it("parses Elasticsearch URLs and rejects non-empty paths or Snowflake domains", () => {
    expect(
      parseConnectionString(
        Engine.ELASTICSEARCH,
        "https://elastic:secret@search.example.com/"
      )
    ).toEqual({
      host: "search.example.com",
      port: "443",
      username: "elastic",
      password: "secret",
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(Engine.ELASTICSEARCH, "http://search.example.com")
    ).toEqual({
      host: "search.example.com",
      port: "80",
      username: undefined,
      password: undefined,
      tls: { useSsl: false },
    });

    expect(
      parseConnectionString(
        Engine.ELASTICSEARCH,
        "opensearch://admin:secret@os.example.com?ssl=true&skip_verify=true"
      )
    ).toEqual({
      host: "os.example.com",
      port: "9200",
      username: "admin",
      password: "secret",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(
        Engine.ELASTICSEARCH,
        "https://search.example.com/_search"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.ELASTICSEARCH,
        "https://xy12345.snowflakecomputing.com"
      )
    ).toBeUndefined();
  });

  it("parses Cassandra multi-host URIs", () => {
    expect(
      parseConnectionString(
        Engine.CASSANDRA,
        "cassandra://cass:secret@n1.example.com,n2.example.com:9142/myks?ssl=true"
      )
    ).toEqual({
      host: "n1.example.com",
      port: "9042",
      username: "cass",
      password: "secret",
      additionalAddresses: [{ host: "n2.example.com", port: "9142" }],
      tls: { useSsl: true, verifyTlsCertificate: true },
    });
  });

  it("parses Trino URIs with up to two path segments and SSLVerification", () => {
    expect(
      parseConnectionString(
        Engine.TRINO,
        "trino://analyst:secret@trino.example.com/hive/default?SSLVerification=NONE"
      )
    ).toEqual({
      host: "trino.example.com",
      port: "8080",
      username: "analyst",
      password: "secret",
      tls: { useSsl: true, verifyTlsCertificate: false },
    });

    expect(
      parseConnectionString(Engine.TRINO, "https://trino.example.com/hive")
    ).toEqual({
      host: "trino.example.com",
      port: "443",
      username: undefined,
      password: undefined,
      tls: { useSsl: true, verifyTlsCertificate: true },
    });

    expect(
      parseConnectionString(
        Engine.TRINO,
        "https://dbc.example.com/sql/1.0/warehouses"
      )
    ).toBeUndefined();
    expect(
      parseConnectionString(
        Engine.TRINO,
        "https://xy12345.snowflakecomputing.com"
      )
    ).toBeUndefined();
  });

  it("parses Hive JDBC URLs and beeline commands while rejecting ZooKeeper discovery, TLS, HTTP transport, and unsupported auth URLs", () => {
    expect(
      parseConnectionString(
        Engine.HIVE,
        "jdbc:hive2://hive.example.com:10000/default;user=hive;password=secret"
      )
    ).toEqual({
      host: "hive.example.com",
      port: "10000",
      username: "hive",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.HIVE,
        "jdbc:hive2://hive.example.com:10000/default;ssl=false;user=hive;password=secret"
      )
    ).toEqual({
      host: "hive.example.com",
      port: "10000",
      username: "hive",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.HIVE,
        "jdbc:hive2://hive.example.com:10000/default;transportMode=binary;auth=NONE;user=hive;password=secret"
      )
    ).toEqual({
      host: "hive.example.com",
      port: "10000",
      username: "hive",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.HIVE,
        "jdbc:hive2://hive.example.com:10000/default;auth=LDAP;user=hive;password=secret"
      )
    ).toEqual({
      host: "hive.example.com",
      port: "10000",
      username: "hive",
      password: "secret",
    });

    expect(
      parseConnectionString(
        Engine.HIVE,
        'beeline -u "jdbc:hive2://hive.example.com/app" -n admin -p s3cr3t'
      )
    ).toEqual({
      host: "hive.example.com",
      port: "10000",
      username: "admin",
      password: "s3cr3t",
    });

    expect(
      parseConnectionString(
        Engine.HIVE,
        "hive://hive.example.com:10001/default"
      )
    ).toEqual({
      host: "hive.example.com",
      port: "10001",
      username: undefined,
      password: undefined,
    });

    for (const rejected of [
      "jdbc:hive2://zk1.example.com:2181,zk2.example.com:2181/;serviceDiscoveryMode=zooKeeper",
      "jdbc:hive2://hive.example.com:10000/default;ssl=true;user=hive;password=secret",
      "jdbc:hive2://hive.example.com:10000/default?ssl=true",
      "jdbc:hive2://hive.example.com:10001/default;transportMode=http;httpPath=cliservice;user=hive;password=secret",
      "jdbc:hive2://hive.example.com:10001/default;transportMode=HTTP;user=hive;password=secret",
      "jdbc:hive2://hive.example.com:10001/default?transportMode=http&httpPath=cliservice",
      "jdbc:hive2://hive.example.com:10001/default;transport=http;http_path=cliservice",
      "jdbc:hive2://hive.example.com:10001/default;httpPath=cliservice;user=hive;password=secret",
      'beeline -u "jdbc:hive2://hive.example.com:10001/default;transportMode=http;httpPath=cliservice" -n hive -p secret',
      "jdbc:hive2://hive.example.com:10000/default;auth=noSasl;user=hive;password=secret",
      "jdbc:hive2://hive.example.com:10000/default;auth=kerberos;user=hive",
      "jdbc:hive2://hive.example.com:10000/default;principal=hive/hive.example.com@EXAMPLE.COM",
    ]) {
      expect(parseConnectionString(Engine.HIVE, rejected)).toBeUndefined();
    }

    expect(
      detectConnectionStringEngine(
        "jdbc:hive2://hive.example.com:10001/default;transportMode=http;httpPath=cliservice;user=hive;password=secret",
        Engine.POSTGRES
      )
    ).toBeUndefined();
    expect(
      detectConnectionStringEngine(
        'beeline -u "jdbc:hive2://hive.example.com:10001/default;transportMode=http;httpPath=cliservice" -n hive -p secret',
        Engine.POSTGRES
      )
    ).toBeUndefined();
  });

  it("parses Databricks URIs, JDBC URLs, and warehouse web URLs", () => {
    expect(
      parseConnectionString(
        Engine.DATABRICKS,
        "databricks://token:dapi123@dbc.cloud.databricks.com/main?http_path=/sql/1.0/warehouses/wh123"
      )
    ).toEqual({
      host: "https://dbc.cloud.databricks.com",
      port: "",
      warehouseId: "wh123",
      token: "dapi123",
    });

    expect(
      parseConnectionString(
        Engine.DATABRICKS,
        "jdbc:databricks://dbc.example.com:8443/default;transportMode=http;ssl=1;httpPath=/sql/1.0/warehouses/wh456;UID=token;PWD=abc#def"
      )
    ).toEqual({
      host: "https://dbc.example.com:8443",
      port: "",
      warehouseId: "wh456",
      token: "abc#def",
    });

    expect(
      parseConnectionString(
        Engine.DATABRICKS,
        "https://dbc.cloud.databricks.com/sql/1.0/warehouses/wh789?o=12345"
      )
    ).toEqual({
      host: "https://dbc.cloud.databricks.com",
      port: "",
      warehouseId: "wh789",
      token: undefined,
    });

    expect(
      parseConnectionString(
        Engine.DATABRICKS,
        "databricks://token:<personal-access-token>@dbc.cloud.databricks.com?http_path=/sql/1.0/warehouses/wh123"
      )
    ).toEqual({
      host: "https://dbc.cloud.databricks.com",
      port: "",
      warehouseId: "wh123",
      token: undefined,
    });
  });
});

describe("applyParsedConnectionString - additional engines", () => {
  it("switches MongoDB between SRV and multi-host while clearing stale fields", () => {
    const multiHost = applyParsedConnectionString(makeDraft(), Engine.MONGODB, {
      host: "n1.example.com",
      port: "27017",
      srv: false,
      additionalAddresses: [{ host: "n2.example.com", port: "27018" }],
      replicaSet: "rs0",
      directConnection: false,
      authenticationDatabase: "admin",
    });
    expect(multiHost.srv).toBe(false);
    expect(multiHost.additionalAddresses).toHaveLength(1);
    expect(multiHost.replicaSet).toBe("rs0");
    expect(multiHost.authenticationDatabase).toBe("admin");

    const srvDraft = applyParsedConnectionString(multiHost, Engine.MONGODB, {
      host: "cluster0.example.mongodb.net",
      port: "",
      srv: true,
      additionalAddresses: [],
      directConnection: false,
    });
    expect(srvDraft.srv).toBe(true);
    expect(srvDraft.port).toBe("");
    expect(srvDraft.additionalAddresses).toEqual([]);
    expect(srvDraft.replicaSet).toBe("");
    expect(srvDraft.directConnection).toBe(false);
    expect(srvDraft.authenticationDatabase).toBe("");
  });

  it("clears stale Redis Sentinel fields when importing Standalone and allows empty username", () => {
    const sentinel = applyParsedConnectionString(makeDraft(), Engine.REDIS, {
      host: "s1.example.com",
      port: "26379",
      redisType: DataSource_RedisType.SENTINEL,
      additionalAddresses: [{ host: "s2.example.com", port: "26379" }],
      masterName: "mymaster",
      masterUsername: "muser",
      masterPassword: "mpass",
    });
    expect(sentinel.redisType).toBe(DataSource_RedisType.SENTINEL);
    expect(sentinel.masterName).toBe("mymaster");
    expect(sentinel.updatedMasterPassword).toBe("mpass");

    const standalone = applyParsedConnectionString(sentinel, Engine.REDIS, {
      host: "redis.example.com",
      port: "6379",
      username: "",
      password: "secret",
      redisType: DataSource_RedisType.STANDALONE,
      additionalAddresses: [],
    });
    expect(standalone.username).toBe("");
    expect(standalone.updatedPassword).toBe("secret");
    expect(standalone.redisType).toBe(DataSource_RedisType.STANDALONE);
    expect(standalone.additionalAddresses).toEqual([]);
    expect(standalone.masterName).toBe("");
    expect(standalone.masterUsername).toBe("");
    expect(standalone.updatedMasterPassword).toBe("");
    expect(standalone.updatedSecretFields).not.toContain("masterPassword");
  });

  it("populates Cassandra additionalAddresses, clears Hive saslConfig, and sets database for CockroachDB/Redshift", () => {
    const cass = applyParsedConnectionString(makeDraft(), Engine.CASSANDRA, {
      host: "n1.example.com",
      port: "9042",
      additionalAddresses: [{ host: "n2.example.com", port: "9142" }],
    });
    expect(cass.additionalAddresses).toHaveLength(1);
    expect(cass.additionalAddresses[0]).toMatchObject({
      host: "n2.example.com",
      port: "9142",
    });

    const hive = applyParsedConnectionString(
      makeDraft({
        saslConfig: create(SASLConfigSchema, {
          mechanism: {
            case: "krbConfig",
            value: create(KerberosConfigSchema, { primary: "hive" }),
          },
        }),
      }),
      Engine.HIVE,
      { host: "hive.example.com", port: "10000" }
    );
    expect(hive.saslConfig).toBeUndefined();

    const crdb = applyParsedConnectionString(makeDraft(), Engine.COCKROACHDB, {
      host: "roach.example.com",
      port: "26257",
      database: "defaultdb",
    });
    expect(crdb.database).toBe("defaultdb");

    const rs = applyParsedConnectionString(makeDraft(), Engine.REDSHIFT, {
      host: "rs.example.com",
      port: "5439",
      database: "dev",
    });
    expect(rs.database).toBe("dev");
  });

  it("populates Databricks warehouseId and token and clears stale token when omitted", () => {
    const withToken = applyParsedConnectionString(
      makeDraft(),
      Engine.DATABRICKS,
      {
        host: "https://dbc.cloud.databricks.com",
        port: "",
        warehouseId: "wh123",
        token: "dapi123",
      }
    );
    expect(withToken.host).toBe("https://dbc.cloud.databricks.com");
    expect(withToken.port).toBe("");
    expect(withToken.warehouseId).toBe("wh123");
    expect(withToken.updatedToken).toBe("dapi123");
    expect(withToken.updatedSecretFields).toContain("authenticationPrivateKey");

    const withoutToken = applyParsedConnectionString(
      withToken,
      Engine.DATABRICKS,
      {
        host: "https://dbc2.cloud.databricks.com",
        port: "",
        warehouseId: "wh456",
      }
    );
    expect(withoutToken.warehouseId).toBe("wh456");
    expect(withoutToken.updatedToken).toBe("");
    expect(withoutToken.authenticationPrivateKey).toBe("");
    expect(withoutToken.updatedSecretFields).not.toContain(
      "authenticationPrivateKey"
    );
  });

  it("clears stale Snowflake key-pair credentials when applying a Snowflake URL", () => {
    const withKeyPair = makeDraft({
      authenticationPrivateKey:
        "-----BEGIN PRIVATE KEY-----\nstale\n-----END PRIVATE KEY-----",
      updatedToken:
        "-----BEGIN PRIVATE KEY-----\nstale\n-----END PRIVATE KEY-----",
      authenticationPrivateKeyPassphrase: "stale-passphrase",
      updatedSecretFields: [
        "authenticationPrivateKey",
        "authenticationPrivateKeyPassphrase",
      ],
    });

    const withPassword = applyParsedConnectionString(
      withKeyPair,
      Engine.SNOWFLAKE,
      {
        host: "org-account",
        port: "",
        username: "bytebase",
        password: "secret",
      }
    );
    expect(withPassword.host).toBe("org-account");
    expect(withPassword.username).toBe("bytebase");
    expect(withPassword.updatedPassword).toBe("secret");
    expect(withPassword.authenticationPrivateKey).toBe("");
    expect(withPassword.updatedToken).toBe("");
    expect(withPassword.authenticationPrivateKeyPassphrase).toBe("");
    expect(withPassword.updatedSecretFields).toEqual(["password"]);

    const withoutPassword = applyParsedConnectionString(
      withKeyPair,
      Engine.SNOWFLAKE,
      {
        host: "org-account-2",
        port: "",
        username: "bytebase",
      }
    );
    expect(withoutPassword.host).toBe("org-account-2");
    expect(withoutPassword.updatedPassword).toBe("");
    expect(withoutPassword.authenticationPrivateKey).toBe("");
    expect(withoutPassword.updatedToken).toBe("");
    expect(withoutPassword.authenticationPrivateKeyPassphrase).toBe("");
    expect(withoutPassword.updatedSecretFields).toEqual([]);
  });
});
