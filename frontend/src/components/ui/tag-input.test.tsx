import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test } from "vitest";
import { TagInput } from "./tag-input";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let latest: string[] = [];

function Harness({ initial }: { initial: string[] }) {
  const [value, setValue] = useState(initial);
  latest = value;
  return (
    <TagInput
      value={value}
      onChange={setValue}
      removeLabel={(item) => `Remove ${item}`}
    />
  );
}

const render = (initial: string[]) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => root.render(<Harness initial={initial} />));
  const input = container.querySelector("input") as HTMLInputElement;
  const type = (text: string) =>
    act(() => {
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(input, text);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  const press = (key: string) =>
    act(() => {
      input.dispatchEvent(
        new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true })
      );
    });
  return { container, root, type, press };
};

describe("TagInput", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("adds a typed value on Enter and ignores blanks and duplicates", () => {
    const { root, type, press } = render(["bytebase"]);
    type(" sts.example.com ");
    press("Enter");
    expect(latest).toEqual(["bytebase", "sts.example.com"]);
    type("bytebase");
    press("Enter");
    type("   ");
    press("Enter");
    expect(latest).toEqual(["bytebase", "sts.example.com"]);
    act(() => root.unmount());
  });

  test("shows read-only values without an input or remove buttons", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() =>
      root.render(
        <TagInput
          readOnly
          value={["bytebase"]}
          onChange={() => undefined}
          removeLabel={(item) => `Remove ${item}`}
        />
      )
    );
    expect(container.textContent).toBe("bytebase");
    expect(container.querySelector("input")).toBeNull();
    expect(container.querySelector("button")).toBeNull();
    act(() => root.unmount());
  });

  test("removes a value with its button or Backspace", () => {
    const { container, root, press } = render(["a", "b", "c"]);
    act(() => {
      (
        container.querySelector('[aria-label="Remove b"]') as HTMLButtonElement
      ).click();
    });
    expect(latest).toEqual(["a", "c"]);
    press("Backspace");
    expect(latest).toEqual(["a"]);
    act(() => root.unmount());
  });
});
