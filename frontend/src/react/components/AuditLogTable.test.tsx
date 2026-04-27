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
  useSubscriptionV1Store: vi.fn(() => ({ hasFeature: () => true })),
  useUserStore: vi.fn(() => ({ fetchUserList: vi.fn() })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/connect", () => ({
  auditLogServiceClientConnect: {
    searchAuditLogs: mocks.searchAuditLogs,
    exportAuditLogs: mocks.exportAuditLogs,
  },
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useSubscriptionV1Store: mocks.useSubscriptionV1Store,
  useUserStore: mocks.useUserStore,
}));

vi.mock("@/store/modules/v1/common", () => ({
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

vi.mock("@/react/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div data-testid="advanced-search" />,
}));

vi.mock("@/react/components/TimeRangePicker", () => ({
  TimeRangePicker: () => <div data-testid="time-range-picker" />,
}));

vi.mock("@/react/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));

vi.mock("@/react/hooks/usePagedData", () => ({
  PagedTableFooter: () => <div data-testid="paged-table-footer" />,
}));

vi.mock("@/react/hooks/useSessionPageSize", () => ({
  getPageSizeOptions: () => [10],
  useSessionPageSize: () => [10, vi.fn()],
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/connect/methods", () => ({
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
});
