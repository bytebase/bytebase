import * as stylex from "@stylexjs/stylex";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import {
  FormControlAffix,
  FormControlGroup,
  FormControlRow,
  FormFieldGroup,
  FormFieldRow,
  FormInlineAffix,
  FormMessage,
} from "./form";
import {
  formControlAffixStyle,
  formControlGroupStyle,
  formControlRowStyle,
  formFieldGroupStyle,
  formFieldRowStyle,
  formInlineAffixStyle,
  formMessageStyle,
} from "./styles.stylex";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

describe("FormMessage", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders field validation messages as alert text", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(FormMessage, null, "Title is required")
    );

    const message = container.querySelector('[data-slot="form-message"]');
    expect(message?.getAttribute("role")).toBe("alert");
    expect(message?.textContent).toBe("Title is required");
    expect(message?.className).toContain(
      stylex.props(formMessageStyle()).className ?? ""
    );

    unmount();
  });

  test("renders field groups and horizontal field rows with shared spacing", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormFieldGroup,
        null,
        createElement(
          FormFieldRow,
          null,
          createElement("label", null, "Environment"),
          createElement("input", { "aria-label": "Environment" })
        )
      )
    );

    const group = container.querySelector('[data-slot="form-field-group"]');
    const row = container.querySelector('[data-slot="form-field-row"]');

    expect(group?.className).toContain(
      stylex.props(formFieldGroupStyle()).className ?? ""
    );
    expect(row?.className).toContain(
      stylex.props(formFieldRowStyle()).className ?? ""
    );
    expect(row?.children).toHaveLength(2);

    unmount();
  });

  test("renders inline affix controls with non-shrinking affix text", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormInlineAffix,
        null,
        createElement("input", { "aria-label": "Email local part" }),
        createElement(FormControlAffix, null, "@"),
        createElement(FormControlAffix, null, "example.com")
      )
    );

    const affix = container.querySelector('[data-slot="form-inline-affix"]');
    const affixText = container.querySelector(
      '[data-slot="form-control-affix"]'
    );

    expect(affix?.className).toContain(
      stylex.props(formInlineAffixStyle()).className ?? ""
    );
    expect(affixText?.className).toContain(
      stylex.props(formControlAffixStyle()).className ?? ""
    );
    expect(affix?.textContent).toBe("@example.com");

    unmount();
  });
});

describe("Form control layouts", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders inline control rows with shared alignment and gap styles", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormControlGroup,
        null,
        createElement(
          FormControlRow,
          null,
          createElement("input", { "aria-label": "Parameter name" }),
          createElement("input", { "aria-label": "Parameter value" }),
          createElement("button", { type: "button" }, "Add")
        )
      )
    );

    const group = container.querySelector('[data-slot="form-control-group"]');
    const row = container.querySelector('[data-slot="form-control-row"]');

    expect(group?.className).toContain(
      stylex.props(formControlGroupStyle()).className ?? ""
    );
    expect(row?.className).toContain(
      stylex.props(formControlRowStyle()).className ?? ""
    );
    expect(row?.children).toHaveLength(3);
    expect(row?.textContent).toBe("Add");

    unmount();
  });
});
