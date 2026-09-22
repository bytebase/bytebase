import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  AlgorithmSchema,
  SemanticTypeSetting_SemanticTypeSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { SemanticTypeSelect } from "./SemanticTypeSelect";

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
