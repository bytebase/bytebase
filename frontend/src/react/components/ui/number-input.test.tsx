import * as stylex from "@stylexjs/stylex";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { NumberInput } from "./number-input";
import { controlSizeStyle } from "./styles.stylex";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

function mount(node: React.ReactNode) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(node);
  });
  return { container, root };
}

const expectClasses = (className: string | undefined, expected: string) => {
  for (const expectedClass of expected.split(" ")) {
    expect(className ?? "").toContain(expectedClass);
  }
};

describe("NumberInput", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("renders empty when value is null", () => {
    const { container } = mount(
      <NumberInput value={null} onValueChange={() => {}} />
    );
    const input = container.querySelector("input") as HTMLInputElement | null;
    expect(input).toBeInstanceOf(HTMLInputElement);
    expect(input?.value).toBe("");
  });

  test("displays the controlled numeric value as a formatted string", () => {
    const { container } = mount(
      <NumberInput value={42} onValueChange={() => {}} />
    );
    const input = container.querySelector("input") as HTMLInputElement | null;
    expect(input?.value).toBe("42");
  });

  test("renders the suffix content alongside the input", () => {
    const { container } = mount(
      <NumberInput value={10} onValueChange={() => {}} suffix="rows" />
    );
    expect(container.textContent).toContain("rows");
    expect(container.querySelector("input")).toBeInstanceOf(HTMLInputElement);
  });

  test("applies className to the outer wrapper and size variant to the input", () => {
    const { container } = mount(
      <NumberInput
        value={null}
        onValueChange={() => {}}
        className="w-60"
        size="sm"
      />
    );
    const wrapper = container.firstElementChild as HTMLElement | null;
    const input = container.querySelector("input");
    expect(wrapper?.className).toContain("w-60");
    expectClasses(
      input?.className,
      stylex.props(controlSizeStyle("sm")).className ?? ""
    );
  });

  test("forwards the disabled prop to the underlying input", () => {
    const { container } = mount(
      <NumberInput value={1} onValueChange={() => {}} disabled />
    );
    const input = container.querySelector("input") as HTMLInputElement | null;
    expect(input?.disabled).toBe(true);
  });
});
