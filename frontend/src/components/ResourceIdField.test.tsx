import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ResourceIdField } from "./ResourceIdField";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("ResourceIdField", () => {
  afterEach(cleanup);

  test("forwards autocomplete to the editable input", () => {
    const { container } = render(
      <ResourceIdField
        value="project-id"
        resourceName="Project"
        autoComplete="off"
      />
    );
    fireEvent.click(container.querySelector("button")!);

    expect(container.querySelector("input")).toHaveAttribute(
      "autocomplete",
      "off"
    );
  });
});
