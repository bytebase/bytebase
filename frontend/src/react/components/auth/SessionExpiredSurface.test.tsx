import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

vi.mock("@/react/pages/auth/SigninPage", () => ({
  SigninPage: () => <button data-testid="signin-bridge">Sign in</button>,
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

import { Dialog, DialogContent } from "@/react/components/ui/dialog";
import { SessionExpiredSurface } from "./SessionExpiredSurface";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SessionExpiredSurface", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("mounts into the critical root", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(<SessionExpiredSurface currentPath="/instances" />);
      await Promise.resolve();
    });

    const criticalRoot = document.getElementById("bb-react-layer-critical");
    expect(criticalRoot).toBeInstanceOf(HTMLDivElement);
    expect(
      criticalRoot?.querySelector("[data-session-expired-surface]")
    ).toBeTruthy();

    await act(async () => {
      root.unmount();
    });
  });

  test("moves focus into the critical dialog", async () => {
    const backgroundButton = document.createElement("button");
    backgroundButton.textContent = "Background";
    document.body.appendChild(backgroundButton);
    backgroundButton.focus();

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(<SessionExpiredSurface currentPath="/instances" />);
      await Promise.resolve();
    });

    const criticalRoot = document.getElementById("bb-react-layer-critical");
    expect(criticalRoot).toBeInstanceOf(HTMLDivElement);

    await act(async () => {
      await vi.waitFor(() => {
        expect(document.activeElement).not.toBe(backgroundButton);
        expect(
          criticalRoot?.contains(document.activeElement as Node)
        ).toBeTruthy();
      });
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("does not let Escape dismiss lower-layer dialogs", async () => {
    const onOpenChange = vi.fn();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <>
          <Dialog open onOpenChange={onOpenChange}>
            <DialogContent>Underlying dialog</DialogContent>
          </Dialog>
          <SessionExpiredSurface currentPath="/instances" />
        </>
      );
      await Promise.resolve();
    });

    const signinButton = document.querySelector(
      "[data-testid='signin-bridge']"
    ) as HTMLButtonElement | null;

    expect(signinButton).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      signinButton?.focus();
      signinButton?.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Escape",
          bubbles: true,
          cancelable: true,
        })
      );
      await Promise.resolve();
    });

    expect(onOpenChange).not.toHaveBeenCalled();

    await act(async () => {
      root.unmount();
    });
  });

  test("keeps the agent layer inert while the critical surface is open", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    const agentRoot = document.createElement("div");

    agentRoot.id = "bb-react-layer-agent";
    document.body.appendChild(agentRoot);

    await act(async () => {
      root.render(
        <>
          <Dialog open>
            <DialogContent>Underlying dialog</DialogContent>
          </Dialog>
          <SessionExpiredSurface currentPath="/instances" />
        </>
      );
      await Promise.resolve();
    });

    await vi.waitFor(() => {
      expect(agentRoot.getAttribute("aria-hidden")).toBe("true");
    });

    await act(async () => {
      root.unmount();
    });
  });
});
