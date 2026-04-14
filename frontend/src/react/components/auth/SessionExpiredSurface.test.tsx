import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

vi.mock("./SigninBridge", () => ({
  SigninBridge: () => <div data-testid="signin-bridge" />,
}));

vi.mock("@/store", () => ({
  useAuthStore: () => ({
    logout: vi.fn(),
  }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

import { SessionExpiredSurface } from "./SessionExpiredSurface";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SessionExpiredSurface", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("mounts into the critical root", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(<SessionExpiredSurface currentPath="/instances" />);
    });

    const criticalRoot = document.getElementById("bb-react-layer-critical");
    expect(criticalRoot).toBeInstanceOf(HTMLDivElement);
    expect(
      criticalRoot?.querySelector("[data-session-expired-surface]")
    ).toBeTruthy();
  });
});
