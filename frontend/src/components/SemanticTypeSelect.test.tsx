import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  AlgorithmSchema,
  SemanticTypeSetting_SemanticTypeSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { SemanticTypeOption, SemanticTypeSelect } from "./SemanticTypeSelect";

const semanticTypes = [
  create(SemanticTypeSetting_SemanticTypeSchema, {
    id: "bb.default",
    title: "Default",
  }),
  create(SemanticTypeSetting_SemanticTypeSchema, {
    id: "bb.default-partial",
    title: "Default Partial",
  }),
  create(SemanticTypeSetting_SemanticTypeSchema, {
    id: "EMAIL",
    title: "Email",
    description: "Email address",
    algorithm: create(AlgorithmSchema, {
      mask: {
        case: "fullMask",
        value: { substitution: "*" },
      },
    }),
  }),
];

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { effect?: string }) =>
      options?.effect ? `${key}: ${options.effect}` : key,
  }),
}));

vi.mock("@/hooks/useSemanticTypes", () => ({
  useSemanticTypes: () => ({ configuredSemanticTypes: [], semanticTypes }),
}));

afterEach(cleanup);

test("shows built-in and configured semantic types with their masking descriptions", async () => {
  render(
    <SemanticTypeSelect
      value="bb.default"
      onValueChange={vi.fn()}
      aria-label="Semantic type"
    />
  );

  const select = screen.getByLabelText("Semantic type");
  expect(select.textContent).toContain("Default");
  expect(select.textContent).not.toContain("bb.default");

  fireEvent.click(select);

  expect(await screen.findByRole("listbox")).toHaveClass(
    "w-(--anchor-width)"
  );
  expect(
    await screen.findByText(
      "dynamic.settings.sensitive-data.semantic-types.template.bb-default.algorithm.description"
    )
  ).toBeInTheDocument();
  expect(screen.getByRole("option", { name: /Email/ })).toHaveTextContent(
    "settings.sensitive-data.semantic-types.masking-effect: settings.sensitive-data.algorithms.full-mask.self"
  );
});

test("uses the supplied empty label for a cleared semantic type", () => {
  render(
    <SemanticTypeSelect
      value=""
      emptyLabel="No semantic type"
      onValueChange={vi.fn()}
      aria-label="Semantic type"
    />
  );

  expect(screen.getByLabelText("Semantic type").textContent).toContain(
    "No semantic type"
  );
});

test("forwards field validation to the semantic type combobox", () => {
  render(
    <>
      <SemanticTypeSelect
        value=""
        onValueChange={vi.fn()}
        aria-label="Semantic type"
        aria-invalid
        aria-describedby="semantic-type-error"
      />
      <p id="semantic-type-error">Choose a semantic type</p>
    </>
  );

  expect(screen.getByRole("combobox", { name: "Semantic type" })).toHaveAttribute(
    "aria-invalid",
    "true"
  );
  expect(
    screen.getByRole("combobox", { name: "Semantic type" })
  ).toHaveAccessibleDescription("Choose a semantic type");
});

test.each(["focus", "hover"])(
  "shows a configured semantic description only in the opt-in %s tooltip",
  async (interaction) => {
    const { container } = render(
      <SemanticTypeOption semanticType={semanticTypes[2]} descriptionAsTooltip />
    );
    const description =
      "settings.sensitive-data.semantic-types.masking-effect: settings.sensitive-data.algorithms.full-mask.self";

    expect(container.textContent).toBe("Email");
    expect(screen.queryByText(description)).not.toBeInTheDocument();
    const info = screen.getByRole("button", { name: "common.description" });
    expect(info).not.toHaveAttribute("tabindex", "-1");
    expect(info.querySelector("svg")).toHaveClass("size-3.5");

    if (interaction === "focus") {
      fireEvent.focus(info);
    } else {
      fireEvent.pointerEnter(info, { pointerType: "mouse" });
      fireEvent.mouseEnter(info);
      fireEvent.mouseMove(info);
    }

    expect(await screen.findByText(description)).toBeVisible();
    expect(container.textContent).toBe("Email");
  }
);

test("preserves a built-in type's computed description in tooltip mode", async () => {
  render(
    <SemanticTypeOption semanticType={semanticTypes[0]} descriptionAsTooltip />
  );

  fireEvent.focus(screen.getByRole("button", { name: "common.description" }));

  expect(
    await screen.findByText(
      "dynamic.settings.sensitive-data.semantic-types.template.bb-default.algorithm.description"
    )
  ).toBeVisible();
});

test("wraps the full semantic type name in tooltip mode", () => {
  const title = "LongSemanticTypeName".repeat(8);
  render(
    <SemanticTypeOption
      semanticType={create(SemanticTypeSetting_SemanticTypeSchema, {
        id: "CUSTOM",
        title,
        description: "Configured semantic type",
      })}
      descriptionAsTooltip
    />
  );

  expect(screen.getByText(title)).toHaveClass("wrap-anywhere");
  expect(screen.getByText(title)).not.toHaveClass("truncate");
});

test("does not show an empty description trigger", () => {
  render(
    <SemanticTypeOption
      semanticType={create(SemanticTypeSetting_SemanticTypeSchema, {
        id: "CUSTOM",
        title: "Custom",
      })}
      descriptionAsTooltip
    />
  );

  expect(screen.getByText("Custom")).toBeInTheDocument();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
