import { create as createProto } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import type { ReactElement, ReactNode } from "react";
import { act, createContext, createElement, useContext } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { SavedQueryPolicy } from "@/types/proto-es/v1/saved_query_service_pb";
import {
  SavedQueryBinding_Level,
  SavedQueryPolicySchema,
} from "@/types/proto-es/v1/saved_query_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getSavedQueryPolicy: vi.fn<() => Promise<unknown>>(),
  setSavedQueryPolicy: vi.fn<(name: string, policy: unknown) => Promise<unknown>>(),
  batchGetOrFetchUsers: vi.fn(async () => []),
  batchGetOrFetchGroups: vi.fn(async () => []),
  notify: vi.fn(),
  usersByBinding: {} as Record<string, { title: string; email: string }>,
  groupsByBinding: {} as Record<string, { name: string; title: string }>,
  // What the picker stub hands to onChange when clicked.
  pickerSelection: [] as string[],
  pickerProps: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    currentUser: { email: "admin@x.com" },
    getSavedQueryPolicy: mocks.getSavedQueryPolicy,
    setSavedQueryPolicy: mocks.setSavedQueryPolicy,
    batchGetOrFetchUsers: mocks.batchGetOrFetchUsers,
    batchGetOrFetchGroups: mocks.batchGetOrFetchGroups,
    notify: mocks.notify,
    getUserByIdentifier: (identifier: string) =>
      mocks.usersByBinding[identifier],
    getGroupByIdentifier: (identifier: string) =>
      mocks.groupsByBinding[identifier],
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

// The picker's own behavior (search, toggle, labels) has its own tests; here a
// stub exposes the `onChange` contract: clicking it reports
// `mocks.pickerSelection` the way a user picking those accounts would.
vi.mock("@/components/AccountMultiSelect", () => ({
  AccountMultiSelect: (props: {
    value: string[];
    onChange: (value: string[]) => void;
    disabled?: boolean;
    excludeAccounts?: string[];
    placeholder?: string;
  }) => {
    mocks.pickerProps(props);
    return createElement(
      "button",
      {
        type: "button",
        "data-testid": "account-picker",
        disabled: props.disabled,
        onClick: () => props.onChange(mocks.pickerSelection),
      },
      "picker"
    );
  },
}));

// Functional stand-in for the Base UI select. Faithful on the axis that
// matters here: a bare `<SelectValue />` renders the raw stringified value
// (Base UI has no Radix-style echo of the selected item's label), while a
// render-function child maps value → label. Items are buttons that fire
// `onValueChange` so tests can drive level changes.
vi.mock("@/components/ui/select", () => {
  const SelectCtx = createContext<{
    value?: unknown;
    onValueChange?: (value: never) => void;
    disabled?: boolean;
  }>({});
  return {
    Select: ({
      children,
      value,
      onValueChange,
      disabled,
    }: {
      children: ReactNode;
      value?: unknown;
      onValueChange?: (value: never) => void;
      disabled?: boolean;
    }) =>
      createElement(
        SelectCtx.Provider,
        { value: { value, onValueChange, disabled } },
        children
      ),
    SelectTrigger: ({ children }: { children?: ReactNode }) =>
      createElement("div", { "data-testid": "select-trigger" }, children),
    SelectValue: ({
      children,
    }: {
      children?: ReactNode | ((value: unknown) => ReactNode);
    }) => {
      const ctx = useContext(SelectCtx);
      return createElement(
        "span",
        { "data-testid": "select-value" },
        typeof children === "function"
          ? children(ctx.value)
          : (children ?? String(ctx.value))
      );
    },
    SelectContent: ({ children }: { children?: ReactNode }) =>
      createElement("div", {}, children),
    SelectItem: ({
      children,
      value,
    }: {
      children?: ReactNode;
      value: unknown;
    }) => {
      const ctx = useContext(SelectCtx);
      return createElement(
        "button",
        {
          type: "button",
          "data-testid": "select-item",
          "data-value": String(value),
          disabled: ctx.disabled,
          onClick: () => ctx.onValueChange?.(value as never),
        },
        children
      );
    },
  };
});

// The hover card fetches on open; rows only need its children rendered.
vi.mock("@/components/UserHoverCard", () => ({
  UserHoverCard: ({ children }: { children: ReactNode }) => children,
}));

let SavedQueryGrantEditor: typeof import("./SavedQueryGrantEditor").SavedQueryGrantEditor;

const mockSavedQuery = {
  name: "projects/proj1/savedQueries/1",
  project: "projects/proj1",
  creator: "users/test@example.com",
  title: "test sheet",
} as never;

const makePolicy = (
  init?: Partial<{ bindings: unknown[]; etag: string }>
): SavedQueryPolicy =>
  createProto(SavedQueryPolicySchema, {
    bindings: [
      {
        level: SavedQueryBinding_Level.VIEWER,
        members: ["user:viewer@x.com", "group:team@x.com"],
      },
      {
        level: SavedQueryBinding_Level.EDITOR,
        members: ["user:editor@x.com"],
      },
    ],
    etag: "v1",
    ...init,
  } as never);

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: async () => {
      await act(async () => {
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

const rowFor = (container: HTMLElement, member: string) =>
  container.querySelector<HTMLElement>(
    `[data-testid="grant-row"][data-member="${member}"]`
  );

const clickLevelItem = async (scope: HTMLElement, level: number) => {
  const item = Array.from(
    scope.querySelectorAll<HTMLButtonElement>('[data-testid="select-item"]')
  ).find((candidate) => candidate.dataset.value === String(level));
  expect(item).toBeTruthy();
  await act(async () => {
    item?.click();
  });
};

const lastPickerProps = () =>
  mocks.pickerProps.mock.lastCall?.[0] as {
    value: string[];
    excludeAccounts?: string[];
    placeholder?: string;
  };

const writtenBindings = () => {
  const policy = mocks.setSavedQueryPolicy.mock.lastCall?.[1] as
    | SavedQueryPolicy
    | undefined;
  return policy?.bindings.map((binding) => ({
    level: binding.level,
    members: [...binding.members],
  }));
};

beforeEach(async () => {
  // resetAllMocks, not clearAllMocks: it also drops unconsumed mock*Once
  // queues (a test that queues a rejection but bails early must not leak it
  // into the next test's first write).
  vi.resetAllMocks();
  mocks.getSavedQueryPolicy.mockImplementation(async () => makePolicy());
  // Echo the written policy back, the way the server returns the stored one.
  mocks.setSavedQueryPolicy.mockImplementation(
    async (_name, policy) => policy as SavedQueryPolicy
  );
  mocks.usersByBinding = {
    "user:viewer@x.com": { title: "Viewer Vi", email: "viewer@x.com" },
    "user:editor@x.com": { title: "Editor Ed", email: "editor@x.com" },
  };
  mocks.groupsByBinding = {
    "group:team@x.com": { name: "groups/team@x.com", title: "Team X" },
  };
  mocks.pickerSelection = [];
  ({ SavedQueryGrantEditor } = await import("./SavedQueryGrantEditor"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("SavedQueryGrantEditor", () => {
  test("renders one row per grantee with resolved names and level labels", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    const rows = container.querySelectorAll('[data-testid="grant-row"]');
    expect(rows).toHaveLength(3);

    const viewerRow = rowFor(container, "user:viewer@x.com");
    expect(viewerRow?.textContent).toContain("Viewer Vi");
    expect(viewerRow?.textContent).toContain("viewer@x.com");
    expect(
      viewerRow?.querySelector('[data-testid="select-value"]')?.textContent
    ).toBe("sql-editor.saved-query-share.viewer");

    const groupRow = rowFor(container, "group:team@x.com");
    expect(groupRow?.textContent).toContain("Team X");

    const editorRow = rowFor(container, "user:editor@x.com");
    expect(
      editorRow?.querySelector('[data-testid="select-value"]')?.textContent
    ).toBe("sql-editor.saved-query-share.editor");

    // The list carries an accessible name via its caption.
    expect(
      container.querySelector("ul")?.getAttribute("aria-labelledby")
    ).toBe("saved-query-grantee-caption");

    // Regression for the raw-enum bug: no select renders "1" or "2".
    for (const value of container.querySelectorAll(
      '[data-testid="select-value"]'
    )) {
      expect(["1", "2"]).not.toContain(value.textContent);
    }
    unmount();
  });

  test("prefetches user and group display names referenced by the policy", async () => {
    const { render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    expect(mocks.batchGetOrFetchUsers).toHaveBeenCalledWith([
      "user:viewer@x.com",
      "user:editor@x.com",
    ]);
    expect(mocks.batchGetOrFetchGroups).toHaveBeenCalledWith([
      "group:team@x.com",
    ]);
    unmount();
  });

  test("changing a row's level moves the member and keeps the etag", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    const viewerRow = rowFor(container, "user:viewer@x.com");
    await clickLevelItem(viewerRow as HTMLElement, SavedQueryBinding_Level.EDITOR);

    expect(mocks.setSavedQueryPolicy).toHaveBeenCalledTimes(1);
    expect(writtenBindings()).toEqual([
      {
        level: SavedQueryBinding_Level.VIEWER,
        members: ["group:team@x.com"],
      },
      {
        level: SavedQueryBinding_Level.EDITOR,
        members: ["user:editor@x.com", "user:viewer@x.com"],
      },
    ]);
    const written = mocks.setSavedQueryPolicy.mock.lastCall?.[1] as SavedQueryPolicy;
    expect(written.etag).toBe("v1");
    unmount();
  });

  test("re-selecting a row's current level writes nothing", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    const viewerRow = rowFor(container, "user:viewer@x.com");
    await clickLevelItem(viewerRow as HTMLElement, SavedQueryBinding_Level.VIEWER);

    expect(mocks.setSavedQueryPolicy).not.toHaveBeenCalled();
    unmount();
  });

  test("removing a member drops them and strips the emptied binding", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    const editorRow = rowFor(container, "user:editor@x.com");
    const removeButton = editorRow?.querySelector<HTMLButtonElement>(
      'button[aria-label="common.remove"]'
    );
    expect(removeButton).toBeTruthy();
    await act(async () => {
      removeButton?.click();
    });

    expect(writtenBindings()).toEqual([
      {
        level: SavedQueryBinding_Level.VIEWER,
        members: ["user:viewer@x.com", "group:team@x.com"],
      },
    ]);
    unmount();
  });

  test("staged people do not write until Add commits them at the invite level", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    // Resting state: only the field — the commit controls do not exist yet.
    const invite = container.querySelector<HTMLElement>(
      '[data-testid="grant-invite"]'
    );
    expect(
      container.querySelector('[data-testid="grant-add"]')
    ).toBeNull();
    expect(invite?.querySelector('[data-testid="select-item"]')).toBeNull();

    // Stage a new user: the chips and the level+Add row appear, but there is
    // no policy write and no grant row yet.
    mocks.pickerSelection = ["user:new@x.com"];
    const picker = container.querySelector<HTMLButtonElement>(
      '[data-testid="account-picker"]'
    );
    await act(async () => {
      picker?.click();
    });
    expect(mocks.setSavedQueryPolicy).not.toHaveBeenCalled();
    expect(lastPickerProps().value).toEqual(["user:new@x.com"]);
    expect(rowFor(container, "user:new@x.com")).toBeNull();
    const addButton = container.querySelector<HTMLButtonElement>(
      '[data-testid="grant-add"]'
    );
    expect(addButton?.disabled).toBe(false);

    // The invite level is chosen after staging, Drive-style.
    await clickLevelItem(invite as HTMLElement, SavedQueryBinding_Level.EDITOR);
    await act(async () => {
      addButton?.click();
    });
    expect(writtenBindings()).toEqual([
      {
        level: SavedQueryBinding_Level.VIEWER,
        members: ["user:viewer@x.com", "group:team@x.com"],
      },
      {
        level: SavedQueryBinding_Level.EDITOR,
        members: ["user:editor@x.com", "user:new@x.com"],
      },
    ]);
    expect(lastPickerProps().value).toEqual([]);
    expect(rowFor(container, "user:new@x.com")).not.toBeNull();
    // The compose session is over: the commit controls fold away, the level
    // defaults back to Viewer for the next one, and focus re-homes to the
    // section instead of falling to <body>.
    expect(container.querySelector('[data-testid="grant-add"]')).toBeNull();
    expect(document.activeElement).toBe(container.querySelector("section"));

    mocks.pickerSelection = ["user:another@x.com"];
    await act(async () => {
      picker?.click();
    });
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('[data-testid="grant-add"]')
        ?.click();
    });
    expect(writtenBindings()).toContainEqual({
      level: SavedQueryBinding_Level.VIEWER,
      members: ["user:viewer@x.com", "group:team@x.com", "user:another@x.com"],
    });
    unmount();
  });

  test("the picker excludes the caller, the creator, and current grantees", async () => {
    const { render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    expect([...(lastPickerProps().excludeAccounts ?? [])].sort()).toEqual(
      [
        "group:team@x.com",
        "user:admin@x.com",
        "user:editor@x.com",
        "user:test@example.com",
        "user:viewer@x.com",
      ].sort()
    );
    expect(lastPickerProps().placeholder).toBe(
      "sql-editor.saved-query-share.add-people"
    );
    unmount();
  });

  test("Add with only already-granted selections clears staging without writing", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    mocks.pickerSelection = ["user:viewer@x.com"];
    const picker = container.querySelector<HTMLButtonElement>(
      '[data-testid="account-picker"]'
    );
    await act(async () => {
      picker?.click();
    });
    const addButton = container.querySelector<HTMLButtonElement>(
      '[data-testid="grant-add"]'
    );
    await act(async () => {
      addButton?.click();
    });

    expect(mocks.setSavedQueryPolicy).not.toHaveBeenCalled();
    expect(lastPickerProps().value).toEqual([]);
    unmount();
  });

  test("rejects non user/group grantees at selection time", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    mocks.pickerSelection = ["serviceAccount:sa@x.com"];
    const picker = container.querySelector<HTMLButtonElement>(
      '[data-testid="account-picker"]'
    );
    await act(async () => {
      picker?.click();
    });

    expect(mocks.notify).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "WARN",
        title: "sql-editor.saved-query-share.only-users-and-groups",
      })
    );
    // Nothing staged: the chip never appears and the commit row never shows.
    expect(lastPickerProps().value).toEqual([]);
    expect(container.querySelector('[data-testid="grant-add"]')).toBeNull();
    expect(mocks.setSavedQueryPolicy).not.toHaveBeenCalled();
    unmount();
  });

  test("pins the creator as a static Owner row, with a you badge for the caller", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    const ownerRow = container.querySelector<HTMLElement>(
      '[data-testid="grant-owner-row"]'
    );
    expect(ownerRow?.textContent).toContain("test@example.com");
    expect(ownerRow?.textContent).toContain(
      "sql-editor.saved-query-share.owner"
    );
    // The creator is not the caller here, so no badge; and the Owner row
    // carries no controls.
    expect(ownerRow?.textContent).not.toContain("common.you");
    expect(ownerRow?.querySelector("button")).toBeNull();
    unmount();

    const own = renderIntoContainer(
      <SavedQueryGrantEditor
        savedQuery={
          { ...(mockSavedQuery as object), creator: "users/admin@x.com" } as never
        }
        canManage={true}
      />
    );
    await own.render();
    expect(
      own.container.querySelector('[data-testid="grant-owner-row"]')
        ?.textContent
    ).toContain("common.you");
    own.unmount();
  });

  test("a creator listed in the bindings is hidden from rows but kept on write", async () => {
    mocks.getSavedQueryPolicy.mockImplementation(async () =>
      makePolicy({
        bindings: [
          {
            level: SavedQueryBinding_Level.VIEWER,
            members: ["user:test@example.com", "user:viewer@x.com"],
          },
          {
            level: SavedQueryBinding_Level.EDITOR,
            members: ["user:editor@x.com"],
          },
        ],
      })
    );
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    expect(rowFor(container, "user:test@example.com")).toBeNull();
    expect(
      container.querySelector('[data-testid="grant-owner-row"]')
    ).not.toBeNull();

    // A rewrite through another row keeps the hidden creator binding.
    const editorRow = rowFor(container, "user:editor@x.com");
    await clickLevelItem(editorRow as HTMLElement, SavedQueryBinding_Level.VIEWER);
    expect(writtenBindings()).toEqual([
      {
        level: SavedQueryBinding_Level.VIEWER,
        members: [
          "user:test@example.com",
          "user:viewer@x.com",
          "user:editor@x.com",
        ],
      },
    ]);
    unmount();
  });

  test("a failed Add keeps the staged people for retry", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    mocks.pickerSelection = ["user:new@x.com"];
    const picker = container.querySelector<HTMLButtonElement>(
      '[data-testid="account-picker"]'
    );
    await act(async () => {
      picker?.click();
    });
    mocks.setSavedQueryPolicy.mockRejectedValueOnce(new Error("boom"));
    const addButton = container.querySelector<HTMLButtonElement>(
      '[data-testid="grant-add"]'
    );
    await act(async () => {
      addButton?.click();
    });

    expect(mocks.notify).toHaveBeenCalledWith(
      expect.objectContaining({ style: "CRITICAL" })
    );
    expect(lastPickerProps().value).toEqual(["user:new@x.com"]);
    unmount();
  });

  test("an aborted write warns about the conflict and reloads the policy", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();
    expect(mocks.getSavedQueryPolicy).toHaveBeenCalledTimes(1);

    mocks.setSavedQueryPolicy.mockRejectedValueOnce(
      new ConnectError("conflict", Code.Aborted)
    );
    const editorRow = rowFor(container, "user:editor@x.com");
    await clickLevelItem(editorRow as HTMLElement, SavedQueryBinding_Level.VIEWER);

    expect(mocks.notify).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "WARN",
        title: "sql-editor.saved-query-share.policy-changed",
      })
    );
    expect(mocks.getSavedQueryPolicy).toHaveBeenCalledTimes(2);
    // The conflict reload must hand the controls back for a retry.
    expect(
      container.querySelector<HTMLButtonElement>(
        '[data-testid="account-picker"]'
      )?.disabled
    ).toBe(false);
    unmount();
  });

  test("controls stay disabled while a write is in flight", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    let finish: (policy: SavedQueryPolicy) => void = () => {};
    mocks.setSavedQueryPolicy.mockImplementationOnce(
      (_name, policy) =>
        new Promise((resolve) => {
          finish = () => resolve(policy as SavedQueryPolicy);
        })
    );
    const editorRow = rowFor(container, "user:editor@x.com");
    const removeButton = editorRow?.querySelector<HTMLButtonElement>(
      'button[aria-label="common.remove"]'
    );
    await act(async () => {
      removeButton?.click();
    });

    expect(
      container.querySelector<HTMLButtonElement>(
        '[data-testid="account-picker"]'
      )?.disabled
    ).toBe(true);
    expect(
      rowFor(container, "user:viewer@x.com")?.querySelector<HTMLButtonElement>(
        'button[aria-label="common.remove"]'
      )?.disabled
    ).toBe(true);

    await act(async () => {
      finish(makePolicy());
    });
    expect(
      container.querySelector<HTMLButtonElement>(
        '[data-testid="account-picker"]'
      )?.disabled
    ).toBe(false);
    unmount();
  });

  test("read-only view lists grantees with level text and no controls", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={false} />
    );
    await render();

    expect(container.querySelectorAll('[data-testid="grant-row"]')).toHaveLength(
      3
    );
    expect(container.querySelector('[data-testid="account-picker"]')).toBeNull();
    expect(container.querySelector('[data-testid="select-item"]')).toBeNull();
    expect(
      container.querySelector('button[aria-label="common.remove"]')
    ).toBeNull();
    expect(rowFor(container, "user:editor@x.com")?.textContent).toContain(
      "sql-editor.saved-query-share.editor"
    );
    unmount();
  });

  test("empty policy shows the private hint to managers and not-shared to readers", async () => {
    mocks.getSavedQueryPolicy.mockImplementation(async () =>
      makePolicy({ bindings: [] })
    );

    const managed = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await managed.render();
    expect(managed.container.textContent).toContain(
      "sql-editor.saved-query-share.private-hint"
    );
    managed.unmount();

    // Readers see the Owner row instead of text that would contradict it.
    const readOnly = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={false} />
    );
    await readOnly.render();
    expect(
      readOnly.container.querySelector('[data-testid="grant-owner-row"]')
    ).not.toBeNull();
    expect(readOnly.container.textContent).not.toContain(
      "sql-editor.saved-query-share.not-shared"
    );
    readOnly.unmount();
  });

  test("a write finishing after a saved-query switch cannot touch the new editor", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    // Query A: start a remove whose write resolves under our control.
    await act(async () => {
      root.render(
        <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
      );
    });
    let finish: () => void = () => {};
    mocks.setSavedQueryPolicy.mockImplementationOnce(
      (_name, policy) =>
        new Promise((resolve) => {
          finish = () => resolve(policy as SavedQueryPolicy);
        })
    );
    const editorRow = rowFor(container, "user:editor@x.com");
    await act(async () => {
      editorRow
        ?.querySelector<HTMLButtonElement>('button[aria-label="common.remove"]')
        ?.click();
    });

    // Switch to query B while A's write is still in flight.
    mocks.getSavedQueryPolicy.mockImplementation(async () =>
      makePolicy({
        bindings: [
          { level: SavedQueryBinding_Level.VIEWER, members: ["user:b@x.com"] },
        ],
        etag: "b1",
      })
    );
    await act(async () => {
      root.render(
        <SavedQueryGrantEditor
          savedQuery={
            {
              ...(mockSavedQuery as object),
              name: "projects/proj1/savedQueries/2",
            } as never
          }
          canManage={true}
        />
      );
    });
    expect(rowFor(container, "user:b@x.com")).not.toBeNull();
    // B's controls are live, not held hostage by A's in-flight write.
    expect(
      container.querySelector<HTMLButtonElement>(
        '[data-testid="account-picker"]'
      )?.disabled
    ).toBe(false);

    // A's write resolves late: B's rows must be untouched by its echo.
    await act(async () => {
      finish();
    });
    expect(rowFor(container, "user:b@x.com")).not.toBeNull();
    expect(rowFor(container, "user:editor@x.com")).toBeNull();

    act(() => root.unmount());
    container.remove();
  });

  test("bindings at unknown levels pass through rewrites untouched", async () => {
    mocks.getSavedQueryPolicy.mockImplementation(async () =>
      makePolicy({
        bindings: [
          {
            level: SavedQueryBinding_Level.VIEWER,
            members: ["user:viewer@x.com"],
          },
          { level: 99, members: ["user:future@x.com"] },
        ],
      })
    );
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    // The unknown level renders no row, but survives the rewrite verbatim.
    expect(rowFor(container, "user:future@x.com")).toBeNull();
    const viewerRow = rowFor(container, "user:viewer@x.com");
    await clickLevelItem(
      viewerRow as HTMLElement,
      SavedQueryBinding_Level.EDITOR
    );
    expect(writtenBindings()).toEqual([
      {
        level: SavedQueryBinding_Level.EDITOR,
        members: ["user:viewer@x.com"],
      },
      { level: 99, members: ["user:future@x.com"] },
    ]);
    unmount();
  });

  test("deselecting every chip resets the invite level to Viewer", async () => {
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryGrantEditor savedQuery={mockSavedQuery} canManage={true} />
    );
    await render();

    const picker = container.querySelector<HTMLButtonElement>(
      '[data-testid="account-picker"]'
    );
    mocks.pickerSelection = ["user:new@x.com"];
    await act(async () => {
      picker?.click();
    });
    const invite = container.querySelector<HTMLElement>(
      '[data-testid="grant-invite"]'
    );
    await clickLevelItem(invite as HTMLElement, SavedQueryBinding_Level.EDITOR);

    // Abandon the compose session by clearing the selection, then stage again:
    // the level must be back at the safe default.
    mocks.pickerSelection = [];
    await act(async () => {
      picker?.click();
    });
    mocks.pickerSelection = ["user:new@x.com"];
    await act(async () => {
      picker?.click();
    });
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('[data-testid="grant-add"]')
        ?.click();
    });
    expect(writtenBindings()).toContainEqual({
      level: SavedQueryBinding_Level.VIEWER,
      members: ["user:viewer@x.com", "group:team@x.com", "user:new@x.com"],
    });
    unmount();
  });
});
