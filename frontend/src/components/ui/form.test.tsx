import * as stylex from "@stylexjs/stylex";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  FormControlGroup,
  FormControlRow,
  FormError,
  FormField,
  FormLabel,
  ResponsiveFormLayout,
  FormFieldGroup,
  FormSection,
  FormTitle,
} from "./form";
import {
  denseFormContentStyle,
  denseFormHeaderStyle,
  denseFormSectionStyle,
  denseFormTitleStyle,
  formControlGroupStyle,
  formControlRowStyle,
  formErrorStyle,
  formFieldGroupStyle,
  formFieldSectionGapStyle,
  formFieldTitleStyle,
  formLabelStyle,
  formSectionDescriptionStyle,
  formSectionStyle,
  formSectionTitleStyle,
  singleColumnFormContentStyle,
  singleColumnFormHeaderStyle,
  singleColumnFormSectionStyle,
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

  test("supports section spacing between a field header and its control", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormField,
        { title: "Review policy", spacing: "section" },
        createElement("textarea")
      )
    );

    const field = container.querySelector('[data-slot="form-field"]');
    expect(field?.className).toContain(
      stylex.props(formFieldSectionGapStyle()).className ?? ""
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

  test("renders the description in the header, under the title", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormSection,
        { title: "AI review", description: "Reviews the SQL of an issue." },
        createElement("div", null, "Content")
      )
    );

    const header = container.querySelector('[data-slot="form-section-header"]');
    const description = container.querySelector(
      '[data-slot="form-section-description"]'
    );

    expect(description?.parentElement).toBe(header);
    expect(description?.previousElementSibling?.getAttribute("data-slot")).toBe(
      "form-section-title"
    );
    expect(description?.textContent).toBe("Reviews the SQL of an issue.");
    expect(description?.className).toContain(
      stylex.props(formSectionDescriptionStyle()).className ?? ""
    );

    unmount();
  });

  test("renders no description slot without a description", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(FormSection, { title: "General" }, "Content")
    );

    expect(
      container.querySelector('[data-slot="form-section-description"]')
    ).toBeNull();

    unmount();
  });

  test("keeps the page section title and puts the header above the content in the single-column layout", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormSection,
        { title: "Standard rules", layout: "single-column" },
        createElement("div", null, "Content")
      )
    );

    const section = container.querySelector('[data-slot="form-section"]');
    const header = container.querySelector('[data-slot="form-section-header"]');
    const title = container.querySelector('[data-slot="form-section-title"]');
    const content = container.querySelector(
      '[data-slot="form-section-content"]'
    );

    expect(section?.className).toContain(
      stylex.props(singleColumnFormSectionStyle()).className ?? ""
    );
    expect(section?.className).not.toContain(
      stylex.props(formSectionStyle()).className ?? ""
    );
    expect(header?.className).toContain(
      stylex.props(singleColumnFormHeaderStyle()).className ?? ""
    );
    expect(title?.className).toContain(
      stylex.props(formSectionTitleStyle()).className ?? ""
    );
    expect(content?.className).toContain(
      stylex.props(singleColumnFormContentStyle()).className ?? ""
    );
    expect(header?.nextElementSibling).toBe(content);

    unmount();
  });

  test("keeps the dense styles in the stacked layout, with a description", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(
        FormSection,
        {
          title: "Connection",
          description: "Where Bytebase reaches the instance.",
          layout: "stacked",
        },
        createElement("div", null, "Content")
      )
    );

    const slot = (name: string) =>
      container.querySelector(`[data-slot="form-section${name}"]`);

    expect(slot("")?.className).toContain(
      stylex.props(denseFormSectionStyle()).className ?? ""
    );
    expect(slot("-header")?.className).toContain(
      stylex.props(denseFormHeaderStyle()).className ?? ""
    );
    expect(slot("-title")?.className).toContain(
      stylex.props(denseFormTitleStyle()).className ?? ""
    );
    expect(slot("-content")?.className).toContain(
      stylex.props(denseFormContentStyle()).className ?? ""
    );
    expect(slot("-description")?.parentElement).toBe(slot("-header"));

    unmount();
  });
});

describe("responsive form fields", () => {
  test("keeps dependent fields compact and vertical inside a horizontal form", () => {
    const { container, unmount } = renderIntoContainer(
      <ResponsiveFormLayout layout="horizontal">
        <FormField title="Authentication">
          <ResponsiveFormLayout layout="inherit" density="compact">
            <FormField
              title="Vault URL"
              description="Use the public endpoint."
            >
              <input aria-label="Vault URL" />
            </FormField>
          </ResponsiveFormLayout>
        </FormField>
      </ResponsiveFormLayout>
    );

    const dependentField = container.querySelectorAll(
      '[data-slot="form-field"]'
    )[1];
    const title = dependentField?.querySelector(
      '[data-slot="form-field-title"]'
    );
    const input = dependentField?.querySelector("input");
    const description = dependentField?.querySelector(
      '[data-slot="form-field-description"]'
    );
    const header = dependentField?.querySelector(
      '[data-slot="form-field-header"]'
    );

    expect(
      dependentField?.querySelector('[data-slot="form-field-control"]')
    ).toBeNull();
    expect(title?.className).toContain(
      stylex.props(formLabelStyle()).className ?? ""
    );
    expect(header?.lastElementChild).toBe(description);
    expect(input?.nextElementSibling).not.toBe(description);

    unmount();
  });

  test("preserves vertical consumers and associates labels after horizontal reflow", () => {
    const container = document.createElement("div");
    document.body.append(container);
    const root = createRoot(container);
    const field = (
      <FormField>
        <FormLabel htmlFor="form-test-host">Host</FormLabel>
        <input id="form-test-host" />
      </FormField>
    );
    act(() => root.render(field));
    expect(container.querySelector('[data-slot="form-field-control"]')).toBeNull();
    expect(container.querySelector("input")?.labels?.[0]?.textContent).toBe("Host");
    act(() => root.render(
      <ResponsiveFormLayout layout="horizontal">{field}</ResponsiveFormLayout>
    ));
    expect(container.querySelector('[data-slot="form-field-control"] input')).not.toBeNull();
    expect(container.querySelector("input")?.labels?.[0]?.textContent).toBe("Host");
    act(() => root.unmount());
    container.remove();
  });
});


test("reflows with the form width and preserves input values", () => {
  let resize: ResizeObserverCallback | undefined;
  vi.stubGlobal("ResizeObserver", class {
    constructor(callback: ResizeObserverCallback) { resize = callback; }
    observe() {}
    disconnect() {}
  });
  const { container, unmount } = renderIntoContainer(
    <ResponsiveFormLayout>
      <FormField title="Name"><input defaultValue="production" /></FormField>
    </ResponsiveFormLayout>
  );
  try {
    act(() => resize?.(
      [{ contentRect: { width: 700 } } as ResizeObserverEntry],
      {} as ResizeObserver
    ));
    expect(container.querySelector('[data-slot="form-field-control"] input')).not.toBeNull();
    act(() => resize?.(
      [{ contentRect: { width: 400 } } as ResizeObserverEntry],
      {} as ResizeObserver
    ));
    expect(container.querySelector('[data-slot="form-field-control"]')).toBeNull();
    expect(container.querySelector("input")?.value).toBe("production");
  } finally {
    unmount();
    vi.unstubAllGlobals();
  }
});
