import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { UserPasswordSection } from "./UserPasswordSection";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("UserPasswordSection", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  test("marks invalid password fields with the shared error state", () => {
    act(() => {
      root.render(
        <UserPasswordSection
          password="short"
          passwordConfirm="different"
          onPasswordChange={() => undefined}
          onPasswordConfirmChange={() => undefined}
        />
      );
    });

    const inputs = container.querySelectorAll("input");
    expect(inputs).toHaveLength(2);
    expect(inputs[0]?.getAttribute("aria-invalid")).toBe("true");
    expect(inputs[1]?.getAttribute("aria-invalid")).toBe("true");
  });
});
