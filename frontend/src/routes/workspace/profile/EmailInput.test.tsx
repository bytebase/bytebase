import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { EmailInput } from "./EmailInput";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: object) => unknown) =>
    selector({
      getWorkspaceProfile: () => ({
        domains: ["example.com"],
        enforceIdentityDomain: false,
      }),
    }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("EmailInput", () => {
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

  test.each([
    { showDomain: false, value: "INVALID@example.com" },
    { showDomain: true, value: "INVALID@example.com" },
  ])("marks an invalid email with the shared error state", ({ showDomain, value }) => {
    act(() => {
      root.render(
        <EmailInput
          value={value}
          showDomain={showDomain}
          onChange={() => undefined}
        />
      );
    });

    expect(container.querySelector("input")?.getAttribute("aria-invalid")).toBe(
      "true"
    );
  });
});
