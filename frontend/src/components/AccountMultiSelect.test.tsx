import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  listUsers: vi.fn(),
  listGroups: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "current@example.com" }),
}));

vi.mock("@/hooks/useClickOutside", () => ({
  useClickOutside: () => {},
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    listUsers: mocks.listUsers,
    listGroups: mocks.listGroups,
  });
  return {
    useAppStore: (selector: (s: ReturnType<typeof state>) => unknown) =>
      selector(state()),
  };
});

vi.mock("@/types", () => ({
  AccountType: {
    USER: 0,
    SERVICE_ACCOUNT: 1,
    WORKLOAD_IDENTITY: 2,
  },
  ALL_USERS_USER_EMAIL: "allUsers",
  getAccountTypeByEmail: () => 0,
  serviceAccountBindingPrefix: "serviceAccount:",
  userBindingPrefix: "user:",
  workloadIdentityBindingPrefix: "workloadIdentity:",
}));

vi.mock("@/utils", () => ({
  getDefaultPagination: () => 50,
  isValidEmail: (value: string) => value.includes("@"),
}));

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: ({
    onChange,
    value,
  }: {
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    value: string;
  }) => <input data-testid="search" value={value} onChange={onChange} />,
}));

vi.mock("@/components/HighlightLabelText", () => ({
  HighlightLabelText: ({
    keyword,
    text,
  }: {
    keyword?: string;
    text: string;
  }) => <b data-keyword={keyword}>{text}</b>,
}));

vi.mock("./UserAvatar", () => ({
  getAvatarColor: () => "#000000",
  getInitials: () => "A",
}));

let AccountMultiSelect: typeof import("./AccountMultiSelect").AccountMultiSelect;

beforeEach(async () => {
  vi.useFakeTimers();
  mocks.listUsers.mockResolvedValue({
    users: [
      {
        name: "users/alice@example.com",
        email: "alice@example.com",
        title: "Alice",
      },
    ],
    nextPageToken: "",
  });
  mocks.listGroups.mockResolvedValue({ groups: [], nextPageToken: "" });
  ({ AccountMultiSelect } = await import("./AccountMultiSelect"));
});

afterEach(() => {
  vi.useRealTimers();
  document.body.innerHTML = "";
});

describe("AccountMultiSelect", () => {
  test("highlights the search query in account results", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() => {
      root.render(<AccountMultiSelect value={[]} onChange={() => {}} />);
    });
    act(() => {
      container.firstElementChild?.firstElementChild?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "ali");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      vi.advanceTimersByTime(300);
      await Promise.resolve();
      await Promise.resolve();
    });

    const title = Array.from(container.querySelectorAll("b")).find(
      (element) => element.textContent === "Alice"
    );
    expect(title?.dataset.keyword).toBe("ali");

    act(() => root.unmount());
  });

  test("hides excluded accounts from the dropdown", async () => {
    mocks.listUsers.mockResolvedValue({
      users: [
        {
          name: "users/alice@example.com",
          email: "alice@example.com",
          title: "Alice",
        },
        {
          name: "users/bob@example.com",
          email: "bob@example.com",
          title: "Bob",
        },
      ],
      nextPageToken: "",
    });
    mocks.listGroups.mockResolvedValue({
      groups: [
        {
          name: "groups/g1@example.com",
          email: "g1@example.com",
          title: "G1",
          members: [],
        },
        {
          name: "groups/g2@example.com",
          email: "g2@example.com",
          title: "G2",
          members: [],
        },
      ],
      nextPageToken: "",
    });

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={() => {}}
          excludeAccounts={["user:alice@example.com", "group:g1@example.com"]}
        />
      );
      await Promise.resolve();
      await Promise.resolve();
    });
    await act(async () => {
      container.firstElementChild?.firstElementChild?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    const rendered = Array.from(container.querySelectorAll("b")).map(
      (element) => element.textContent
    );
    expect(rendered).toContain("Bob");
    expect(rendered).toContain("G2");
    expect(rendered).not.toContain("Alice");
    expect(rendered).not.toContain("G1");

    // The fetch over-fetches by the exclusion count so filtering cannot
    // shrink the pickable page below the default.
    expect(mocks.listUsers).toHaveBeenCalledWith(
      expect.objectContaining({ pageSize: 52 })
    );

    act(() => root.unmount());
  });

  test("shows the no-data state when every fetched account is excluded", async () => {
    mocks.listGroups.mockResolvedValue({ groups: [], nextPageToken: "" });

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={() => {}}
          excludeAccounts={["user:alice@example.com"]}
        />
      );
      await Promise.resolve();
      await Promise.resolve();
    });
    await act(async () => {
      container.firstElementChild?.firstElementChild?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    expect(container.textContent).toContain("common.no-data");
    act(() => root.unmount());
  });

  test("Escape closes only the dropdown and does not bubble further", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    // Listen above the React root (React 18 delegates at the container, so a
    // same-node listener would fire regardless of stopPropagation) — the real
    // popover's Escape dismissal also listens at the document level.
    const outerKeydown = vi.fn();
    document.addEventListener("keydown", outerKeydown);
    await act(async () => {
      root.render(<AccountMultiSelect value={[]} onChange={() => {}} />);
      await Promise.resolve();
      await Promise.resolve();
    });
    await act(async () => {
      container.firstElementChild?.firstElementChild?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    const search = container.querySelector('[data-testid="search"]');
    expect(search).not.toBeNull();

    await act(async () => {
      search?.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
      );
    });

    expect(container.querySelector('[data-testid="search"]')).toBeNull();
    expect(outerKeydown).not.toHaveBeenCalled();
    document.removeEventListener("keydown", outerKeydown);
    act(() => root.unmount());
  });

  test("renders a caller-provided placeholder when empty", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={() => {}}
          placeholder="Add users or groups"
        />
      );
    });
    expect(container.textContent).toContain("Add users or groups");
    act(() => root.unmount());
  });
});
