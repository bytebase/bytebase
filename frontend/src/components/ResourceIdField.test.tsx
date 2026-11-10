import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ResourceIdField } from "./ResourceIdField";

vi.mock("react-i18next", () => {
  // One function for every render, as the real hook returns: the field's
  // effects depend on it.
  const t = (key: string) => key;
  return {
    initReactI18next: { type: "3rdParty", init: () => {} },
    useTranslation: () => ({ t }),
  };
});

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

  test("regenerates the ID when it is mounted again, unless asked to keep it", async () => {
    const onChange = vi.fn();
    render(
      <ResourceIdField
        suffix
        value="payments-ab12"
        resourceName="Project"
        resourceTitle="Payments"
        onChange={onChange}
      />
    );

    await waitFor(() => expect(onChange).toHaveBeenCalled());
    expect(onChange).not.toHaveBeenCalledWith("payments-ab12");
  });

  test("keeps the ID it is mounted with until the title changes", async () => {
    const onChange = vi.fn();
    const onValidationChange = vi.fn();
    const field = (title: string) => (
      <ResourceIdField
        suffix
        keepInitialValue
        value="payments-ab12"
        resourceName="Project"
        resourceTitle={title}
        onChange={onChange}
        onValidationChange={onValidationChange}
      />
    );
    const { rerender } = render(field("Payments"));

    await waitFor(() => expect(onValidationChange).toHaveBeenCalledWith(true));
    expect(onChange).not.toHaveBeenCalled();

    rerender(field("Billing"));

    await waitFor(() =>
      expect(onChange).toHaveBeenLastCalledWith(
        expect.stringMatching(/^billing-[a-z0-9]{4}$/)
      )
    );
  });

  test("checks the ID it kept, and reports one that is no longer valid", async () => {
    const onChange = vi.fn();
    const onValidationChange = vi.fn();
    const { findByText } = render(
      <ResourceIdField
        suffix
        keepInitialValue
        value="payments-ab12"
        resourceName="Project"
        resourceTitle="Payments"
        validate={async (id) => [
          { type: "error", message: `${id} is taken` },
        ]}
        onChange={onChange}
        onValidationChange={onValidationChange}
      />
    );

    expect(await findByText("payments-ab12 is taken")).toBeInTheDocument();
    expect(onValidationChange).toHaveBeenLastCalledWith(false);
    expect(onChange).not.toHaveBeenCalled();
  });

  test("generates an ID when there is none to keep", async () => {
    const onChange = vi.fn();
    render(
      <ResourceIdField
        suffix
        keepInitialValue
        value=""
        resourceName="Project"
        resourceTitle="Payments"
        onChange={onChange}
      />
    );

    await waitFor(() =>
      expect(onChange).toHaveBeenLastCalledWith(
        expect.stringMatching(/^payments-[a-z0-9]{4}$/)
      )
    );
  });
});
