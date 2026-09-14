import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  DatabaseMetadataSchema,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { MarkSensitiveDataSheet } from "./MarkSensitiveDataSheet";

const mocks = vi.hoisted(() => ({
  metadata: { schemas: [] } as unknown,
  updateColumnCatalog: vi.fn(async () => {}),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppDatabaseMetadata", () => ({
  useAppDatabaseMetadata: () => mocks.metadata,
}));

vi.mock("@/lib/column-data-table/utils", () => ({
  updateColumnCatalog: mocks.updateColumnCatalog,
}));

const props = {
  database: create(DatabaseSchema$, {
    name: "instances/instance/databases/database",
  }),
  open: true,
  semanticTypeList: [],
  onOpenChange: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.metadata = create(DatabaseMetadataSchema, {
    schemas: [{ name: "public" }, { name: "private" }],
  });
});

afterEach(cleanup);

async function selectOption(label: string, name: string) {
  fireEvent.click(screen.getByLabelText(label));
  const option = await screen.findByRole("option", { name });
  fireEvent.pointerDown(option, { pointerType: "mouse" });
  fireEvent.click(option);
}

test("applies the semantic type to multiple selected columns in one save", async () => {
  mocks.metadata = create(DatabaseMetadataSchema, {
    schemas: [
      {
        name: "public",
        tables: [
          {
            name: "customers",
            columns: [{ name: "email" }, { name: "phone" }],
          },
        ],
      },
    ],
  });
  render(<MarkSensitiveDataSheet {...props} />);
  await selectOption("common.table", "customers");
  await selectOption("common.column", "email");
  const phone = await screen.findByRole("option", { name: "phone" });
  fireEvent.pointerDown(phone, { pointerType: "mouse" });
  fireEvent.click(phone);
  expect(screen.getByRole("option", { name: "email" })).toHaveAttribute(
    "aria-selected",
    "true"
  );
  expect(phone).toHaveAttribute("aria-selected", "true");
  fireEvent.keyDown(phone, { key: "Escape" });
  fireEvent.click(screen.getByRole("button", { name: "common.save" }));
  expect(mocks.updateColumnCatalog).toHaveBeenCalledExactlyOnceWith({
    database: props.database.name,
    schema: "public",
    table: "customers",
    column: ["email", "phone"],
    columnCatalog: { semanticType: "bb.default" },
    notification: "common.updated",
  });
});

test("clears selected columns when switching tables", async () => {
  mocks.metadata = create(DatabaseMetadataSchema, {
    schemas: [
      {
        name: "public",
        tables: [
          { name: "customers", columns: [{ name: "email" }] },
          { name: "orders", columns: [{ name: "id" }] },
        ],
      },
    ],
  });
  render(<MarkSensitiveDataSheet {...props} />);
  await selectOption("common.table", "customers");
  await selectOption("common.column", "email");
  fireEvent.keyDown(screen.getByLabelText("common.column"), { key: "Escape" });
  await selectOption("common.table", "orders");
  expect(screen.getByLabelText("common.column").textContent).toBe(
    "common.select"
  );
  expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled();
});

test("defaults to the first schema without selecting a table or column", () => {
  render(<MarkSensitiveDataSheet {...props} />);

  expect(screen.getByLabelText("common.schema").textContent).toBe("public");
  expect(screen.getByLabelText("common.table").hasAttribute("disabled")).toBe(
    false
  );
  expect(screen.getByLabelText("common.table").textContent).toBe("common.select");
  expect(screen.getByLabelText("common.column")).toHaveAttribute(
    "aria-disabled", "true"
  );
});

test("defaults when metadata arrives and preserves a manual selection until reopened", async () => {
  mocks.metadata = create(DatabaseMetadataSchema);
  const view = render(<MarkSensitiveDataSheet {...props} />);
  expect(screen.getByLabelText("common.table").hasAttribute("disabled")).toBe(
    true
  );

  mocks.metadata = create(DatabaseMetadataSchema, {
    schemas: [{ name: "public" }, { name: "private" }],
  });
  view.rerender(<MarkSensitiveDataSheet {...props} />);
  expect(screen.getByLabelText("common.schema").textContent).toBe("public");

  fireEvent.click(screen.getByLabelText("common.schema"));
  const option = await screen.findByRole("option", { name: "private" });
  fireEvent.pointerDown(option, { pointerType: "mouse" });
  fireEvent.click(option);
  view.rerender(<MarkSensitiveDataSheet {...props} />);
  expect(screen.getByLabelText("common.schema").textContent).toBe("private");

  view.rerender(<MarkSensitiveDataSheet {...props} open={false} />);
  view.rerender(<MarkSensitiveDataSheet {...props} />);
  expect(screen.getByLabelText("common.schema").textContent).toBe("public");
});
