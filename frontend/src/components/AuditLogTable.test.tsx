import { create } from "@bufbuild/protobuf";
import { anyPack } from "@bufbuild/protobuf/wkt";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { StatusSchema } from "@/types/proto-es/google/rpc/status_pb";
import {
  AuditLog_Severity,
  AuditLogSchema,
} from "@/types/proto-es/v1/audit_log_service_pb";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  searchAuditLogs: vi.fn(),
  exportAuditLogs: vi.fn(),
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  pushNotification: vi.fn(),
  usePlanFeature: vi.fn(() => true),
  listUsers: vi.fn(
    async (): Promise<{
      users: Array<{ name: string; email: string; title: string }>;
    }> => ({ users: [] })
  ),
  listServiceAccounts: vi.fn(
    async (): Promise<{
      serviceAccounts: Array<{ name: string; email: string; title: string }>;
    }> => ({ serviceAccounts: [] })
  ),
  listWorkloadIdentities: vi.fn(
    async (): Promise<{
      workloadIdentities: Array<{ name: string; email: string; title: string }>;
    }> => ({ workloadIdentities: [] })
  ),
  scopeOptions: { value: undefined as unknown },
  onSearchParamsChange: { value: undefined as unknown },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/api", () => ({
  auditLogServiceClientConnect: {
    searchAuditLogs: mocks.searchAuditLogs,
    exportAuditLogs: mocks.exportAuditLogs,
  },
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      listUsers: mocks.listUsers,
      listServiceAccounts: mocks.listServiceAccounts,
      listWorkloadIdentities: mocks.listWorkloadIdentities,
      projectsByName: { "projects/project-a": {} },
      hasWorkspacePermission: () => true,
      hasProjectPermission: () => true,
    }),
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: mocks.usePlanFeature,
  useWorkspaceResourceName: () => "workspaces/default",
}));

vi.mock("@/stores/modules/v1/common", () => ({
  extractUserEmail: (name: string) => name.replace("users/", ""),
  getProjectIdPlanUidStageUidFromRolloutName: () => [
    "project",
    "plan",
    "stage",
  ],
  planNamePrefix: "plans/",
  projectNamePrefix: "projects/",
  serviceAccountNamePrefix: "serviceAccounts/",
  userNamePrefix: "users/",
  workloadIdentityNamePrefix: "workloadIdentities/",
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: ({
    scopeOptions,
    onParamsChange,
  }: {
    scopeOptions: unknown;
    onParamsChange: unknown;
  }) => {
    mocks.scopeOptions.value = scopeOptions;
    mocks.onSearchParamsChange.value = onParamsChange;
    return <div data-testid="advanced-search" />;
  },
}));

vi.mock("@/components/TimeRangePicker", () => ({
  TimeRangePicker: () => <div data-testid="time-range-picker" />,
}));

vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));

vi.mock("@/hooks/usePagedData", () => ({
  PagedTableFooter: () => <div data-testid="paged-table-footer" />,
}));

vi.mock("@/hooks/useSessionPageSize", () => ({
  getPageSizeOptions: () => [10],
  useSessionPageSize: () => [10, vi.fn()],
}));

vi.mock("@/api/methods", () => ({
  ALL_METHODS_WITH_AUDIT: [],
}));

vi.mock("@/utils", () => ({
  formatAbsoluteDateTime: () => "2026-04-27 00:00:00",
  getDefaultPagination: () => 1000,
  humanizeDurationV1: () => "0ms",
}));

vi.mock("@/types", () => ({
  getDateForPbTimestampProtoEs: () => new Date("2026-04-27T00:00:00Z"),
}));

globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let AuditLogTable: typeof import("./AuditLogTable").AuditLogTable;

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

beforeEach(async () => {
  vi.clearAllMocks();
  ({ AuditLogTable } = await import("./AuditLogTable"));
});

afterEach(() => {
  // Restore here, not in the test body: an assertion that throws before the
  // restore would leak fake timers into every later test in the file.
  vi.useRealTimers();
  document.body.innerHTML = "";
});

describe("AuditLogTable", () => {
  test("searches only the matching special-account kind for a prefixed actor", async () => {
    mocks.searchAuditLogs.mockResolvedValue({ auditLogs: [], nextPageToken: "" });
    mocks.listServiceAccounts.mockResolvedValue({
      serviceAccounts: [
        {
          name: "serviceAccounts/deploy@service.bytebase.com",
          email: "deploy@service.bytebase.com",
          title: "Deploy",
        },
      ],
    });
    mocks.listWorkloadIdentities.mockResolvedValue({
      workloadIdentities: [
        {
          name: "workloadIdentities/ci@workload.bytebase.com",
          email: "ci@workload.bytebase.com",
          title: "CI",
        },
      ],
    });

    const { render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/project-a" canExport={false} />
    );
    await render();

    const actorScope = (
      mocks.scopeOptions.value as Array<{
        id: string;
        onSearch?: (keyword: string) => Promise<Array<{ value: string }>>;
      }>
    ).find((scope) => scope.id === "actor");
    const serviceAccounts = await actorScope?.onSearch?.(
      "serviceAccounts/deploy"
    );

    expect(serviceAccounts).toEqual([
      expect.objectContaining({
        value: "serviceAccounts/deploy@service.bytebase.com",
      }),
    ]);
    expect(mocks.listUsers).not.toHaveBeenCalled();
    expect(mocks.listWorkloadIdentities).not.toHaveBeenCalled();

    const workloadIdentities = await actorScope?.onSearch?.(
      "workloadIdentities/ci"
    );
    expect(workloadIdentities).toEqual([
      expect.objectContaining({
        value: "workloadIdentities/ci@workload.bytebase.com",
      }),
    ]);
    expect(mocks.listUsers).not.toHaveBeenCalled();

    await act(async () => {
      (
        mocks.onSearchParamsChange.value as
          | ((params: {
              query: string;
              scopes: Array<{ id: string; value: string }>;
            }) => void)
          | undefined
      )?.({
        query: "",
        scopes: [
          {
            id: "actor",
            value: "serviceAccounts/deploy@service.bytebase.com",
          },
        ],
      });
      await Promise.resolve();
    });
    expect(mocks.searchAuditLogs).toHaveBeenLastCalledWith(
      expect.objectContaining({
        filter:
          '(actor == "serviceAccounts/deploy@service.bytebase.com" || actor == "users/deploy@service.bytebase.com")',
      })
    );

    unmount();
  });

  test("falls back to users for an unprefixed actor", async () => {
    mocks.searchAuditLogs.mockResolvedValue({ auditLogs: [], nextPageToken: "" });
    mocks.listUsers.mockResolvedValue({
      users: [
        {
          name: "users/alice@example.com",
          email: "alice@example.com",
          title: "Alice",
        },
      ],
    });

    const { render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/project-a" canExport={false} />
    );
    await render();

    const actorScope = (
      mocks.scopeOptions.value as Array<{
        id: string;
        onSearch?: (keyword: string) => Promise<Array<{ value: string }>>;
      }>
    ).find((scope) => scope.id === "actor");
    const users = await actorScope?.onSearch?.("alice");

    expect(users).toEqual([
      expect.objectContaining({ value: "users/alice@example.com" }),
    ]);
    expect(mocks.listServiceAccounts).not.toHaveBeenCalled();
    expect(mocks.listWorkloadIdentities).not.toHaveBeenCalled();

    unmount();
  });

  test("renders a special-account actor as its full principal name", async () => {
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          name: "auditLogs/1",
          method: "/bytebase.v1.SQLService/Query",
          actor: "serviceAccounts/deploy@service.bytebase.com",
        }),
      ],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );
    await render();
    await act(async () => {
      await Promise.resolve();
    });

    expect(container.textContent).toContain(
      "serviceAccounts/deploy@service.bytebase.com"
    );
    expect(container.querySelector("a")).toBeNull();

    unmount();
  });

  test("shows the full actor value on hover", async () => {
    vi.useFakeTimers();
    const actor = "serviceAccounts/terraform@example.com";
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          name: "auditLogs/1",
          method: "/bytebase.v1.SQLService/Query",
          actor,
        }),
      ],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );
    await render();
    await act(async () => {
      await Promise.resolve();
    });

    const trigger = [...container.querySelectorAll("span")].find(
      (element) => element.textContent === actor
    );
    expect(trigger).toBeInstanceOf(HTMLSpanElement);
    await act(async () => {
      trigger?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
    });

    expect(document.getElementById("bb-react-layer-overlay")?.textContent).toContain(actor);

    unmount();
  });

  test("renders status details with PermissionDeniedDetail", async () => {
    const permissionDeniedDetail = create(PermissionDeniedDetailSchema, {
      method: "/bytebase.v1.SQLService/Query",
      requiredPermissions: ["bb.sql.select"],
      resources: ["instances/prod/databases/app"],
    });
    const status = create(StatusSchema, {
      code: 7,
      message: "permission denied",
      details: [anyPack(PermissionDeniedDetailSchema, permissionDeniedDetail)],
    });
    const auditLog = create(AuditLogSchema, {
      name: "auditLogs/1",
      severity: AuditLog_Severity.ERROR,
      method: "/bytebase.v1.SQLService/Query",
      actor: "users/user@example.com",
      status,
    });
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [auditLog],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );

    await render();
    await act(async () => {
      await Promise.resolve();
    });

    expect(container.textContent).toContain("PermissionDeniedDetail");
    expect(container.textContent).toContain("bb.sql.select");

    unmount();
  });

  test("badges only the rows carrying MCP delegation provenance", async () => {
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          name: "auditLogs/1",
          method: "/bytebase.v1.SQLService/Query",
          actor: "users/agent@example.com",
          mcpDelegation: { correlationId: "corr-1" },
        }),
        create(AuditLogSchema, {
          name: "auditLogs/2",
          method: "/bytebase.v1.SQLService/Query",
          actor: "users/human@example.com",
        }),
      ],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );
    await render();
    await act(async () => {
      await Promise.resolve();
    });

    const rows = [...container.querySelectorAll("tbody tr")];
    expect(rows).toHaveLength(2);
    expect(rows[0].textContent).toContain("agent@example.com");
    expect(rows[0].textContent).toContain("audit-log.mcp.badge");
    expect(rows[1].textContent).toContain("human@example.com");
    expect(rows[1].textContent).not.toContain("audit-log.mcp.badge");

    unmount();
  });

  test("a delegation whose every field is empty still badges the row", async () => {
    // Presence of the message is the marker: a pre-grant legacy session — a
    // plain web-session token at /mcp — stores no scope, resource, or client
    // ID, and must still be badged.
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          name: "auditLogs/1",
          method: "/bytebase.v1.SQLService/Query",
          actor: "users/agent@example.com",
          mcpDelegation: {},
        }),
      ],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );
    await render();
    await act(async () => {
      await Promise.resolve();
    });

    expect(container.textContent).toContain("audit-log.mcp.badge");

    unmount();
  });

  // Reads the tooltip as label -> value pairs rather than flat text, so a
  // field rendered under the wrong label fails instead of passing on
  // substring presence.
  const openTooltipFields = async (container: HTMLElement) => {
    // The Tooltip trigger wraps the Badge, so both spans carry the same
    // textContent; the first in document order is the trigger. focusin
    // bubbles, so either would open the tooltip.
    const trigger = [...container.querySelectorAll("span")].find(
      (el) => el.textContent === "audit-log.mcp.badge"
    );
    expect(trigger).toBeInstanceOf(HTMLSpanElement);
    await act(async () => {
      trigger?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
    });
    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay).toBeInstanceOf(HTMLDivElement);
    const pairs: Record<string, string> = {};
    for (const row of overlay?.querySelectorAll("div") ?? []) {
      const spans = row.querySelectorAll(":scope > span");
      if (spans.length === 2) {
        pairs[spans[0].textContent ?? ""] = spans[1].textContent ?? "";
      }
    }
    return { overlayText: overlay?.textContent ?? "", pairs };
  };

  test("the badge tooltip pairs each grant field with its own label", async () => {
    vi.useFakeTimers();
    // The ordinary grant-backed session: every field populated, including the
    // consented scope that decides read-only vs read-write.
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          name: "auditLogs/1",
          method: "/bytebase.v1.SQLService/Query",
          actor: "users/agent@example.com",
          mcpDelegation: {
            clientId: "bb_oauth_client",
            correlationId: "8b1f0a1e-corr",
            resource: "https://example.com/mcp",
            scope: "mcp:read-write",
          },
        }),
      ],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );
    await render();
    await act(async () => {
      await Promise.resolve();
    });

    const { overlayText, pairs } = await openTooltipFields(container);
    expect(overlayText).toContain("audit-log.mcp.origin");
    expect(pairs).toEqual({
      "audit-log.mcp.correlation-id": "8b1f0a1e-corr",
      "common.scope": "mcp:read-write",
      "common.resource": "https://example.com/mcp",
      "audit-log.mcp.client-id": "bb_oauth_client",
    });

    unmount();
  });

  test("the tooltip omits the fields a scope-omitting grant left empty", async () => {
    vi.useFakeTimers();
    // A client that omitted `scope` at consent — a steady-state population,
    // since discovery deliberately does not advertise the vocabulary until
    // P1b enforces it. See DelegatedGrant in backend/common/context.go.
    mocks.searchAuditLogs.mockResolvedValue({
      auditLogs: [
        create(AuditLogSchema, {
          name: "auditLogs/1",
          method: "/bytebase.v1.SQLService/Query",
          actor: "users/agent@example.com",
          mcpDelegation: {
            correlationId: "8b1f0a1e-corr",
            resource: "https://example.com/mcp",
          },
        }),
      ],
      nextPageToken: "",
    });

    const { container, render, unmount } = renderIntoContainer(
      <AuditLogTable parent="projects/-" canExport={false} />
    );
    await render();
    await act(async () => {
      await Promise.resolve();
    });

    const { overlayText, pairs } = await openTooltipFields(container);
    expect(overlayText).toContain("audit-log.mcp.origin");
    expect(pairs).toEqual({
      "audit-log.mcp.correlation-id": "8b1f0a1e-corr",
      "common.resource": "https://example.com/mcp",
    });
    expect(overlayText).not.toContain("common.scope");
    expect(overlayText).not.toContain("audit-log.mcp.client-id");

    unmount();
  });
});
