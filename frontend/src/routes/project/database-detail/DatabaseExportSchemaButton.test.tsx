import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseExportSchemaButton } from "./DatabaseExportSchemaButton";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("DatabaseExportSchemaButton", () => {
  test("uses the standard database action button height", () => {
    render(
      <DatabaseExportSchemaButton
        database={
          {
            name: "projects/demo/instances/prod/databases/customers",
          } as Database
        }
      />
    );

    const trigger = screen.getByRole("button", {
      name: /database.export-schema$/,
    });
    expect(trigger).toHaveClass("h-9");
    expect(trigger).not.toHaveClass("h-8");
  });
});
