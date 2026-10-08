import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  DatabaseMetadataSchema,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import {
  AlgorithmSchema,
  SemanticTypeSetting_SemanticTypeSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { getSemanticTemplateList } from "@/types/semanticTypes";
import { MarkSensitiveDataSheet } from "./MarkSensitiveDataSheet";

const mocks = vi.hoisted(() => ({
  metadata: { schemas: [] } as unknown,
  updateColumnCatalog: vi.fn(async () => {}),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { effect?: string }) =>
      options?.effect ? `${key}: ${options.effect}` : key,
  }),
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
  await selectOption("settings.sensitive-data.columns-to-mask", "email");
  const phone = await screen.findByRole("option", { name: "phone" });
  fireEvent.pointerDown(phone, { pointerType: "mouse" });
  fireEvent.click(phone);
  expect(screen.getByRole("option", { name: "email" })).toHaveAttribute(
    "aria-selected",
    "true"
  );
  expect(phone).toHaveAttribute("aria-selected", "true");
  fireEvent.keyDown(phone, { key: "Escape" });
  fireEvent.click(
    screen.getByRole("button", {
      name: "settings.sensitive-data.apply-masking",
    })
  );
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
  await selectOption("settings.sensitive-data.columns-to-mask", "email");
  fireEvent.keyDown(
    screen.getByLabelText("settings.sensitive-data.columns-to-mask"),
    { key: "Escape" }
  );
  await selectOption("common.table", "orders");
  expect(
    screen.getByLabelText("settings.sensitive-data.columns-to-mask")
      .textContent
  ).toBe("common.select");
  expect(
    screen.getByRole("button", {
      name: "settings.sensitive-data.apply-masking",
    })
  ).toBeDisabled();
});

test("defaults to the first schema without selecting a table or column", () => {
  render(<MarkSensitiveDataSheet {...props} />);

  expect(screen.getByLabelText("common.schema").textContent).toBe("public");
  expect(screen.getByLabelText("common.table").hasAttribute("disabled")).toBe(
    false
  );
  expect(screen.getByLabelText("common.table").textContent).toBe("common.select");
  expect(
    screen.getByLabelText("settings.sensitive-data.columns-to-mask")
  ).toHaveAttribute("aria-disabled", "true");
});

test("describes the masking task with outcome-specific labels", () => {
  render(<MarkSensitiveDataSheet {...props} />);

  expect(
    screen.getByText("settings.sensitive-data.mark-sensitive-data-description")
  ).toBeTruthy();
  expect(
    screen.getByLabelText("settings.sensitive-data.columns-to-mask")
  ).toBeTruthy();
  expect(
    screen.getByText("settings.sensitive-data.semantic-type-description")
  ).toBeTruthy();
  expect(
    screen.getByRole("button", {
      name: "settings.sensitive-data.apply-masking",
    })
  ).toBeTruthy();
});

test("renders the semantic type title and explains a custom masking effect", async () => {
  const semanticType = create(SemanticTypeSetting_SemanticTypeSchema, {
    id: "email",
    title: "Email",
    algorithm: create(AlgorithmSchema, {
      mask: {
        case: "fullMask",
        value: { substitution: "*" },
      },
    }),
  });
  render(
    <MarkSensitiveDataSheet
      {...props}
      semanticTypeList={[
        ...getSemanticTemplateList(((key: string) => key) as never),
        semanticType,
      ]}
    />
  );

  const select = screen.getByLabelText(
    "settings.sensitive-data.semantic-types.table.semantic-type"
  );
  expect(select.textContent).toContain(
    "dynamic.settings.sensitive-data.semantic-types.template.bb-default.title"
  );
  expect(select.textContent).not.toContain("bb.default");

  fireEvent.click(select);
  const option = await screen.findByRole("option", { name: /Email/ });
  expect(option.textContent).toContain(
    "settings.sensitive-data.semantic-types.masking-effect"
  );
  expect(option.textContent).toContain(
    "settings.sensitive-data.algorithms.full-mask.self"
  );
});

test("explains the masking effect for built-in semantic types", async () => {
  const semanticTypeList = getSemanticTemplateList(
    ((key: string) => key) as never
  );
  render(
    <MarkSensitiveDataSheet
      {...props}
      semanticTypeList={semanticTypeList}
    />
  );

  const select = screen.getByLabelText(
    "settings.sensitive-data.semantic-types.table.semantic-type"
  );
  fireEvent.click(select);

  const defaultOption = await screen.findByRole("option", {
    name: /dynamic\.settings\.sensitive-data\.semantic-types\.template\.bb-default\.title/,
  });
  expect(defaultOption.textContent).toContain(
    "dynamic.settings.sensitive-data.semantic-types.template.bb-default.algorithm.description"
  );
  const partialOption = await screen.findByRole("option", {
    name: /dynamic\.settings\.sensitive-data\.semantic-types\.template\.bb-default-partial\.title/,
  });
  const partialDescription = screen.getByText(
    "dynamic.settings.sensitive-data.semantic-types.template.bb-default-partial.algorithm.description"
  );
  expect(partialDescription).toHaveClass("whitespace-normal");
  expect(partialOption).toContainElement(partialDescription);
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
