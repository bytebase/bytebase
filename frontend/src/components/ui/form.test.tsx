import { readFileSync } from "node:fs";
import { join } from "node:path";
import * as stylex from "@stylexjs/stylex";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import {
  FormControlGroup,
  FormControlRow,
  FormError,
  FormField,
  FormFieldGroup,
  FormSection,
  FormTitle,
} from "./form";
import {
  formControlGroupStyle,
  formControlRowStyle,
  formErrorStyle,
  formFieldGroupStyle,
  formFieldTitleStyle,
  formSectionStyle,
} from "./styles.stylex";

const formSource = readFileSync(join(import.meta.dirname, "form.tsx"), "utf8");

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

describe("FormError", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders field validation messages as alert text", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(FormError, null, "Title is required")
    );

    const message = container.querySelector('[data-slot="form-error"]');
    expect(message?.getAttribute("role")).toBe("alert");
    expect(message?.textContent).toBe("Title is required");
    expect(message?.className).toContain(
      stylex.props(formErrorStyle()).className ?? ""
    );

    unmount();
  });

  test("renders field groups with shared spacing", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormFieldGroup,
        null,
        createElement(FormField, {
          title: "Environment",
        })
      )
    );

    const group = container.querySelector('[data-slot="form-field-group"]');

    expect(group?.className).toContain(
      stylex.props(formFieldGroupStyle()).className ?? ""
    );
    expect(group?.children).toHaveLength(1);

    unmount();
  });

  test("renders form titles with the visual title style", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormField,
        null,
        createElement(FormTitle, null, "Database name")
      )
    );

    const title = container.querySelector('[data-slot="form-field-title"]');
    expect(title?.textContent).toBe("Database name");
    expect(title?.className).toContain(
      stylex.props(formFieldTitleStyle()).className ?? ""
    );

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

describe("Form section layouts", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders settings form sections with shared title and content slots", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormSection,
        { id: "general", title: "General" },
        createElement(
          FormFieldGroup,
          null,
          createElement(FormField, {
            title: "Default landing page",
            description: "Choose the first screen after signing in.",
          })
        )
      )
    );

    const section = container.querySelector('[data-slot="form-section"]');
    const title = container.querySelector('[data-slot="form-section-title"]');
    const content = container.querySelector(
      '[data-slot="form-section-content"]'
    );
    const fieldTitle = container.querySelector(
      '[data-slot="form-field-title"]'
    );
    const fieldHeader = container.querySelector(
      '[data-slot="form-field-header"]'
    );
    const fieldDescription = container.querySelector(
      '[data-slot="form-field-description"]'
    );

    expect(section?.getAttribute("id")).toBe("general");
    expect(section?.className).toContain(
      stylex.props(formSectionStyle()).className ?? ""
    );
    expect(title?.getAttribute("role")).toBe("heading");
    expect(title?.getAttribute("aria-level")).toBe("2");
    expect(title?.textContent).toBe("General");
    expect(content?.textContent).toBe(
      "Default landing pageChoose the first screen after signing in."
    );
    expect(fieldHeader).not.toBeNull();
    expect(fieldTitle?.textContent).toBe("Default landing page");
    expect(fieldDescription?.textContent).toBe(
      "Choose the first screen after signing in."
    );
    expect(section?.textContent).toBe(
      "GeneralDefault landing pageChoose the first screen after signing in."
    );

    unmount();
  });

  test("renders complex section title nodes without nesting them in h2", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormSection,
        {
          title: createElement(
            "span",
            { className: "inline-flex" },
            "Announcement",
            createElement("span", null, "Badge")
          ),
        },
        createElement("div", null, "Content")
      )
    );

    const title = container.querySelector('[data-slot="form-section-title"]');

    expect(title?.getAttribute("role")).toBe("heading");
    expect(title?.getAttribute("aria-level")).toBe("2");
    expect(title?.querySelector("h2")).toBeNull();
    expect(title?.textContent).toBe("AnnouncementBadge");

    unmount();
  });

  test("does not expose legacy header composition primitives", () => {
    expect(formSource).not.toContain("function FormSectionHeader");
    expect(formSource).not.toContain("function FormSectionTitle");
    expect(formSource).not.toContain("function FormSectionContent");
    expect(formSource).not.toContain("function FormFieldHeader");
    expect(formSource).not.toContain("function FormFieldTitle");
    expect(formSource).not.toContain("function FormFieldSubtitle");
    expect(formSource).not.toContain("function FormHelperText");
    expect(formSource).not.toContain("function FormMessage");
    expect(formSource).not.toContain("function FormControlAffix");
    expect(formSource).not.toContain("function FormInlineAffix");
    expect(formSource).not.toContain("function FormFieldRow");
  });

  test("documents every exposed API with a usage example", () => {
    const exportedApis = [
      "FormControlGroup",
      "FormControlRow",
      "FormError",
      "FormField",
      "FormFieldGroup",
      "FormLabel",
      "FormSection",
      "FormTitle",
    ];

    for (const api of exportedApis) {
      expect(formSource, api).toMatch(
        new RegExp(
          String.raw`/\*\*[\s\S]*?@example[\s\S]*?\*/\nfunction ${api}\b`
        )
      );
    }
  });
});
