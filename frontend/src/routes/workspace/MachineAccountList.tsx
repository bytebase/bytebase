import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { EllipsisVertical } from "lucide-react";
import { type ReactNode, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { ProjectPageInfo } from "@/components/ProjectPageLayout";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { EllipsisText } from "@/components/ui/ellipsis-text";
import { SearchInput } from "@/components/ui/search-input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { TableCell, TableEmptyView, TableRow } from "@/components/ui/table";
import { WorkspacePageInfo } from "@/components/WorkspacePageLayout";
import { useAppStore } from "@/stores/app";
import type { AccountFilter } from "@/stores/app/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { getTimeForPbTimestampProtoEs } from "@/types/timestamp";
import { displayRoleTitle } from "@/utils/role";
import { isBindingPolicyExpired } from "@/utils/v1/iam";

// Shared by the service account and workload identity list pages, which show
// the same row anatomy over two account types.

export type AccountStateFilter = "active" | "inactive" | "all";

export const accountListRequest = (
  state: AccountStateFilter,
  query: string
): { showDeleted: boolean; filter: AccountFilter } => ({
  showDeleted: state !== "active",
  filter: { query, state: state === "inactive" ? State.DELETED : undefined },
});

// Whether an account belongs in the list as filtered, mirroring the server
// filter `accountListRequest` builds, so a local update cannot show a row the
// next fetch would not return.
export const accountMatchesList = (
  account: { title: string; email: string; state: State },
  state: AccountStateFilter,
  query: string
): boolean => {
  const deleted = account.state === State.DELETED;
  if (state === "active" && deleted) return false;
  if (state === "inactive" && !deleted) return false;
  const search = query.trim().toLowerCase();
  return (
    !search ||
    account.title.toLowerCase().includes(search) ||
    account.email.toLowerCase().includes(search)
  );
};

export function AccountPageInfo({
  isProject,
  description,
}: {
  readonly isProject: boolean;
  readonly description: string;
}) {
  const PageInfo = isProject ? ProjectPageInfo : WorkspacePageInfo;
  return <PageInfo description={description} />;
}

export function AccountListToolbar({
  query,
  onQueryChange,
  state,
  onStateChange,
  action,
}: {
  readonly query: string;
  readonly onQueryChange: (query: string) => void;
  readonly state: AccountStateFilter;
  readonly onStateChange: (state: AccountStateFilter) => void;
  readonly action: ReactNode;
}) {
  const { t } = useTranslation();
  const items = [
    { value: "active", label: t("common.active") },
    { value: "inactive", label: t("common.inactive") },
    { value: "all", label: t("common.all") },
  ];
  return (
    <div className="flex w-full flex-wrap items-center gap-2">
      <SearchInput
        wrapperClassName="w-full flex-none sm:w-80"
        placeholder={t("account-list.search-placeholder")}
        aria-label={t("account-list.search-placeholder")}
        value={query}
        onChange={(e) => onQueryChange(e.target.value)}
      />
      <Select
        items={items}
        value={state}
        onValueChange={(value) => {
          if (value) onStateChange(value as AccountStateFilter);
        }}
      >
        <SelectTrigger className="w-32" aria-label={t("common.state")}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <div className="ml-auto">{action}</div>
    </div>
  );
}

export function AccountNameCell({
  title,
  email,
  inactive,
  onOpen,
}: {
  readonly title: string;
  readonly email: string;
  readonly inactive: boolean;
  readonly onOpen?: () => void;
}) {
  const { t } = useTranslation();
  const name = title || email;
  return (
    <div className="flex min-w-0 flex-col">
      <div className="flex min-w-0 items-center gap-x-2">
        {onOpen ? (
          <Button
            appearance="link"
            className="h-auto min-w-0 justify-start p-0 font-medium text-main"
            onClick={onOpen}
          >
            <EllipsisText text={name} />
          </Button>
        ) : (
          <EllipsisText text={name} className="font-medium text-main" />
        )}
        {inactive && (
          <span className="shrink-0 text-xs text-control-light">
            {t("common.inactive")}
          </span>
        )}
      </div>
      <div className="flex min-w-0 items-center gap-x-1">
        <EllipsisText text={email} className="text-xs text-control-light" />
        <CopyButton content={email} />
      </div>
    </div>
  );
}

// Roles the account holds in the page's scope: the project IAM policy on a
// project page, the workspace policy on the workspace page.
export function useAccountRoles(
  projectName: string | undefined
): (member: string) => string[] {
  const policy = useAppStore((s) =>
    projectName ? s.projectPoliciesByName[projectName] : s.workspacePolicy
  );
  const loadProjectIamPolicy = useAppStore((s) => s.loadProjectIamPolicy);
  const fetchWorkspaceIamPolicy = useAppStore((s) => s.fetchWorkspaceIamPolicy);
  useEffect(() => {
    if (policy) return;
    if (projectName) {
      void loadProjectIamPolicy(projectName, true);
    } else {
      void fetchWorkspaceIamPolicy(true);
    }
  }, [policy, projectName, loadProjectIamPolicy, fetchWorkspaceIamPolicy]);

  return useMemo(() => {
    const rolesByMember = new Map<string, string[]>();
    for (const binding of policy?.bindings ?? []) {
      if (isBindingPolicyExpired(binding)) continue;
      for (const member of binding.members) {
        const roles = rolesByMember.get(member) ?? [];
        if (!roles.includes(binding.role)) roles.push(binding.role);
        rolesByMember.set(member, roles);
      }
    }
    return (member: string) => rolesByMember.get(member) ?? [];
  }, [policy]);
}

export function AccountRoleCell({ roles }: { readonly roles: string[] }) {
  const { t } = useTranslation();
  if (roles.length === 0) {
    return (
      <span className="whitespace-nowrap text-control-light">
        {t("account-list.no-role")}
      </span>
    );
  }
  return (
    <div className="flex min-w-60 flex-wrap gap-1">
      {roles.map((role) => (
        <Badge key={role} className="text-xs whitespace-nowrap">
          {displayRoleTitle(role)}
        </Badge>
      ))}
    </div>
  );
}

export function AccountCreatedCell({
  createTime,
}: {
  readonly createTime: Timestamp | undefined;
}) {
  if (!createTime) return <span className="text-control-light">-</span>;
  return (
    <HumanizeTs
      tsMs={getTimeForPbTimestampProtoEs(createTime)}
      className="whitespace-nowrap text-control-light"
    />
  );
}

export type AccountMenuItem = {
  readonly key: string;
  readonly label: string;
  readonly onSelect: () => void;
  readonly destructive?: boolean;
};

export function AccountRowMenu({
  label,
  items,
}: {
  readonly label: string;
  readonly items: AccountMenuItem[];
}) {
  const { t } = useTranslation();
  if (items.length === 0) return null;
  const regular = items.filter((item) => !item.destructive);
  const destructive = items.filter((item) => item.destructive);
  const render = (item: AccountMenuItem) => (
    <DropdownMenuItem
      key={item.key}
      className={item.destructive ? "text-error" : undefined}
      onClick={item.onSelect}
    >
      {item.label}
    </DropdownMenuItem>
  );
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            appearance="secondary"
            size="sm"
            aria-label={t("account-list.more-actions", { name: label })}
          />
        }
      >
        <EllipsisVertical className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        {regular.map(render)}
        {regular.length > 0 && destructive.length > 0 && (
          <DropdownMenuSeparator />
        )}
        {destructive.map(render)}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function DeactivateAccountDialog({
  account,
  pending,
  onCancel,
  onConfirm,
}: {
  readonly account: { title: string; email: string } | undefined;
  readonly pending: boolean;
  readonly onCancel: () => void;
  readonly onConfirm: () => void;
}) {
  const { t } = useTranslation();
  return (
    <AlertDialog
      open={!!account}
      onOpenChange={(open) => {
        if (!open && !pending) onCancel();
      }}
    >
      <AlertDialogContent className="max-w-lg">
        <AlertDialogTitle>
          {t("account-list.deactivate-title", {
            name: account?.title || account?.email,
          })}
        </AlertDialogTitle>
        <AlertDialogDescription>
          {t("account-list.deactivate-description", { email: account?.email })}
        </AlertDialogDescription>
        <AlertDialogFooter>
          <Button appearance="outline" disabled={pending} onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button variant="destructive" disabled={pending} onClick={onConfirm}>
            {t("common.deactivate")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// The page info above already says what the account type is for, and the
// toolbar already has Create, so the empty row only says the list is empty.
export function AccountEmptyRow({
  colSpan,
  filtered,
  emptyText,
  onClearFilters,
}: {
  readonly colSpan: number;
  readonly filtered: boolean;
  readonly emptyText: string;
  readonly onClearFilters: () => void;
}) {
  const { t } = useTranslation();
  return (
    <TableEmptyView colSpan={colSpan}>
      <div className="flex flex-col items-center gap-y-3 text-control-light">
        <p>{filtered ? t("account-list.nothing-matches") : emptyText}</p>
        {filtered && (
          <Button appearance="outline" size="sm" onClick={onClearFilters}>
            {t("common.clear-filters")}
          </Button>
        )}
      </div>
    </TableEmptyView>
  );
}

// Rows shaped like the loaded table, so the page does not jump when data
// arrives.
export function AccountSkeletonRows({ columns }: { readonly columns: number }) {
  return (
    <>
      {[0, 1, 2].map((row) => (
        <TableRow key={row} aria-hidden="true">
          {Array.from({ length: columns }, (_, column) => (
            <TableCell key={column}>
              <div
                className={
                  column === 0 ? "flex flex-col gap-y-1.5" : "flex items-center"
                }
              >
                <div className="h-4 w-3/4 max-w-40 animate-pulse rounded-xs bg-control-bg" />
                {column === 0 && (
                  <div className="h-3 w-1/2 max-w-56 animate-pulse rounded-xs bg-control-bg" />
                )}
              </div>
            </TableCell>
          ))}
        </TableRow>
      ))}
    </>
  );
}
