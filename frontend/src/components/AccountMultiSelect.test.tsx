import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  listUsers: vi.fn(),
  listGroups: vi.fn(),
  listServiceAccounts: vi.fn(),
  listWorkloadIdentities: vi.fn(),
  hasWorkspacePermission: vi.fn(),
  hasProjectPermission: vi.fn(),
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
  const state = {
    listUsers: mocks.listUsers,
    listGroups: mocks.listGroups,
    listServiceAccounts: mocks.listServiceAccounts,
    listWorkloadIdentities: mocks.listWorkloadIdentities,
    projectsByName: {
      "projects/project-1": { name: "projects/project-1" },
    },
    hasWorkspacePermission: mocks.hasWorkspacePermission,
    hasProjectPermission: mocks.hasProjectPermission,
  };
  return {
    useAppStore: (selector: (s: typeof state) => unknown) => selector(state),
  };
});

vi.mock("@/types", () => ({
  AccountType: {
    USER: 0,
    SERVICE_ACCOUNT: 1,
    WORKLOAD_IDENTITY: 2,
  },
  ALL_USERS_USER_EMAIL: "allUsers",
  getAccountTypeByFullname: (value: string) =>
    value.startsWith("serviceAccounts/")
      ? 1
      : value.startsWith("workloadIdentities/")
        ? 2
        : 0,
  getAccountTypeByEmail: (value: string) =>
    !value.includes(":") && value.endsWith("@service.bytebase.com")
      ? 1
      : !value.includes(":") && value.endsWith("@workload.bytebase.com")
        ? 2
        : 0,
  getServiceAccountNameInBinding: (email: string) => `serviceAccount:${email}`,
  getWorkloadIdentityNameInBinding: (email: string) =>
    `workloadIdentity:${email}`,
  groupBindingPrefix: "group:",
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
  mocks.hasWorkspacePermission.mockReturnValue(true);
  mocks.hasProjectPermission.mockReturnValue(true);
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
  mocks.listServiceAccounts.mockResolvedValue({
    serviceAccounts: [],
    nextPageToken: "",
  });
  mocks.listWorkloadIdentities.mockResolvedValue({
    workloadIdentities: [],
    nextPageToken: "",
  });
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

  test("hides the all-users shortcut while searching", async () => {
    mocks.listUsers.mockResolvedValue({ users: [], nextPageToken: "" });
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={() => {}}
          includeAllUsers
        />
      );
      await Promise.resolve();
    });
    await act(async () => {
      container.firstElementChild?.firstElementChild?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(container.textContent).toContain("settings.members.all-users");

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "terr");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain("settings.members.all-users");
    expect(container.textContent).toContain("common.no-data");
    act(() => root.unmount());
  });

  test("renders special account chips with their canonical email", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() => {
      root.render(
        <AccountMultiSelect
          value={[
            "serviceAccount:deploy@service.bytebase.com",
            "workloadIdentity:ci@workload.bytebase.com",
          ]}
          onChange={() => {}}
        />
      );
    });

    expect(container.textContent).toContain("deploy@service.bytebase.com");
    expect(container.textContent).toContain("ci@workload.bytebase.com");
    act(() => root.unmount());
  });

  test("does not offer malformed special account identifiers", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
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

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "serviceAccounts/not-an-email");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain(
      "settings.members.service-account"
    );
    act(() => root.unmount());
  });

  test("does not treat IAM member bindings as search input", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
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

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "serviceAccount:deploy@service.bytebase.com");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain(
      "settings.members.service-account"
    );
    act(() => root.unmount());
  });

  test("discovers a service account by partial search and serializes its binding", async () => {
    mocks.listServiceAccounts.mockResolvedValue({
      serviceAccounts: [
        {
          name: "serviceAccounts/deploy@service.bytebase.com",
          email: "deploy@service.bytebase.com",
          title: "Deploy bot",
        },
      ],
      nextPageToken: "",
    });
    const onChange = vi.fn();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={onChange}
          accountParents={["workspaces/default"]}
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

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "deploy");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).toContain("Deploy bot");
    expect(mocks.listServiceAccounts).toHaveBeenLastCalledWith({
      parent: "workspaces/default",
      pageSize: 50,
      showDeleted: false,
      filter: { query: "deploy" },
      skipCache: true,
    });
    const deployBot = Array.from(container.querySelectorAll("b")).find(
      (element) => element.textContent === "Deploy bot"
    );
    await act(async () => {
      deployBot?.closest(".cursor-pointer")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onChange).toHaveBeenCalledWith([
      "serviceAccount:deploy@service.bytebase.com",
    ]);
    act(() => root.unmount());
  });

  test("queries each parent, deduplicates discovered accounts, and supports workload identities", async () => {
    mocks.listServiceAccounts.mockResolvedValue({
      serviceAccounts: [
        {
          name: "serviceAccounts/deploy@service.bytebase.com",
          email: "deploy@service.bytebase.com",
          title: "Deploy bot",
        },
      ],
      nextPageToken: "",
    });
    mocks.listWorkloadIdentities.mockResolvedValue({
      workloadIdentities: [
        {
          name: "workloadIdentities/ci@workload.bytebase.com",
          email: "ci@workload.bytebase.com",
          title: "CI bot",
        },
      ],
      nextPageToken: "",
    });
    const onChange = vi.fn();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={onChange}
          accountParents={["workspaces/default", "projects/project-1"]}
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

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "ci");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(
      mocks.listServiceAccounts.mock.calls
        .slice(-2)
        .map(([params]) => params.parent)
    ).toEqual(["workspaces/default", "projects/project-1"]);
    expect(container.textContent).toContain("settings.members.service-accounts");
    expect(container.textContent).toContain(
      "settings.members.workload-identities"
    );
    expect(container.textContent?.match(/Deploy bot/g)).toHaveLength(1);
    const ciBot = Array.from(container.querySelectorAll("b")).find(
      (element) => element.textContent === "CI bot"
    );
    await act(async () => {
      ciBot?.closest(".cursor-pointer")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onChange).toHaveBeenCalledWith([
      "workloadIdentity:ci@workload.bytebase.com",
    ]);
    act(() => root.unmount());
  });

  test("hides excluded special accounts and preserves exact-email fallback after list failure", async () => {
    mocks.listServiceAccounts.mockRejectedValue(new Error("forbidden"));
    mocks.listWorkloadIdentities.mockRejectedValue(new Error("forbidden"));
    const onChange = vi.fn();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={onChange}
          accountParents={["workspaces/default"]}
          excludeAccounts={["serviceAccount:deploy@service.bytebase.com"]}
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

    const input = container.querySelector(
      '[data-testid="search"]'
    ) as HTMLInputElement;
    const valueSetter = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    )?.set;
    await act(async () => {
      valueSetter?.call(input, "deploy@service.bytebase.com");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain(
      "settings.members.service-account"
    );

    await act(async () => {
      valueSetter?.call(input, "ci@workload.bytebase.com");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });
    const ci = Array.from(container.querySelectorAll("b")).find(
      (element) => element.textContent === "ci"
    );
    await act(async () => {
      ci?.closest(".cursor-pointer")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onChange).toHaveBeenCalledWith([
      "workloadIdentity:ci@workload.bytebase.com",
    ]);
    act(() => root.unmount());
  });

  test("does not list special accounts without permission but keeps exact-email fallback", async () => {
    mocks.hasWorkspacePermission.mockReturnValue(false);
    mocks.hasProjectPermission.mockReturnValue(false);
    mocks.listServiceAccounts.mockClear();
    mocks.listWorkloadIdentities.mockClear();
    const onChange = vi.fn();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    await act(async () => {
      root.render(
        <AccountMultiSelect
          value={[]}
          onChange={onChange}
          accountParents={["workspaces/default", "projects/project-1"]}
        />
      );
      await Promise.resolve();
    });

    expect(mocks.listServiceAccounts).not.toHaveBeenCalled();
    expect(mocks.listWorkloadIdentities).not.toHaveBeenCalled();

    await act(async () => {
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
      valueSetter?.call(input, "deploy@service.bytebase.com");
      input.dispatchEvent(new Event("input", { bubbles: true }));
      await Promise.resolve();
    });
    const deploy = Array.from(container.querySelectorAll("b")).find(
      (element) => element.textContent === "deploy"
    );
    await act(async () => {
      deploy?.closest(".cursor-pointer")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(onChange).toHaveBeenCalledWith([
      "serviceAccount:deploy@service.bytebase.com",
    ]);
    act(() => root.unmount());
  });
});
