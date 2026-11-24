import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  TIMESTAMP_COLUMN_MIN_WIDTH,
  TIMESTAMP_COLUMN_WIDTH,
} from "@/components/timestampColumn";
import { distributeColumnWidths } from "@/hooks/useColumnWidths";
import { useAppStore } from "@/stores/app";
import {
  shownTimestampInstants,
  shownTimestampModes,
  shownTimestampTruncates,
} from "@/test-utils/humanizeTs";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { getAccessGrantDisplayStatusText } from "@/utils/accessGrant";

vi.mock("@/hooks/useMediaQuery", () => ({
  useMediaQuery: () => false,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

import {
  AccessGrantRow,
  grantColumns,
  ProjectAccessGrantsPage,
} from "./ProjectAccessGrantsPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

afterEach(cleanup);

describe("grantColumns", () => {
  const columns = grantColumns((key) => key);
  const at = (key: string) => columns.findIndex((column) => column.key === key);
  const preferred = columns.reduce((sum, c) => sum + c.defaultWidth, 0);
  const floors = columns.reduce(
    (sum, c) => sum + (c.resizable === false ? c.defaultWidth : c.minWidth),
    0
  );
  const full = (widths: number[], key: string) =>
    widths[at(key)] === columns[at(key)].defaultWidth;
  const floored = (widths: number[], key: string) =>
    widths[at(key)] === columns[at(key)].minWidth;

  test("reads status, creator, databases, statement, expiry, creation, then actions", () => {
    expect(columns.map((column) => column.key)).toEqual([
      "status",
      "creator",
      "databases",
      "statement",
      "expiration",
      "created",
      "actions",
    ]);
  });

  test("opens both dates whole once the table holds every column", () => {
    for (const width of [preferred, preferred + 300]) {
      const widths = distributeColumnWidths(columns, width);
      expect(widths[at("created")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
      expect(widths[at("expiration")]).toBe(TIMESTAMP_COLUMN_WIDTH.operational);
    }
  });

  test("gives way dates first, then statement, then creator, then databases", () => {
    for (let width = floors; width < preferred; width++) {
      const widths = distributeColumnWidths(columns, width);
      expect(widths.reduce((sum, w) => sum + w, 0)).toBe(width);
      columns.forEach((column, i) => {
        expect(widths[i]).toBeGreaterThanOrEqual(column.minWidth);
      });
      if (!full(widths, "statement")) {
        expect(floored(widths, "created") && floored(widths, "expiration")).toBe(
          true
        );
      }
      if (!full(widths, "creator")) {
        expect(floored(widths, "statement")).toBe(true);
      }
      if (!full(widths, "databases")) {
        expect(floored(widths, "creator")).toBe(true);
      }
      if (!full(widths, "status")) {
        expect(floored(widths, "databases")).toBe(true);
      }
    }
  });

  test("floors both dates at the timestamp minimum", () => {
    expect(columns[at("created")].minWidth).toBe(TIMESTAMP_COLUMN_MIN_WIDTH);
    expect(columns[at("expiration")].minWidth).toBe(TIMESTAMP_COLUMN_MIN_WIDTH);
  });

  test("keeps compact row actions at their usable width while other columns yield", () => {
    for (const width of [floors - 100, floors, preferred, preferred + 300]) {
      expect(distributeColumnWidths(columns, width)[at("actions")]).toBe(112);
    }
  });
});

describe("AccessGrantRow", () => {
  const roots: ReturnType<typeof createRoot>[] = [];
  afterEach(() => {
    for (const root of roots.splice(0)) {
      act(() => root.unmount());
    }
  });

  test("puts each cell under its own column", () => {
    const columns = grantColumns((key) => key);
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);
    const nowMs = Date.now();
    const createdMs = nowMs - 60_000;
    const expiresMs = nowMs + 86_400_000;
    act(() =>
      root.render(
        <table>
          <tbody>
            <AccessGrantRow
              columns={columns}
              grant={create(AccessGrantSchema, {
                name: "projects/p1/accessGrants/1",
                creator: "users/alice@example.com",
                status: AccessGrant_Status.ACTIVE,
                query: "SELECT secret FROM vault",
                createTime: timestampFromMs(createdMs),
                expiration: {
                  case: "expireTime",
                  value: timestampFromMs(expiresMs),
                },
              })}
              canActivate={false}
              canRevoke={true}
              onActivate={() => {}}
              onRevoke={() => {}}
            />
          </tbody>
        </table>
      )
    );

    const cells = Array.from(container.querySelectorAll("td"));
    const cell = (key: string) =>
      cells[columns.findIndex((column) => column.key === key)];
    expect(cells).toHaveLength(columns.length);
    expect(cell("status").textContent).toBe(
      getAccessGrantDisplayStatusText("ACTIVE")
    );
    expect(cell("creator").textContent).toBe("alice@example.com");
    expect(cell("databases").textContent).toBe("-");
    expect(cell("statement").textContent).toBe("SELECT secret FROM vault");
    expect(shownTimestampInstants(cell("expiration"))).toEqual([
      String(expiresMs),
    ]);
    expect(shownTimestampInstants(cell("created"))).toEqual([
      String(createdMs),
    ]);
    expect(
      within(cell("actions")).getByRole("button", { name: "common.more" })
    ).toBeInstanceOf(HTMLButtonElement);
    expect(cell("actions").textContent).not.toContain("sql-editor.revoke-access");
  });

  test("dates a grant's creation and expiry in one form, each fitting its box", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);
    act(() =>
      root.render(
        <table>
          <tbody>
            <AccessGrantRow
              columns={grantColumns((key) => key)}
              grant={create(AccessGrantSchema, {
                name: "projects/p1/accessGrants/1",
                createTime: timestampFromMs(1_000),
                expiration: {
                  case: "expireTime",
                  value: timestampFromMs(2_000),
                },
              })}
              canActivate={false}
              canRevoke={false}
              onActivate={() => {}}
              onRevoke={() => {}}
            />
          </tbody>
        </table>
      )
    );

    expect(shownTimestampModes(container)).toEqual([
      "operational",
      "operational",
    ]);
    expect(shownTimestampTruncates(container)).toEqual([true, true]);
  });
});

describe("AccessGrantRow actions", () => {
  function renderRow(
    overrides: Partial<ComponentProps<typeof AccessGrantRow>> = {}
  ) {
    const props: ComponentProps<typeof AccessGrantRow> = {
      columns: grantColumns((key) => key),
      grant: create(AccessGrantSchema, {
        name: "projects/p1/accessGrants/1",
        creator: "users/alice@example.com",
        status: AccessGrant_Status.ACTIVE,
        query: "SELECT id FROM orders",
        issue: "projects/p1/issues/42",
        expiration: {
          case: "expireTime",
          value: timestampFromMs(Date.now() + 86_400_000),
        },
      }),
      canActivate: true,
      canRevoke: true,
      onActivate: vi.fn(),
      onRevoke: vi.fn(),
      ...overrides,
    };
    const view = render(
      <table>
        <tbody>
          <AccessGrantRow {...props} />
        </tbody>
      </table>
    );
    return {
      ...props,
      rerender: (next: Partial<ComponentProps<typeof AccessGrantRow>>) =>
        view.rerender(
          <table>
            <tbody>
              <AccessGrantRow {...props} {...next} />
            </tbody>
          </table>
        ),
    };
  }

  async function openConfirmation(label: string) {
    fireEvent.click(screen.getByRole("button", { name: "common.more" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: label }));
    return screen.findByRole("dialog", { name: label });
  }

  test("keeps issue navigation visible as a labeled icon link", () => {
    renderRow();
    const link = screen.getByRole("link", { name: "sql-editor.view-issue" });
    expect(link).toHaveAttribute("href", "/projects/p1/issues/42");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link.querySelector("svg")).toBeInTheDocument();
    expect(link.textContent).toBe("");
    expect(screen.queryByRole("menuitem")).not.toBeInTheDocument();
  });

  test.each([
    { status: AccessGrant_Status.ACTIVE, action: "revoke", allowed: true },
    { status: AccessGrant_Status.REVOKED, action: "activate", allowed: true },
    { status: AccessGrant_Status.ACTIVE, action: "revoke", allowed: false },
    { status: AccessGrant_Status.REVOKED, action: "activate", allowed: false },
    { status: AccessGrant_Status.PENDING, action: "none", allowed: false },
  ])(
    "offers only the permitted $action action for status $status (allowed=$allowed)",
    async ({ status, action, allowed }) => {
      const props = renderRow({
        grant: create(AccessGrantSchema, {
          name: "projects/p1/accessGrants/1",
          status,
        }),
        canActivate: action === "none" || (action === "activate" && allowed),
        canRevoke: action === "none" || (action === "revoke" && allowed),
      });
      expect(screen.queryByRole("link")).not.toBeInTheDocument();
      if (!allowed) {
        expect(
          screen.queryByRole("button", { name: "common.more" })
        ).not.toBeInTheDocument();
        return;
      }
      fireEvent.click(screen.getByRole("button", { name: "common.more" }));
      expect(await screen.findAllByRole("menuitem")).toHaveLength(1);
      expect(
        screen.getByRole("menuitem", { name: `sql-editor.${action}-access` })
      ).toBeInTheDocument();
      expect(props.onActivate).not.toHaveBeenCalled();
      expect(props.onRevoke).not.toHaveBeenCalled();
    }
  );

  test.each([
    { status: AccessGrant_Status.ACTIVE, offsetMs: -60_000 },
    { status: AccessGrant_Status.REVOKED, offsetMs: -60_000 },
    { status: AccessGrant_Status.ACTIVE, offsetMs: 0 },
    { status: AccessGrant_Status.REVOKED, offsetMs: 0 },
  ])(
    "hides state-changing actions at expiration ($status, $offsetMs ms)",
    ({ status, offsetMs }) => {
      const nowMs = Date.now();
      const clock = vi.spyOn(Date, "now").mockReturnValue(nowMs);
      try {
        renderRow({
          grant: create(AccessGrantSchema, {
            name: "projects/p1/accessGrants/1",
            status,
            issue: "projects/p1/issues/42",
            expiration: {
              case: "expireTime",
              value: timestampFromMs(nowMs + offsetMs),
            },
          }),
        });
        expect(
          screen.queryByRole("button", { name: "common.more" })
        ).not.toBeInTheDocument();
        expect(
          screen.getByRole("link", { name: "sql-editor.view-issue" })
        ).toBeInTheDocument();
      } finally {
        clock.mockRestore();
      }
    }
  );

  test("dismisses activation confirmation when a revoked grant expires", async () => {
    let nowMs = Date.now();
    const deadlineMs = nowMs + 250;
    const clock = vi.spyOn(Date, "now").mockImplementation(() => nowMs);
    try {
      const props = renderRow({
        grant: create(AccessGrantSchema, {
          name: "projects/p1/accessGrants/1",
          status: AccessGrant_Status.REVOKED,
          expiration: {
            case: "expireTime",
            value: timestampFromMs(deadlineMs),
          },
        }),
      });
      const popup = await openConfirmation("sql-editor.activate-access");
      expect(popup).toBeVisible();
      nowMs = deadlineMs;
      await waitFor(
        () => expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
        { timeout: 3_000 }
      );
      expect(
        screen.queryByRole("button", { name: "common.more" })
      ).not.toBeInTheDocument();
      expect(props.onActivate).not.toHaveBeenCalled();
    } finally {
      cleanup();
      clock.mockRestore();
    }
  });

  test("does not activate after expiration even before the deadline timer fires", async () => {
    let nowMs = Date.now();
    const deadlineMs = nowMs + 60_000;
    const clock = vi.spyOn(Date, "now").mockImplementation(() => nowMs);
    try {
      const props = renderRow({
        grant: create(AccessGrantSchema, {
          name: "projects/p1/accessGrants/1",
          status: AccessGrant_Status.REVOKED,
          expiration: {
            case: "expireTime",
            value: timestampFromMs(deadlineMs),
          },
        }),
      });
      const popup = await openConfirmation("sql-editor.activate-access");
      nowMs = deadlineMs;
      fireEvent.click(
        within(popup).getByRole("button", { name: "sql-editor.activate-access" })
      );
      expect(props.onActivate).not.toHaveBeenCalled();
    } finally {
      cleanup();
      clock.mockRestore();
    }
  });

  test.each([
    { status: AccessGrant_Status.ACTIVE, action: "revoke" },
    { status: AccessGrant_Status.REVOKED, action: "activate" },
  ])("requires explicit confirmation to $action", async ({ status, action }) => {
    const props = renderRow({
      grant: create(AccessGrantSchema, {
        name: "projects/p1/accessGrants/1",
        status,
      }),
    });
    const label = `sql-editor.${action}-access`;
    const popup = await openConfirmation(label);
    expect(popup).not.toHaveAttribute("aria-modal", "true");
    expect(within(popup).getByText(`sql-editor.${action}-confirm`)).toBeVisible();
    expect(props.onActivate).not.toHaveBeenCalled();
    expect(props.onRevoke).not.toHaveBeenCalled();
    fireEvent.click(within(popup).getByRole("button", { name: "common.cancel" }));
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(props.onActivate).not.toHaveBeenCalled();
    expect(props.onRevoke).not.toHaveBeenCalled();

    const confirmation = await openConfirmation(label);
    fireEvent.click(within(confirmation).getByRole("button", { name: label }));
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(
      action === "revoke" ? props.onRevoke : props.onActivate
    ).toHaveBeenCalledTimes(1);
    expect(
      action === "revoke" ? props.onActivate : props.onRevoke
    ).not.toHaveBeenCalled();
  });

  test("dismisses confirmation with Escape without changing the grant", async () => {
    const props = renderRow();
    const popup = await openConfirmation("sql-editor.revoke-access");
    fireEvent.keyDown(popup, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(props.onRevoke).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "common.more" })).toHaveFocus()
    );
  });

  test("dismisses confirmation when clicking outside without changing the grant", async () => {
    const props = renderRow();
    render(<button type="button">Outside</button>);
    await openConfirmation("sql-editor.revoke-access");
    const outside = screen.getByRole("button", { name: "Outside" });
    fireEvent.pointerDown(outside, { button: 0, pointerType: "mouse" });
    fireEvent.mouseDown(outside, { button: 0 });
    fireEvent.mouseUp(outside, { button: 0 });
    fireEvent.click(outside);
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(props.onRevoke).not.toHaveBeenCalled();
  });

  test.each(["permission", "status"])(
    "discards confirmation after the grant loses its eligible %s",
    async (changed) => {
      const props = renderRow();
      await openConfirmation("sql-editor.revoke-access");
      props.rerender(
        changed === "permission"
          ? { canRevoke: false }
          : { grant: { ...props.grant, status: AccessGrant_Status.PENDING } }
      );
      await waitFor(() =>
        expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
      );
      expect(props.onRevoke).not.toHaveBeenCalled();
      props.rerender({});
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      expect(props.onRevoke).not.toHaveBeenCalled();
    }
  );

  test("prevents repeat submission while a write is pending", async () => {
    let resolveWrite!: () => void;
    const onRevoke = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveWrite = resolve;
        })
    );
    renderRow({ onRevoke });
    const popup = await openConfirmation("sql-editor.revoke-access");
    const confirm = within(popup).getByRole("button", {
      name: "sql-editor.revoke-access",
    });
    fireEvent.click(confirm);
    await waitFor(() => expect(onRevoke).toHaveBeenCalledTimes(1));
    expect(confirm).toBeDisabled();
    expect(
      within(popup).getByRole("button", { name: "common.cancel" })
    ).toBeDisabled();
    fireEvent.click(confirm);
    expect(onRevoke).toHaveBeenCalledTimes(1);
    expect(popup).toBeVisible();
    await act(async () => resolveWrite());
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
  });

  test("retains a failed confirmation and permits retry", async () => {
    const onRevoke = vi.fn()
      .mockRejectedValueOnce(new Error("Request failed"))
      .mockResolvedValueOnce(undefined);
    renderRow({ onRevoke });
    const popup = await openConfirmation("sql-editor.revoke-access");
    fireEvent.click(
      within(popup).getByRole("button", { name: "sql-editor.revoke-access" })
    );
    expect(await within(popup).findByRole("alert")).toHaveTextContent(
      "Request failed"
    );
    expect(popup).toBeVisible();
    const retry = within(popup).getByRole("button", {
      name: "sql-editor.revoke-access",
    });
    expect(retry).toBeEnabled();
    fireEvent.click(retry);
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(onRevoke).toHaveBeenCalledTimes(2);
  });
});

describe("ProjectAccessGrantsPage refresh", () => {
  let originalState: ReturnType<typeof useAppStore.getState>;

  beforeEach(() => {
    originalState = useAppStore.getState();
    localStorage.removeItem(
      "bb.paged-table.project-projects/p1-access-grants.alice@example.com"
    );
    const project = create(ProjectSchema, {
      name: "projects/p1",
      title: "Example project",
    });
    const user = create(UserSchema, {
      name: "users/alice@example.com",
      email: "alice@example.com",
    });
    useAppStore.setState({
      projectsByName: { [project.name]: project },
      currentUser: user,
      loadCurrentUser: async () => user,
      loadServerInfo: async () => undefined,
      loadSubscription: async () => undefined,
      loadWorkspacePermissionState: async () => {},
      loadProjectIamPolicy: async () => undefined,
      hasProjectPermission: () => true,
      hasInstanceFeature: () => true,
    });
  });

  afterEach(() => {
    cleanup();
    useAppStore.setState(originalState, true);
  });

  test("preserves the row action anchor during the refresh after confirmation", async () => {
    const grant = create(AccessGrantSchema, {
      name: "projects/p1/accessGrants/1",
      creator: "users/alice@example.com",
      status: AccessGrant_Status.ACTIVE,
      query: "SELECT id FROM orders",
    });
    const revokedGrant = { ...grant, status: AccessGrant_Status.REVOKED };
    type ListResult = Awaited<
      ReturnType<ReturnType<typeof useAppStore.getState>["listAccessGrants"]>
    >;
    let resolveRefresh!: (result: ListResult) => void;
    const listAccessGrants = vi
      .fn()
      .mockResolvedValueOnce({ accessGrants: [grant], nextPageToken: "" })
      .mockImplementationOnce(
        () =>
          new Promise<ListResult>((resolve) => {
            resolveRefresh = resolve;
          })
      );
    const revokeAccessGrant = vi.fn(async () => revokedGrant);
    useAppStore.setState({ listAccessGrants, revokeAccessGrant });
    render(<ProjectAccessGrantsPage projectId="p1" />);
    const trigger = await screen.findByRole("button", { name: "common.more" });
    const table = screen.getByRole("table");
    await act(async () => trigger.focus());
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    const revokeItem = await screen.findByRole("menuitem", {
      name: "sql-editor.revoke-access",
    });
    await waitFor(() => expect(revokeItem).toHaveFocus());
    fireEvent.keyDown(revokeItem, { key: "Enter" });
    fireEvent.keyUp(revokeItem, { key: "Enter" });
    const popup = await screen.findByRole("dialog", {
      name: "sql-editor.revoke-access",
    });
    fireEvent.click(
      within(popup).getByRole("button", { name: "sql-editor.revoke-access" })
    );
    await waitFor(() => expect(listAccessGrants).toHaveBeenCalledTimes(2));
    expect(revokeAccessGrant).toHaveBeenCalledWith(grant.name);
    expect(table).toBeInTheDocument();
    expect(trigger).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "common.more" })).toBe(trigger);
    await waitFor(() => expect(trigger).toHaveFocus());
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await act(async () =>
      resolveRefresh({ accessGrants: [revokedGrant], nextPageToken: "" })
    );
    expect(screen.getByRole("table")).toBe(table);
    expect(screen.getByRole("button", { name: "common.more" })).toBe(trigger);
  });

  test.each(["search", "status", "sort", "page size", "project"])(
    "hides stale rows while a new %s query is loading",
    async (changed) => {
      const grant = create(AccessGrantSchema, {
        name: "projects/p1/accessGrants/1",
        creator: "users/alice@example.com",
        status: AccessGrant_Status.ACTIVE,
        query: "SELECT id FROM orders",
      });
      type ListResult = Awaited<
        ReturnType<ReturnType<typeof useAppStore.getState>["listAccessGrants"]>
      >;
      let resolveNext!: (result: ListResult) => void;
      const listAccessGrants = vi
        .fn()
        .mockResolvedValueOnce({
          accessGrants: [grant],
          nextPageToken: "old-token",
        })
        .mockImplementationOnce(
          () =>
            new Promise<ListResult>((resolve) => {
              resolveNext = resolve;
            })
        );
      useAppStore.setState({ listAccessGrants });
      const view = render(<ProjectAccessGrantsPage projectId="p1" />);
      const trigger = await screen.findByRole("button", {
        name: "common.more",
      });
      const table = screen.getByRole("table");

      if (changed === "search") {
        fireEvent.change(
          screen.getByPlaceholderText("issue.advanced-search.filter"),
          { target: { value: "other query" } }
        );
      } else if (changed === "status") {
        const input = screen.getByPlaceholderText("issue.advanced-search.filter");
        fireEvent.change(input, { target: { value: "status:" } });
        fireEvent.click(screen.getByText("common.pending"));
      } else if (changed === "sort") {
        fireEvent.click(
          screen.getByRole("columnheader", { name: "common.creator" })
        );
      } else if (changed === "page size") {
        fireEvent.click(screen.getByRole("combobox"));
        const option = await screen.findByRole("option", {
          name: "100",
        });
        await act(async () => option.focus());
        fireEvent.keyDown(option, { key: "Enter" });
        fireEvent.keyUp(option, { key: "Enter" });
      } else {
        const project = create(ProjectSchema, {
          name: "projects/p2",
          title: "Other project",
        });
        await act(async () =>
          useAppStore.setState((state) => ({
            projectsByName: { ...state.projectsByName, [project.name]: project },
          }))
        );
        view.rerender(<ProjectAccessGrantsPage projectId="p2" />);
      }

      await waitFor(() => expect(listAccessGrants).toHaveBeenCalledTimes(2));
      const request = listAccessGrants.mock.lastCall?.[0];
      expect(request.pageToken).toBeUndefined();
      if (changed === "search")
        expect(request.filter.statement).toBe("other query");
      if (changed === "status") expect(request.filter.status).toEqual(["PENDING"]);
      if (changed === "sort") expect(request.orderBy).toBe("creator desc");
      if (changed === "page size") expect(request.pageSize).toBe(100);
      if (changed === "project") expect(request.parent).toBe("projects/p2");
      expect(table).not.toBeInTheDocument();
      expect(trigger).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "common.more" })
      ).not.toBeInTheDocument();
      expect(screen.getByText("common.loading")).toBeVisible();
      await act(async () => resolveNext({ accessGrants: [], nextPageToken: "" }));
      expect(await screen.findByText("common.no-data")).toBeVisible();
    }
  );

  test("refreshes the current query when an earlier query's action completes", async () => {
    const grant = create(AccessGrantSchema, {
      name: "projects/p1/accessGrants/1",
      creator: "users/alice@example.com",
      status: AccessGrant_Status.ACTIVE,
      query: "SELECT id FROM orders",
    });
    let resolveWrite!: (value: typeof grant) => void;
    const revokeAccessGrant = vi.fn(
      () =>
        new Promise<typeof grant>((resolve) => {
          resolveWrite = resolve;
        })
    );
    const listAccessGrants = vi
      .fn()
      .mockResolvedValueOnce({ accessGrants: [grant], nextPageToken: "" })
      .mockResolvedValue({ accessGrants: [], nextPageToken: "" });
    useAppStore.setState({ listAccessGrants, revokeAccessGrant });
    render(<ProjectAccessGrantsPage projectId="p1" />);
    const trigger = await screen.findByRole("button", { name: "common.more" });
    await act(async () => trigger.focus());
    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    const item = await screen.findByRole("menuitem", {
      name: "sql-editor.revoke-access",
    });
    await waitFor(() => expect(item).toHaveFocus());
    fireEvent.keyDown(item, { key: "Enter" });
    fireEvent.keyUp(item, { key: "Enter" });
    const popup = await screen.findByRole("dialog", {
      name: "sql-editor.revoke-access",
    });
    fireEvent.click(
      within(popup).getByRole("button", { name: "sql-editor.revoke-access" })
    );
    await waitFor(() => expect(revokeAccessGrant).toHaveBeenCalledTimes(1));
    fireEvent.change(
      screen.getByPlaceholderText("issue.advanced-search.filter"),
      { target: { value: "other query" } }
    );
    await waitFor(() => expect(listAccessGrants).toHaveBeenCalledTimes(2));
    expect(listAccessGrants.mock.lastCall?.[0].filter.statement).toBe("other query");
    await act(async () =>
      resolveWrite({ ...grant, status: AccessGrant_Status.REVOKED })
    );
    await waitFor(() => expect(listAccessGrants).toHaveBeenCalledTimes(3));
    expect(listAccessGrants.mock.lastCall?.[0].filter.statement).toBe("other query");
  });

  test("shows loading rather than an empty state before the initial response", async () => {
    let resolveList!: (result: {
      accessGrants: [];
      nextPageToken: string;
    }) => void;
    useAppStore.setState({
      listAccessGrants: () =>
        new Promise((resolve) => {
          resolveList = resolve;
        }),
    });
    render(<ProjectAccessGrantsPage projectId="p1" />);
    expect(await screen.findByText("common.loading")).toBeVisible();
    expect(screen.queryByText("common.no-data")).not.toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
    await act(async () => resolveList({ accessGrants: [], nextPageToken: "" }));
    expect(await screen.findByText("common.no-data")).toBeVisible();
  });
});
