import type { ReactNode } from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  rotate: vi.fn(),
  loadWorkspaceProfile: vi.fn(),
  copied: [] as string[],
  tokenConfigured: false,
  hasRotatePermission: true,
}));

vi.mock("@/api", () => ({
  workspaceServiceClientConnect: {
    rotateDirectorySyncToken: (...args: unknown[]) => mocks.rotate(...args),
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (s: unknown) => unknown) =>
      selector({
        serverInfo: { externalUrl: "https://bb.example.com" },
        workspaceResourceName: () => "workspaces/ws1",
        getWorkspaceProfile: () => ({
          directorySyncTokenConfigured: mocks.tokenConfigured,
        }),
      }),
    {
      getState: () => ({
        loadWorkspaceProfile: mocks.loadWorkspaceProfile,
      }),
    }
  ),
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: (v: string) => {
    mocks.copied.push(v);
    return Promise.resolve(true);
  },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => mocks.hasRotatePermission,
}));

vi.mock("@/components/ExternalUrlAlert", () => ({
  ExternalUrlAlert: () => createElement("div"),
}));
vi.mock("@/components/LearnMoreLink", () => ({
  LearnMoreLink: () => createElement("span"),
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

// Render the real Sheet primitives as plain divs so the test exercises this
// component's logic rather than the overlay library.
vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

import { AADSyncSheet } from "./AADSyncSheet";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

const renderSheet = async (open = true, onClose = vi.fn()) => {
  await act(async () => {
    root.render(createElement(AADSyncSheet, { open, onClose }));
  });
  return onClose;
};

const buttonByText = (text: string) =>
  Array.from(container.querySelectorAll("button")).find((b) =>
    b.textContent?.includes(text)
  );

const tokenInput = () =>
  Array.from(container.querySelectorAll("input")).find((i) =>
    i.value.startsWith("tok-")
  );

describe("AADSyncSheet SCIM token", () => {
  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    mocks.rotate.mockReset();
    mocks.loadWorkspaceProfile.mockReset();
    mocks.copied = [];
    mocks.tokenConfigured = false;
    mocks.hasRotatePermission = true;
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
  });

  it("offers generate when no token exists and regenerate once one does", async () => {
    await renderSheet();
    expect(buttonByText("generate-token")).toBeTruthy();
    expect(buttonByText("regenerate-token")).toBeFalsy();
    expect(container.textContent).toContain("token-not-configured");

    mocks.tokenConfigured = true;
    await renderSheet();
    expect(buttonByText("regenerate-token")).toBeTruthy();
    expect(container.textContent).toContain("token-configured");
  });

  it("shows the minted token once and never renders it again after closing", async () => {
    mocks.rotate.mockResolvedValue({ token: "tok-minted-once" });

    const onClose = await renderSheet();
    await act(async () => {
      buttonByText("generate-token")?.click();
    });

    // Shown once, with the warning, and the profile is refreshed so the button
    // flips to "regenerate".
    expect(tokenInput()?.value).toBe("tok-minted-once");
    expect(container.textContent).toContain("token-shown-once");
    expect(mocks.loadWorkspaceProfile).toHaveBeenCalledWith(true);

    // Closing must clear it: the sheet stays mounted, so without an explicit
    // reset a reopen would redisplay a token that was supposed to be shown once.
    await act(async () => {
      buttonByText("common.cancel")?.click();
    });
    expect(onClose).toHaveBeenCalled();

    mocks.tokenConfigured = true;
    await renderSheet();
    expect(tokenInput()).toBeUndefined();
    expect(container.textContent).not.toContain("tok-minted-once");
    expect(container.textContent).toContain("token-configured");
  });

  it("copies the minted token verbatim", async () => {
    mocks.rotate.mockResolvedValue({ token: "tok-copy-me" });
    await renderSheet();
    await act(async () => {
      buttonByText("generate-token")?.click();
    });

    const copyButtons = Array.from(container.querySelectorAll("button")).filter(
      (b) => b.textContent === ""
    );
    await act(async () => {
      copyButtons[copyButtons.length - 1]?.click();
    });
    expect(mocks.copied).toContain("tok-copy-me");
  });

  it("keeps the sheet usable when rotation fails and shows no token", async () => {
    mocks.rotate.mockRejectedValue(new Error("permission denied"));
    await renderSheet();

    await act(async () => {
      buttonByText("generate-token")?.click();
    });

    expect(tokenInput()).toBeUndefined();
    expect(container.textContent).not.toContain("token-shown-once");
  });

  it("hides the rotate control without permission", async () => {
    mocks.hasRotatePermission = false;
    await renderSheet();
    expect(buttonByText("generate-token")).toBeFalsy();
    expect(buttonByText("regenerate-token")).toBeFalsy();
  });
});
