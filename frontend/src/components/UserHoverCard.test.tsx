import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const alice = {
  name: "users/alice@example.com",
  email: "alice@example.com",
  title: "Alice",
  state: State.ACTIVE,
  profile: {
    source: "Entra ID",
    lastLoginTime: { seconds: 1n, nanos: 0 },
    lastChangePasswordTime: { seconds: 1n, nanos: 0 },
  },
  groups: ["groups/eng@example.com"],
};

const mocks = vi.hoisted(() => ({
  getOrFetchUserByIdentifier: vi.fn(async () => alice),
  getUserByIdentifier: vi.fn(() => alice as unknown),
}));

vi.mock("@/stores/app", () => {
  const buildState = () => ({
    getOrFetchUserByIdentifier: mocks.getOrFetchUserByIdentifier,
    getUserByIdentifier: mocks.getUserByIdentifier,
  });
  const useAppStore = (selector: (state: unknown) => unknown) =>
    selector(buildState());
  useAppStore.getState = () => buildState();
  return { useAppStore };
});

// Render the card body inline; the point of these tests is what the card says,
// not Base UI's hover timing.
vi.mock("@/components/ui/preview-card", () => ({
  PreviewCard: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  PreviewCardTrigger: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  PreviewCardContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="card">{children}</div>
  ),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

let UserHoverCard: typeof import("./UserHoverCard").UserHoverCard;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: async () => {
      await act(async () => {
        root.render(element);
      });
    },
    unmount: () => act(() => root.unmount()),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.getUserByIdentifier.mockReturnValue(alice);
  ({ UserHoverCard } = await import("./UserHoverCard"));
});

describe("UserHoverCard", () => {
  test("answers who the person is", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <UserHoverCard email="alice@example.com">
        <span>Alice</span>
      </UserHoverCard>
    );
    await render();

    const card = container.querySelector('[data-testid="card"]');
    expect(card?.textContent).toContain("Alice");
    expect(card?.textContent).toContain("alice@example.com");

    unmount();
  });

  test("leaves out activity timestamps, roles and groups", async () => {
    // Reading a user record needs no special permission, so anything shown
    // here is shown to every member of the workspace. Activity timestamps in
    // particular would expose everyone's login pattern workspace-wide.
    const { container, render, unmount } = renderIntoContainer(
      <UserHoverCard email="alice@example.com">
        <span>Alice</span>
      </UserHoverCard>
    );
    await render();

    const text = container.querySelector('[data-testid="card"]')?.textContent;
    expect(text).not.toContain("Entra ID");
    expect(text).not.toContain("eng@example.com");
    expect(text).not.toContain("last-sign-in");
    expect(text).not.toContain("last-password-change");

    unmount();
  });

  test("marks a deactivated account", async () => {
    mocks.getUserByIdentifier.mockReturnValue({
      ...alice,
      state: State.DELETED,
    });

    const { container, render, unmount } = renderIntoContainer(
      <UserHoverCard email="alice@example.com">
        <span>Alice</span>
      </UserHoverCard>
    );
    await render();

    expect(
      container.querySelector('[data-testid="card"]')?.textContent
    ).toContain("common.deactivated");

    unmount();
  });

  test("does not fetch until the card actually opens", async () => {
    // The card hangs off every name in a table; fetching eagerly would issue
    // one request per row.
    const { render, unmount } = renderIntoContainer(
      <UserHoverCard email="alice@example.com">
        <span>Alice</span>
      </UserHoverCard>
    );
    await render();

    expect(mocks.getOrFetchUserByIdentifier).not.toHaveBeenCalled();

    unmount();
  });
});
