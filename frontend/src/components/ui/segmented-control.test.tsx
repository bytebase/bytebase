import { act, createElement, type ReactElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { SegmentedControl } from "./segmented-control";

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

const providerOptions = [
  { value: "password", label: "Password" },
  { value: "vault", label: "HashiCorp Vault" },
  { value: "aws", label: "AWS Secrets Manager" },
  { value: "gcp", label: "Google Secret Manager" },
  { value: "azure", label: "Azure Key Vault", disabled: true },
];

describe("SegmentedControl", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("keeps wrapped provider segments separated by control-border gutters", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(SegmentedControl, {
        ariaLabel: "Password source",
        "aria-describedby": "password-source-description",
        value: "password",
        onValueChange: () => undefined,
        options: providerOptions,
      })
    );

    const group = container.querySelector('[role="radiogroup"]');
    expect(group?.className).toContain("flex-wrap");
    expect(group?.className).toContain("gap-px");
    expect(group?.className).toContain("border-control-border");
    expect(group?.className).toContain("bg-control-border");
    expect(group?.className?.split(" ")).not.toContain("p-px");
    expect(group?.getAttribute("aria-describedby")).toBe(
      "password-source-description"
    );

    const segments = [...container.querySelectorAll("label")];
    expect(segments).toHaveLength(providerOptions.length);
    expect(segments.every((segment) => segment.className.includes("flex-auto"))).toBe(
      true
    );
    expect(segments.every((segment) => !segment.className.includes("border-l"))).toBe(
      true
    );
    expect(segments[0]?.className).toContain("bg-accent");
    expect(segments[4]?.className).toContain("cursor-not-allowed");
    expect(segments[4]?.className).toContain("opacity-50");

    unmount();
  });

  test("keeps radio semantics and ignores disabled segments", () => {
    const onValueChange = vi.fn();
    function ControlledSegmentedControl() {
      const [value, setValue] = useState("password");
      return (
        <SegmentedControl
          ariaLabel="Password source"
          value={value}
          onValueChange={(nextValue) => {
            onValueChange(nextValue);
            setValue(nextValue);
          }}
          options={providerOptions}
        />
      );
    }

    const { container, unmount } = renderIntoContainer(
      createElement(ControlledSegmentedControl)
    );
    const radios = [
      ...container.querySelectorAll<HTMLElement>('[role="radio"]'),
    ];

    expect(radios).toHaveLength(providerOptions.length);
    expect(radios[0]?.getAttribute("aria-checked")).toBe("true");
    act(() => {
      radios[2]?.click();
    });
    expect(onValueChange).toHaveBeenLastCalledWith("aws");
    expect(radios[2]?.getAttribute("aria-checked")).toBe("true");

    act(() => {
      radios[4]?.click();
    });
    expect(onValueChange).toHaveBeenCalledTimes(1);
    expect(radios[4]?.getAttribute("aria-disabled")).toBe("true");

    unmount();
  });

  test("does not replace a selected disabled segment background on hover", () => {
    const { container, unmount } = renderIntoContainer(
      createElement(SegmentedControl, {
        ariaLabel: "Database sync",
        value: "azure",
        onValueChange: () => undefined,
        options: providerOptions,
      })
    );

    const selectedDisabledSegment = container.querySelectorAll("label")[4];
    expect(selectedDisabledSegment?.className).toContain("bg-accent");
    expect(selectedDisabledSegment?.className).not.toContain(
      "hover:bg-background"
    );

    unmount();
  });
});
