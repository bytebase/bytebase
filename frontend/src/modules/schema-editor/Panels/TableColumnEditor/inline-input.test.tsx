import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ColumnMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { DataTypeCell } from "./DataTypeCell";
import { DefaultValueCell } from "./DefaultValueCell";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/utils")>();
  return {
    ...actual,
    getDataTypeSuggestionList: () => ["int", "varchar"],
  };
});

const column = (
  overrides: {
    name?: string;
    type?: string;
    hasDefault?: boolean;
    default?: string;
  } = {}
) =>
  create(ColumnMetadataSchema, {
    name: "id",
    type: "int",
    ...overrides,
  });

describe("SchemaEditorLite inline inputs", () => {
  test("disables browser suggestions for the column type input", () => {
    render(
      <DataTypeCell
        column={column()}
        engine={Engine.POSTGRES}
        readonly={false}
        onUpdateValue={vi.fn()}
      />
    );

    expect(screen.getByDisplayValue("int")).toHaveAttribute(
      "autocomplete",
      "off"
    );
  });

  test("disables browser suggestions for the default value input", () => {
    render(
      <DefaultValueCell
        column={column({ hasDefault: true, default: "0" })}
        disabled={false}
        onUpdate={vi.fn()}
      />
    );

    expect(screen.getByDisplayValue("0")).toHaveAttribute(
      "autocomplete",
      "off"
    );
  });
});
