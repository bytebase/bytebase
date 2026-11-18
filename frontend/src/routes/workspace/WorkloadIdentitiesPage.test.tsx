import type { ButtonHTMLAttributes, ReactElement, ReactNode } from "react";
import { act, cloneElement, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const { passThrough, mocks } = vi.hoisted(() => ({
  passThrough: ({ children }: { children?: ReactNode }) => children,
  mocks: {
    deleteWorkloadIdentity: vi.fn(),
    fetchWorkloadIdentity: vi.fn(),
    removeCache: vi.fn(),
    sheetProps: vi.fn(),
  },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string, values?: Record<string, string>) =>
      values
        ? `${key}:${values.owner ?? values.name ?? values.environment ?? ""}`
        : key,
  }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
}));

vi.mock("@/assets/git/github.svg", () => ({ default: "github.svg" }));
vi.mock("@/assets/git/gitlab.svg", () => ({ default: "gitlab.svg" }));

vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: () => createElement("span", {}, "created"),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageInfo: () => null,
  ProjectPageLayout: passThrough,
  ProjectPageToolbar: passThrough,
}));

vi.mock("@/components/WorkspacePageLayout", () => ({
  WorkspacePageInfo: ({ description }: { description: string }) =>
    createElement("p", {}, description),
  WorkspacePageLayout: passThrough,
  WorkspacePageToolbar: passThrough,
}));

vi.mock("@/components/WorkloadIdentityFlow/WorkloadIdentitySheet", () => ({
  WorkloadIdentitySheet: (props: { open: boolean }) => {
    mocks.sheetProps(props);
    return null;
  },
}));

vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({ open, children }: { open: boolean; children: ReactNode }) =>
    open ? createElement("div", { role: "alertdialog" }, children) : null,
  AlertDialogContent: passThrough,
  AlertDialogDescription: passThrough,
  AlertDialogFooter: passThrough,
  AlertDialogTitle: passThrough,
}));

vi.mock("@/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    appearance: _appearance,
    size: _size,
    variant: _variant,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    appearance?: string;
    size?: string;
    variant?: string;
  }) => createElement("button", { type: "button", ...props }, children),
}));

vi.mock("@/components/ui/copy-button", () => ({ CopyButton: () => null }));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: passThrough,
  DropdownMenuContent: ({ children }: { children: ReactNode }) =>
    createElement("div", { role: "menu" }, children),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactNode;
    onClick: () => void;
  }) => createElement("button", { role: "menuitem", onClick }, children),
  DropdownMenuSeparator: () => createElement("hr"),
  DropdownMenuTrigger: ({
    render,
    children,
  }: {
    render: ReactElement;
    children: ReactNode;
  }) => cloneElement(render, {}, children),
}));

vi.mock("@/components/ui/ellipsis-text", () => ({
  EllipsisText: ({ text }: { text: string }) =>
    createElement("span", { "data-text": text }, text),
}));

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: () => createElement("input", { type: "search" }),
}));

vi.mock("@/components/ui/select", () => ({
  Select: passThrough,
  SelectContent: () => null,
  SelectItem: () => null,
  SelectTrigger: () => null,
  SelectValue: () => null,
}));

vi.mock("@/components/ui/table", () => ({
  Table: ({ children }: { children: ReactNode }) =>
    createElement("table", {}, children),
  TableBody: ({ children }: { children: ReactNode }) =>
    createElement("tbody", {}, children),
  TableCell: ({ children }: { children: ReactNode }) =>
    createElement("td", {}, children),
  TableEmptyView: ({ children }: { children: ReactNode }) =>
    createElement("tr", {}, createElement("td", {}, children)),
  TableHead: ({ children }: { children: ReactNode }) =>
    createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) =>
    createElement("thead", {}, children),
  TableRow: ({ children }: { children: ReactNode }) =>
    createElement("tr", {}, children),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => children,
}));

vi.mock("@/hooks/usePagedData", async () => {
  const { create } = await import("@bufbuild/protobuf");
  const pb = await import("@/types/proto-es/v1/workload_identity_service_pb");
  const identity = (
    email: string,
    providerType: number,
    subjectPattern: string,
    issuerUrl = ""
  ) =>
    create(pb.WorkloadIdentitySchema, {
      name: `workloadIdentities/${email}`,
      email,
      title: email.split("@")[0],
      state: 1,
      workloadIdentityConfig: create(pb.WorkloadIdentityConfigSchema, {
        providerType,
        subjectPattern,
        issuerUrl,
      }),
    });
  const { GITHUB, GITLAB, OIDC } = pb.WorkloadIdentityConfig_ProviderType;
  const dataList = [
    identity(
      "deploy-ci@workload.bytebase.com",
      GITHUB,
      "repo:example-org/billing-api:environment:prod"
    ),
    identity("platform-ci@workload.bytebase.com", GITHUB, "repo:example-org/*"),
    identity(
      "mobile-ci@workload.bytebase.com",
      GITLAB,
      "project_path:example-group/mobile/app:ref_type:branch:ref:main"
    ),
    identity(
      "nightly@workload.bytebase.com",
      OIDC,
      "pipeline:nightly:*",
      "https://oidc.ci.example.com"
    ),
  ];
  return {
    PagedTableFooter: () => null,
    usePagedData: () => ({
      dataList,
      hasMore: false,
      isFetchingMore: false,
      isLoading: false,
      loadMore: vi.fn(),
      onPageSizeChange: vi.fn(),
      pageSize: 20,
      pageSizeOptions: [20],
      removeCache: mocks.removeCache,
      updateCache: vi.fn(),
    }),
  };
});

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/stores/app", () => {
  const state = {
    deleteWorkloadIdentity: mocks.deleteWorkloadIdentity,
    fetchWorkloadIdentity: mocks.fetchWorkloadIdentity,
    fetchWorkspaceIamPolicy: vi.fn(),
    listWorkloadIdentities: vi.fn(),
    loadProjectIamPolicy: vi.fn(),
    projectPoliciesByName: {},
    projectsByName: {},
    undeleteWorkloadIdentity: vi.fn(),
    workspacePolicy: {
      bindings: [
        {
          role: "roles/gitopsServiceAgent",
          members: ["workloadIdentity:deploy-ci@workload.bytebase.com"],
        },
      ],
    },
    workspaceResourceName: () => "workspaces/-",
  };
  return {
    useAppStore: Object.assign(
      <T,>(selector: (store: typeof state) => T) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("@/stores/app/workloadIdentity", () => ({
  ensureWorkloadIdentityFullName: (email: string) =>
    `workloadIdentities/${email}`,
}));

vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/types", () => ({
  getWorkloadIdentityNameInBinding: (email: string) =>
    `workloadIdentity:${email}`,
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
}));

vi.mock("@/utils/role", () => ({
  displayRoleTitle: (role: string) => role.replace("roles/", ""),
}));

vi.mock("@/utils/v1/iam", () => ({ isBindingPolicyExpired: () => false }));

import { WorkloadIdentitiesPage } from "./WorkloadIdentitiesPage";

describe("WorkloadIdentitiesPage", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  const rowText = (index: number) =>
    container.querySelectorAll("tbody tr")[index]?.textContent ?? "";

  beforeEach(async () => {
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    mocks.deleteWorkloadIdentity.mockResolvedValue(undefined);
    await act(async () => root.render(createElement(WorkloadIdentitiesPage)));
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  it("states each trust rule in the create form's words", () => {
    expect(rowText(0)).toContain("example-org/billing-api");
    expect(rowText(0)).toContain("account-list.trust-environment");
    expect(rowText(0)).toContain("gitopsServiceAgent");
    expect(rowText(1)).toContain(
      "Workload-identity.done.target-any-repository:example-org"
    );
    expect(rowText(1)).toContain("account-list.no-role");
    expect(rowText(2)).toContain("example-group/mobile/app");
    expect(rowText(2)).toContain("ref_type:branch:ref:main");
    expect(rowText(3)).toContain("oidc.ci.example.com");
    expect(rowText(3)).toContain("pipeline:nightly:*");
  });

  it("names the provider by the form's token source titles", () => {
    const labels = Array.from(
      container.querySelectorAll('[role="img"]'),
      (icon) => icon.getAttribute("aria-label")
    );
    expect(labels).toEqual([
      "workload-identity.token-source.github-title",
      "workload-identity.token-source.github-title",
      "workload-identity.token-source.gitlab-title",
      "workload-identity.token-source.oidc-title",
    ]);
  });

  it("offers Edit and Deactivate, and deactivates after naming the identity", async () => {
    const items = Array.from(
      container.querySelectorAll<HTMLButtonElement>(
        'tbody tr:first-child [role="menuitem"]'
      )
    );
    expect(items.map((item) => item.textContent)).toEqual([
      "common.edit",
      "common.deactivate",
    ]);

    await act(async () => items[1].click());
    expect(
      document.querySelector('[role="alertdialog"]')?.textContent
    ).toContain("account-list.deactivate-title:deploy-ci");

    const confirm = Array.from(
      document.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button')
    ).find((button) => button.textContent === "common.deactivate");
    await act(async () => confirm?.click());

    expect(mocks.deleteWorkloadIdentity).toHaveBeenCalledWith(
      "workloadIdentities/deploy-ci@workload.bytebase.com"
    );
    expect(mocks.removeCache).toHaveBeenCalledWith(
      expect.objectContaining({ email: "deploy-ci@workload.bytebase.com" })
    );
  });
});
