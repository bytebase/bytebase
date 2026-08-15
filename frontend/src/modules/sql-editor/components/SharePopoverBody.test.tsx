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
  pushNotification: vi.fn(),
  extractProjectResourceName: vi.fn(
    (name: string) => name.split("/")[1] ?? name
  ),
  extractSavedQueryID: vi.fn((name: string) => name.split("/")[3] ?? name),
  routerResolve: vi.fn(() => ({
    href: "/sql-editor/projects/proj1/savedQueries/1",
  })),
  hasProjectPermissionV2: vi.fn(() => false),
  getProjectByName: vi.fn((name: string) => ({ name })),
  SavedQueryGrantEditor: vi.fn(() => null),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/stores/app", () => {
  // `notify` reuses the `pushNotification` vi.fn so the existing test
  // assertions on `mocks.pushNotification` keep working after the migration
  // from the Pinia helper to the app-store notification slice.
  const state = () => ({
    serverInfo: mocks.serverInfo,
    notify: mocks.pushNotification,
    currentUser: { email: "test@example.com" },
    getProjectByName: mocks.getProjectByName,
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
  // Mirrors the real helper: the creator, or a project-level
  // bb.savedQueries.setIamPolicy.
  isSavedQueryShareableV1: (sheet: { creator: string }) =>
    sheet.creator === "users/test@example.com" ||
    mocks.hasProjectPermissionV2(),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
}));

// The grant editor reads the policy over the wire and has its own tests; these
// cover the link section, so stub it out rather than mocking a policy fetch.
vi.mock("./SavedQueryGrantEditor", () => ({
  SavedQueryGrantEditor: mocks.SavedQueryGrantEditor,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    resolve: mocks.routerResolve,
  },
}));

let SharePopoverBody: typeof import("./SharePopoverBody").SharePopoverBody;

const mockSavedQuery = {
  name: "projects/proj1/savedQueries/1",
  project: "projects/proj1",
  creator: "users/test@example.com",
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

  test("renders the deep link alongside the grant editor", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
    );
    render();
    const input = container.querySelector("input") as HTMLInputElement;
    expect(input.value).toContain("/savedQueries/1");
    // The link carries location; the grant editor carries access. Both belong
    // in the popover, and the creator gets an editable one.
    expect(mocks.SavedQueryGrantEditor).toHaveBeenCalledWith(
      expect.objectContaining({ canManage: true }),
      undefined
    );
    unmount();
  });

  test("a non-creator without setIamPolicy gets a read-only grant editor", () => {
    mocks.hasProjectPermissionV2.mockReturnValue(false);
    const { render, unmount } = renderIntoContainer(
      <SharePopoverBody
        savedQuery={
          { ...mockSavedQuery, creator: "users/someone-else@example.com" } as never
        }
      />
    );
    render();
    expect(mocks.SavedQueryGrantEditor).toHaveBeenCalledWith(
      expect.objectContaining({ canManage: false }),
      undefined
    );
    unmount();
  });

  test("copy button writes to clipboard and pushes notification", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SharePopoverBody savedQuery={mockSavedQuery as never} />
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
    // query from the tree always has a link, so copy stays enabled there.
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
});
