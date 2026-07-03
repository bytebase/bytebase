import * as stylex from "@stylexjs/stylex";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { Button } from "./button";
import { Combobox } from "./combobox";
import {
  FormControlAffix,
  FormControlGroup,
  FormControlRow,
  FormDescription,
  FormError,
  FormField,
  FormFieldGroup,
  FormFieldRow,
  FormInlineAffix,
  FormLabel,
  FormMessage,
} from "./form";
import { Input } from "./input";
import { NumberInput } from "./number-input";
import { SegmentedControl } from "./segmented-control";
import { Select, SelectTrigger } from "./select";
import {
  controlMinHeightStyle,
  controlMultilineSizeStyle,
  controlSizeStyle,
  formControlAffixStyle,
  formControlGroupStyle,
  formControlRowStyle,
  formDescriptionStyle,
  formErrorStyle,
  formFieldGroupStyle,
  formFieldRowStyle,
  formFieldStyle,
  formInlineAffixStyle,
  formLabelStyle,
  formMessageStyle,
  interactiveRowStyle,
  listRowIconStyle,
  listRowPrimaryTextStyle,
  listRowSecondaryTextStyle,
  listRowStateClassName,
  listRowStyle,
  menuRowStateClassName,
  menuRowStyle,
  overlaySurfaceClassName,
  stickyActionFooterContentStyle,
  stickyActionFooterRightStyle,
  stickyActionFooterSideStyle,
  stickyActionFooterStyle,
} from "./styles.stylex";
import { Textarea } from "./textarea";

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

const expectStyleXClass = (className: string | undefined) => {
  expect(className).toEqual(expect.any(String));
  expect(className ?? "").toContain("x");
};

const expectClasses = (className: string | undefined, expected: string) => {
  for (const expectedClass of expected.split(" ")) {
    expect(className ?? "").toContain(expectedClass);
  }
};

describe("StyleX common UI style contracts", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("returns StyleX props for shared control and row styles", () => {
    expectStyleXClass(stylex.props(controlSizeStyle("sm")).className);
    expectStyleXClass(stylex.props(controlMinHeightStyle("lg")).className);
    expectStyleXClass(stylex.props(controlMultilineSizeStyle("md")).className);
    expectStyleXClass(stylex.props(interactiveRowStyle("md")).className);
  });

  test("returns StyleX props for shared form, menu, and list row styles", () => {
    expectStyleXClass(stylex.props(formFieldStyle()).className);
    expectStyleXClass(stylex.props(formFieldGroupStyle()).className);
    expectStyleXClass(stylex.props(formFieldRowStyle()).className);
    expectStyleXClass(stylex.props(formControlGroupStyle()).className);
    expectStyleXClass(stylex.props(formControlRowStyle()).className);
    expectStyleXClass(stylex.props(formInlineAffixStyle()).className);
    expectStyleXClass(stylex.props(formControlAffixStyle()).className);
    expectStyleXClass(stylex.props(formLabelStyle()).className);
    expectStyleXClass(stylex.props(formDescriptionStyle()).className);
    expectStyleXClass(stylex.props(formErrorStyle()).className);
    expectStyleXClass(stylex.props(formMessageStyle()).className);
    expectStyleXClass(stylex.props(menuRowStyle("md")).className);
    expectStyleXClass(stylex.props(listRowStyle("sm")).className);
    expectStyleXClass(stylex.props(listRowIconStyle()).className);
    expectStyleXClass(stylex.props(listRowPrimaryTextStyle()).className);
    expectStyleXClass(stylex.props(listRowSecondaryTextStyle()).className);
    expectClasses(
      menuRowStateClassName,
      "data-highlighted:bg-control-bg data-disabled:pointer-events-none data-disabled:opacity-50 data-selected:bg-accent/5 aria-selected:bg-accent/5 aria-disabled:pointer-events-none aria-disabled:opacity-50"
    );
    expectStyleXClass(stylex.props(stickyActionFooterStyle()).className);
    expectStyleXClass(stylex.props(stickyActionFooterContentStyle()).className);
    expectStyleXClass(stylex.props(stickyActionFooterSideStyle()).className);
    expectStyleXClass(stylex.props(stickyActionFooterRightStyle()).className);
    expectClasses(
      overlaySurfaceClassName,
      "max-h-60 overflow-y-auto overflow-x-hidden rounded-sm border border-control-border bg-background py-1 shadow-md focus:outline-hidden"
    );
  });

  test("keeps list row state separate from menu row state", () => {
    expectClasses(
      listRowStateClassName,
      "hover:bg-control-bg data-selected:bg-accent/5 aria-selected:bg-accent/5 disabled:pointer-events-none disabled:opacity-50 data-disabled:pointer-events-none data-disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50"
    );
    expect(listRowStateClassName).not.toContain("data-highlighted:");
    expect(menuRowStateClassName).toContain("data-highlighted:bg-control-bg");
  });

  test("applies StyleX classes to shared primitive consumers", () => {
    const button = renderIntoContainer(
      createElement(Button, { size: "sm" }, "Save")
    );
    const input = renderIntoContainer(createElement(Input, { size: "sm" }));
    const segmentedControl = renderIntoContainer(
      createElement(SegmentedControl, {
        ariaLabel: "Mode",
        value: "schema",
        onValueChange: () => undefined,
        size: "lg",
        options: [
          { value: "schema", label: "Schema" },
          { value: "data", label: "Data" },
        ],
      })
    );
    const select = renderIntoContainer(
      createElement(
        Select,
        null,
        createElement(SelectTrigger, { size: "lg" }, "Role")
      )
    );
    const numberInput = renderIntoContainer(
      createElement(NumberInput, {
        value: null,
        onValueChange: () => undefined,
        size: "lg",
      })
    );
    const textarea = renderIntoContainer(
      createElement(Textarea, { size: "lg" })
    );
    const formMessage = renderIntoContainer(
      createElement(FormMessage, null, "Title is required.")
    );
    const controlGroup = renderIntoContainer(
      createElement(
        FormControlGroup,
        null,
        createElement(FormControlRow, null, createElement(Input))
      )
    );
    const fieldGroup = renderIntoContainer(
      createElement(
        FormFieldGroup,
        null,
        createElement(
          FormFieldRow,
          null,
          createElement(FormLabel, null, "Environment"),
          createElement(Input)
        )
      )
    );
    const inlineAffix = renderIntoContainer(
      createElement(
        FormInlineAffix,
        null,
        createElement(Input),
        createElement(FormControlAffix, null, "@"),
        createElement(FormControlAffix, null, "example.com")
      )
    );

    expectClasses(
      button.container.querySelector("button")?.className,
      "h-7 px-2 text-xs leading-4 gap-1"
    );
    expectClasses(
      input.container.querySelector("input")?.className,
      stylex.props(controlSizeStyle("sm")).className ?? ""
    );
    expectClasses(
      segmentedControl.container.querySelector("label")?.className,
      stylex.props(controlMinHeightStyle("lg")).className ?? ""
    );
    expectClasses(
      select.container.querySelector("button")?.className,
      stylex.props(controlSizeStyle("lg")).className ?? ""
    );
    expectClasses(
      numberInput.container.querySelector("input")?.className,
      stylex.props(controlSizeStyle("lg", { paddingInline: false }))
        .className ?? ""
    );
    expectClasses(
      numberInput.container.querySelector("input")?.className,
      "px-4"
    );
    expectClasses(
      textarea.container.querySelector("textarea")?.className,
      stylex.props(controlMultilineSizeStyle("lg")).className ?? ""
    );
    expectClasses(
      formMessage.container.querySelector('[data-slot="form-message"]')
        ?.className,
      stylex.props(formMessageStyle()).className ?? ""
    );
    expectClasses(
      controlGroup.container.querySelector('[data-slot="form-control-group"]')
        ?.className,
      stylex.props(formControlGroupStyle()).className ?? ""
    );
    expectClasses(
      controlGroup.container.querySelector('[data-slot="form-control-row"]')
        ?.className,
      stylex.props(formControlRowStyle()).className ?? ""
    );
    expectClasses(
      fieldGroup.container.querySelector('[data-slot="form-field-group"]')
        ?.className,
      stylex.props(formFieldGroupStyle()).className ?? ""
    );
    expectClasses(
      fieldGroup.container.querySelector('[data-slot="form-field-row"]')
        ?.className,
      stylex.props(formFieldRowStyle()).className ?? ""
    );
    expectClasses(
      inlineAffix.container.querySelector('[data-slot="form-inline-affix"]')
        ?.className,
      stylex.props(formInlineAffixStyle()).className ?? ""
    );
    expectClasses(
      inlineAffix.container.querySelector('[data-slot="form-control-affix"]')
        ?.className,
      stylex.props(formControlAffixStyle()).className ?? ""
    );

    button.unmount();
    input.unmount();
    segmentedControl.unmount();
    select.unmount();
    numberInput.unmount();
    textarea.unmount();
    formMessage.unmount();
    controlGroup.unmount();
    fieldGroup.unmount();
    inlineAffix.unmount();
  });

  test("applies shared menu row contract to combobox option rows", () => {
    const combobox = renderIntoContainer(
      createElement(Combobox, {
        value: "alpha",
        onChange: () => undefined,
        options: [
          { value: "alpha", label: "Alpha" },
          { value: "disabled", label: "Disabled", disabled: true },
        ],
      })
    );

    const trigger = combobox.container.firstElementChild?.firstElementChild;
    expect(trigger).not.toBeNull();

    act(() => {
      trigger?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const option = Array.from(
      combobox.container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Alpha"));
    expect(option).not.toBeUndefined();
    expectClasses(
      option?.className,
      stylex.props(menuRowStyle("sm")).className ?? ""
    );
    expectClasses(option?.className, menuRowStateClassName);
    expect(option?.getAttribute("data-selected")).toBe("true");

    const disabledOption = Array.from(
      combobox.container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("Disabled"));
    expect(disabledOption?.disabled).toBe(true);
    expectClasses(disabledOption?.className, menuRowStateClassName);

    combobox.unmount();
  });

  test("applies StyleX classes to shared form primitives", () => {
    const form = renderIntoContainer(
      createElement(
        FormField,
        null,
        createElement(FormLabel, null, "Project"),
        createElement(FormDescription, null, "Select a project."),
        createElement(FormError, null, "Project is required.")
      )
    );

    expectClasses(
      form.container.firstElementChild?.className,
      stylex.props(formFieldStyle()).className ?? ""
    );
    expectClasses(
      form.container.querySelector("label")?.className,
      stylex.props(formLabelStyle()).className ?? ""
    );
    expectClasses(
      form.container.querySelector("[data-slot='form-description']")?.className,
      stylex.props(formDescriptionStyle()).className ?? ""
    );
    expectClasses(
      form.container.querySelector("[data-slot='form-error']")?.className,
      stylex.props(formErrorStyle()).className ?? ""
    );

    form.unmount();
  });
});
