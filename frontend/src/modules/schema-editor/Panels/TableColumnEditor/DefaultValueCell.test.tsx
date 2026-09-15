import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { ColumnMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { DefaultValueCell } from "./DefaultValueCell";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

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

describe("DefaultValueCell", () => {
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
