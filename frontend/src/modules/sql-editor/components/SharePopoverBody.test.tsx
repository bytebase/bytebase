import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  serverInfo: { externalUrl: "https://example.com" } as
    | { externalUrl: string }
    | undefined,
  useCurrentUser: vi.fn(),
  patchSavedQuery: vi.fn().mockResolvedValue({}),
  pushNotification: vi.fn(),
  extractProjectResourceName: vi.fn(
    (name: string) => name.split("/")[1] ?? name
  ),
  extractSavedQueryID: vi.fn((name: string) => name.split("/")[3] ?? name),
  routerResolve: vi.fn(() => ({ href: "/sql-editor/projects/proj1/sheets/1" })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: mocks.useCurrentUser,
}));

vi.mock("@/stores/app", () => {
  // `notify` reuses the `pushNotification` vi.fn so the existing test
  // assertions on `mocks.pushNotification` keep working after the migration
  // from the Pinia helper to the app-store notification slice.
  const state = () => ({
    serverInfo: mocks.serverInfo,
    patchSavedQuery: mocks.patchSavedQuery,
    notify: mocks.pushNotification,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("@/utils", () => ({
  extractProjectResourceName: mocks.extractProjectResourceName,
  extractSavedQueryID: mocks.extractSavedQueryID,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    resolve: mocks.routerResolve,
  },
}));

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover">{children}</div>
  ),
  PopoverTrigger: ({
    children,
    render: renderEl,
  }: {
    children?: React.ReactNode;
    render?: React.ReactElement;
  }) => (
    <div data-testid="popover-trigger">
      {renderEl ? renderEl : null}
      {children}
    </div>
  ),
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let SharePopoverBody: typeof import("./SharePopoverBody").SharePopoverBody;

const mockSavedQuery = {
  name: "projects/proj1/savedQueries/1",
  project: "projects/proj1",
  creator: "users/test@example.com",
  visibility: 3 /* PRIVATE */,
  title: "test sheet",
};

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();

  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  mocks.serverInfo = { externalUrl: "https://example.com" };
  mocks.useCurrentUser.mockReturnValue({
    email: "test@example.com",
    name: "users/test@example.com",
  });
  mocks.patchSavedQuery.mockResolvedValue({});

  // Mock clipboard
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    configurable: true,
    writable: true,
  });

  ({ SharePopoverBody } = await import("./SharePopoverBody"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("SharePopoverBody", () => {
  test("renders Share title and link input", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
    );
    render();
    expect(container.textContent).toContain("common.share");
    // Should show an input with the link
    expect(container.querySelector("input")).not.toBeNull();
    unmount();
  });

  test("shows 3 visibility options when selector is opened", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
    );
    render();
    // The popover-content should contain 3 options
    const popoverContent = container.querySelector(
      "[data-testid='popover-content']"
    );
    expect(popoverContent).not.toBeNull();
    // 3 option rows each with cursor-pointer class
    const optionRows =
      popoverContent?.querySelectorAll("[data-option-row]") ?? [];
    expect(optionRows.length).toBe(3);
    unmount();
  });

  test("visibility selector disabled when user is not creator", () => {
    mocks.useCurrentUser.mockReturnValue({
      email: "other@example.com",
      name: "users/other@example.com",
    });

    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
    );
    render();
    // Trigger should have disabled styling
    const trigger = container.querySelector("[data-access-trigger]");
    expect(trigger?.getAttribute("data-disabled")).toBe("true");
    unmount();
  });

  test("handleChangeAccess calls patchSavedQuery and pushNotification but does NOT close the outer popover", async () => {
    const patchSavedQuery = mocks.patchSavedQuery;

    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
    );
    render();

    const popoverContent = container.querySelector(
      "[data-testid='popover-content']"
    );
    const optionRows = popoverContent?.querySelectorAll("[data-option-row]");
    expect(optionRows?.length).toBeGreaterThanOrEqual(1);

    // Click second option (Project Read).
    await act(async () => {
      (optionRows?.[1] as HTMLElement)?.click();
    });

    expect(patchSavedQuery).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    // The SharePopoverBody no longer signals "close me" on access
    // change — the outer share popover stays open so the user can copy
    // the just-updated link.
    unmount();
  });

  test("copy button writes to clipboard and pushes notification", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody
        savedQuery={
          { ...mockSavedQuery, visibility: 1 /* PROJECT_READ */ } as never
        }
      />
    );
    render();

    const copyBtn = container.querySelector(
      "[data-copy-btn]"
    ) as HTMLButtonElement;
    expect(copyBtn).not.toBeNull();

    await act(async () => {
      copyBtn.click();
    });

    expect(navigator.clipboard.writeText).toHaveBeenCalled();
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("copy button disabled when there is no shareable saved query link", () => {
    // No saved query (an unsaved draft) → no link → copy disabled. A saved
    // saved query from the tree always has a link, so copy is enabled regardless
    // of the current tab's status (covered by the private-saved-query test below).
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={undefined as never} />
    );
    render();

    const copyBtn = container.querySelector(
      "[data-copy-btn]"
    ) as HTMLButtonElement;
    expect(copyBtn).not.toBeNull();
    expect(copyBtn.disabled).toBe(true);
    unmount();
  });

  test("copy button enabled for a private saved query (share status ignored)", () => {
    // mockSavedQuery is PRIVATE; the copy button stays enabled regardless of
    // share status, gated only by the tab's saved state.
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
    );
    render();

    const copyBtn = container.querySelector(
      "[data-copy-btn]"
    ) as HTMLButtonElement;
    expect(copyBtn).not.toBeNull();
    expect(copyBtn.disabled).toBe(false);
    unmount();
  });
});
