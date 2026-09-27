import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
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

test.each([
  [Engine.TIDB, true],
  [Engine.COCKROACHDB, false],
  [Engine.BIGQUERY, false],
])("shows the read-only description independently of setup examples for engine %s", (engine, hasExample) => {
  render(<CreateDataSourceExample engine={engine as Engine} dataSourceType={DataSourceType.READ_ONLY} authenticationType={DataSource_AuthenticationType.PASSWORD} createInstanceFlag={false} />);
  expect(screen.getByText("This is the connection used by Bytebase to perform read-only operations such as SELECT query.")).toBeTruthy();
  expect(screen.queryByText("Show how to create") !== null).toBe(hasExample);
});
