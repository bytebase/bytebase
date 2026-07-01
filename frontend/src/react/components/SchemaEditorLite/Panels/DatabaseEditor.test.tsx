import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  DatabaseMetadataSchema,
  SchemaMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { SchemaEditorProvider } from "../context";
import type { SchemaEditorContextValue } from "../types";
import { DatabaseEditor } from "./DatabaseEditor";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils", () => ({
  getDatabaseEngine: () => 0,
  hasSchemaProperty: () => true,
}));

vi.mock("./TableList", () => ({
  TableList: () => <div data-testid="table-list" />,
}));

vi.mock("../Modals/TableNamePopover", () => ({
  TableNamePopover: ({
    open,
    schema,
  }: {
    open: boolean;
    schema: { name: string };
  }) =>
    open ? <div data-testid="table-name-popover">{schema.name}</div> : null,
}));

const makeContext = (): SchemaEditorContextValue =>
  ({
    readonly: false,
    tabs: {
      addTab: vi.fn(),
    },
    editStatus: {
      getSchemaStatus: vi.fn(() => "normal"),
    },
  }) as unknown as SchemaEditorContextValue;

describe("DatabaseEditor", () => {
  test("opens the create-table popover from the toolbar button", () => {
    const db = { name: "instances/test/databases/db" } as Database;
    const schema = create(SchemaMetadataSchema, {
      name: "public",
      tables: [],
    });
    const database = create(DatabaseMetadataSchema, {
      name: "instances/test/databases/db/metadata",
      schemas: [schema],
    });

    render(
      <SchemaEditorProvider value={makeContext()}>
        <DatabaseEditor
          db={db}
          database={database}
          selectedSchemaName="public"
          onSelectedSchemaNameChange={vi.fn()}
        />
      </SchemaEditorProvider>
    );

    expect(screen.queryByTestId("table-name-popover")).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", {
        name: "schema-editor.actions.create-table",
      })
    );

    expect(screen.getByTestId("table-name-popover")).toHaveTextContent(
      "public"
    );
  });
});
