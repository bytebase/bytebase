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
});
