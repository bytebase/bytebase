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
  listUsers: vi.fn(async () => ({ users: [] })),
}));

vi.mock("react-i18next", () => ({
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
    }),
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: mocks.usePlanFeature,
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
  userNamePrefix: "users/",
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="advanced-search" />,
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
      user: "users/user@example.com",
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
          user: "users/agent@example.com",
          mcpDelegation: { correlationId: "corr-1" },
        }),
        create(AuditLogSchema, {
          name: "auditLogs/2",
          method: "/bytebase.v1.SQLService/Query",
          user: "users/human@example.com",
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
          user: "users/agent@example.com",
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
          user: "users/agent@example.com",
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
          user: "users/agent@example.com",
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
