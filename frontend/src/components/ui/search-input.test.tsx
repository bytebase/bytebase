import type { ChangeEvent } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SearchInput } from "./search-input";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SearchInput", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    vi.useFakeTimers();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.useRealTimers();
  });

  const type = (input: HTMLInputElement, value: string) => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(
      input,
      value
    );
    input.dispatchEvent(new Event("input", { bubbles: true }));
  };

  test("shows the raw value immediately and debounces onChange", () => {
    const onChange = vi.fn<(event: ChangeEvent<HTMLInputElement>) => void>();
    act(() => {
      root.render(<SearchInput value="" onChange={onChange} />);
    });
    const input = container.querySelector("input") as HTMLInputElement;

    act(() => type(input, "byte"));
    expect(input.value).toBe("byte");
    expect(onChange).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(299));
    expect(onChange).not.toHaveBeenCalled();

    act(() => vi.advanceTimersByTime(1));
    expect(onChange).toHaveBeenCalledOnce();
    expect(onChange.mock.calls[0][0].target.value).toBe("byte");
  });

  test("emits only the latest value", () => {
    const onChange = vi.fn<(event: ChangeEvent<HTMLInputElement>) => void>();
    act(() => {
      root.render(<SearchInput value="" onChange={onChange} />);
    });
    const input = container.querySelector("input") as HTMLInputElement;

    act(() => type(input, "b"));
    act(() => vi.advanceTimersByTime(200));
    act(() => type(input, "byte"));
    act(() => vi.advanceTimersByTime(300));

    expect(onChange).toHaveBeenCalledOnce();
    expect(onChange.mock.calls[0][0].target.value).toBe("byte");
  });

  test("synchronizes an external value change without emitting onChange", () => {
    const onChange = vi.fn<(event: ChangeEvent<HTMLInputElement>) => void>();
    act(() => {
      root.render(<SearchInput value="draft" onChange={onChange} />);
    });
    const input = container.querySelector("input") as HTMLInputElement;

    act(() => type(input, "pending"));
    act(() => {
      root.render(<SearchInput value="reset" onChange={onChange} />);
    });
    act(() => vi.advanceTimersByTime(300));

    expect(input.value).toBe("reset");
    expect(onChange).not.toHaveBeenCalled();
  });
});
