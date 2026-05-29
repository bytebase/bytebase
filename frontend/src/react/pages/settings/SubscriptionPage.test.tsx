import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  exportVCSProviderUsers: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(),
  notify: vi.fn(),
}));

vi.mock("@/connect", () => ({
  subscriptionServiceClientConnect: {
    exportVCSProviderUsers: mocks.exportVCSProviderUsers,
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/components/InstanceAssignmentSheet", () => ({
  InstanceAssignmentSheet: () => null,
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/textarea", () => ({
  Textarea: (props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) => (
    <textarea {...props} />
  ),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useNotify: () => mocks.notify,
  useServerState: () => ({
    activeVcsUserCount: 2,
    activatedInstanceCount: 2,
    isSaaSMode: false,
    totalInstanceCount: 3,
    userCountInIam: 4,
    workspaceResourceName: "workspaces/workspace-id",
  }),
  useSubscriptionState: () => ({
    currentPlan: PlanType.ENTERPRISE,
    expireAt: "",
    hasUnifiedInstanceLicense: true,
    instanceCountLimit: 10,
    instanceLicenseCount: 10,
    isExpired: false,
    isFreePlan: false,
    isSelfHostLicense: true,
    isTrialing: false,
    showTrial: false,
    trialingDays: 14,
    uploadLicense: vi.fn(),
    userCountLimit: 20,
  }),
}));

vi.mock("@/types", () => ({
  ENTERPRISE_INQUIRE_LINK: "https://example.com",
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("./PurchaseSection", () => ({
  PurchaseSection: () => null,
}));

let SubscriptionPage: typeof import("./SubscriptionPage").SubscriptionPage;

describe("SubscriptionPage", () => {
  let container: HTMLDivElement;
  let root: Root;
  class TestBlob {
    readonly parts: BlobPart[];
    readonly type: string;

    constructor(parts: BlobPart[] = [], options: BlobPropertyBag = {}) {
      this.parts = parts;
      this.type = options.type ?? "";
    }
  }

  beforeEach(async () => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 4, 28, 17, 10, 11));
    mocks.hasWorkspacePermissionV2.mockReturnValue(true);
    mocks.exportVCSProviderUsers.mockResolvedValue({
      contentType: "text/csv; charset=utf-8",
      data: new TextEncoder().encode(
        "vcs_type,user_id,user_name,display_name,last_seen_at\ngithub,123,alice,Alice,2026-05-28T09:00:00Z\n"
      ),
    });
    vi.stubGlobal("Blob", TestBlob);
    URL.createObjectURL = vi.fn(() => "blob:vcs-users");
    URL.revokeObjectURL = vi.fn();
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    ({ SubscriptionPage } = await import("./SubscriptionPage"));
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  test("displays unified instance usage as current count over limit", () => {
    act(() => {
      root.render(<SubscriptionPage />);
    });

    expect(
      container.textContent?.includes(
        "subscription.instance-assignment.used-and-total-instance"
      )
    ).toBe(true);
    expect(container.textContent?.includes("3/10")).toBe(true);

    act(() => root.unmount());
    container.remove();
  });

  test("displays active VCS user usage separately from IAM user usage", () => {
    act(() => {
      root.render(<SubscriptionPage />);
    });

    expect(
      container.textContent?.includes("subscription.vcs-users.active")
    ).toBe(true);
    expect(container.textContent?.includes("2/20")).toBe(true);
    expect(container.textContent?.includes("4/20")).toBe(true);

    act(() => root.unmount());
    container.remove();
  });

  test("downloads active VCS users csv", async () => {
    await act(async () => {
      root.render(<SubscriptionPage />);
    });

    const button = container.querySelector(
      'button[aria-label="subscription.vcs-users.download"]'
    );
    expect(button).not.toBeNull();

    await act(async () => {
      button?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mocks.exportVCSProviderUsers).toHaveBeenCalledWith(
      {},
      expect.any(Object)
    );
    const createObjectURL = vi.mocked(URL.createObjectURL);
    expect(createObjectURL).toHaveBeenCalledTimes(1);
    const blob = createObjectURL.mock.calls[0][0] as unknown as TestBlob;
    expect(blob.parts).toEqual([
      "vcs_type,user_id,user_name,display_name,last_seen_at\ngithub,123,alice,Alice,2026-05-28T09:00:00Z\n",
    ]);
    expect(blob.type).toBe("text/csv; charset=utf-8");

    const click = vi.mocked(HTMLAnchorElement.prototype.click);
    expect(click).toHaveBeenCalledTimes(1);
    const anchor = click.mock.instances[0] as HTMLAnchorElement;
    expect(anchor.href).toBe("blob:vcs-users");
    expect(anchor.download).toBe("active-vcs-users.2026-05-28T17-10-11.csv");
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:vcs-users");

    act(() => root.unmount());
    container.remove();
  });

  test("hides active VCS users download without subscription permission", () => {
    mocks.hasWorkspacePermissionV2.mockImplementation(
      (permission) => permission !== "bb.subscription.manage"
    );

    act(() => {
      root.render(<SubscriptionPage />);
    });

    expect(
      container.querySelector(
        'button[aria-label="subscription.vcs-users.download"]'
      )
    ).toBeNull();
    expect(mocks.exportVCSProviderUsers).not.toHaveBeenCalled();

    act(() => root.unmount());
    container.remove();
  });

  test("notifies when active VCS users csv download fails", async () => {
    mocks.exportVCSProviderUsers.mockRejectedValue(new Error("boom"));

    await act(async () => {
      root.render(<SubscriptionPage />);
    });

    const button = container.querySelector(
      'button[aria-label="subscription.vcs-users.download"]'
    );
    await act(async () => {
      button?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mocks.notify).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "subscription.vcs-users.download-failure.title",
      description: "subscription.vcs-users.download-failure.description",
    });
    expect(URL.createObjectURL).not.toHaveBeenCalled();

    act(() => root.unmount());
    container.remove();
  });
});
