import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSource_AuthenticationType, DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { CreateDataSourceExample } from "./CreateDataSourceExample";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key.split(".").reduce<unknown>((value, part) => (value as Record<string, unknown>)[part], enUS) }),
}));
vi.mock("@/types", () => ({
  DATASOURCE_ADMIN_USER_NAME: "bytebase",
  DATASOURCE_READONLY_USER_NAME: "bytebase_readonly",
  languageOfEngineV1: () => "sql",
}));
vi.mock("@/lib/clipboard", () => ({ writeTextToClipboard: vi.fn() }));
vi.mock("@/components/ui/copy-button", () => ({ CopyButton: () => null }));
afterEach(cleanup);

const renderExample = (engine: Engine, dataSourceType: DataSourceType, authenticationType = DataSource_AuthenticationType.PASSWORD) =>
  render(<CreateDataSourceExample engine={engine} dataSourceType={dataSourceType} authenticationType={authenticationType} createInstanceFlag={true} />).container.querySelector("pre")?.textContent ?? "";

test.each([
  [Engine.TIDB, true],
  [Engine.COCKROACHDB, false],
  [Engine.BIGQUERY, false],
])("shows the read-only description independently of setup examples for engine %s", (engine, hasExample) => {
  render(<CreateDataSourceExample engine={engine as Engine} dataSourceType={DataSourceType.READ_ONLY} authenticationType={DataSource_AuthenticationType.PASSWORD} createInstanceFlag={false} />);
  expect(screen.getByText("This is the connection used by Bytebase to perform read-only operations such as SELECT query.")).toBeTruthy();
  expect(screen.queryByText("Show how to create") !== null).toBe(hasExample);
});

describe.each([
  ["password", DataSource_AuthenticationType.PASSWORD],
  ["AWS RDS IAM", DataSource_AuthenticationType.AWS_RDS_IAM],
  ["Google Cloud SQL IAM", DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM],
])("MySQL admin example with %s authentication", (_, authenticationType) => {
  test("grants no privilege that only some MySQL versions accept", () => {
    const example = renderExample(Engine.MYSQL, DataSourceType.ADMIN, authenticationType);
    expect(example).toContain("ON *.* to bytebase@'%';");
    expect(example).not.toMatch(/\/\*!|SET_USER_ID|SET_ANY_DEFINER|ALLOW_NONEXISTENT_DEFINER/);
  });

  test("warns that the definer privilege depends on the version and links to the MySQL docs", () => {
    renderExample(Engine.MYSQL, DataSourceType.ADMIN, authenticationType);
    expect(screen.getByRole("alert").textContent).toContain("also grant SET_USER_ID on MySQL 8.0 or SET_ANY_DEFINER on MySQL 8.4 and later.");
    expect(screen.getByRole("link").getAttribute("href")).toBe("https://docs.bytebase.com/get-started/connect/mysql?source=console#create-a-user-for-bytebase");
  });
});

test("Oracle admin example double-quotes the password and warns about Amazon RDS", () => {
  expect(renderExample(Engine.ORACLE, DataSourceType.ADMIN)).toBe(
    [
      "-- Run in the database Bytebase connects to.",
      "-- In a container database (CDB), that is the pluggable database, not the root.",
      'CREATE USER bytebase IDENTIFIED BY "YOUR_DB_PWD";',
      "GRANT ALL PRIVILEGES TO bytebase;",
    ].join("\n")
  );
  expect(screen.getByRole("alert").textContent).toContain("GRANT ALL PRIVILEGES doesn't work on Amazon RDS for Oracle.");
  expect(screen.getByRole("link").getAttribute("href")).toBe("https://docs.bytebase.com/get-started/connect/oracle?source=console#create-a-user-for-bytebase");
});

test.each([
  ["MySQL read-only", Engine.MYSQL, DataSourceType.READ_ONLY],
  ["TiDB admin", Engine.TIDB, DataSourceType.ADMIN],
])("shows no user setup warning for the %s example", (_, engine, dataSourceType) => {
  expect(renderExample(engine as Engine, dataSourceType as DataSourceType)).not.toBe("");
  expect(screen.queryByRole("alert")).toBeNull();
});
