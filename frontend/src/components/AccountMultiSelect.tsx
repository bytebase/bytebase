import { Check, ChevronDown, KeyRound, Shield, Users, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import { LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { SearchInput } from "@/components/ui/search-input";
import { useCurrentUser } from "@/hooks/useAppState";
import { useClickOutside } from "@/hooks/useClickOutside";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import {
  extractServiceAccountId,
  extractUserEmail,
  extractWorkloadIdentityId,
  groupNamePrefix,
  projectNamePrefix,
  serviceAccountNamePrefix,
  workloadIdentityNamePrefix,
  workspaceNamePrefix,
} from "@/stores/modules/v1/common";
import {
  AccountType,
  ALL_USERS_USER_EMAIL,
  getAccountTypeByEmail,
  getAccountTypeByFullname,
} from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { getDefaultPagination, isValidEmail } from "@/utils";
import {
  convertFullnameToMember,
  convertMemberToFullname,
} from "@/utils/v1/iam";

import { getAvatarColor, getInitials } from "./UserAvatar";

// ---- Account type detection ----
//
// Routes typed input to the right account type.

function getSpecialAccountByFullname(fullname: string): {
  type: AccountType;
  email: string;
} | null {
  switch (getAccountTypeByFullname(fullname)) {
    case AccountType.SERVICE_ACCOUNT: {
      const email = extractServiceAccountId(fullname);
      return isValidEmail(email)
        ? { type: AccountType.SERVICE_ACCOUNT, email }
        : null;
    }
    case AccountType.WORKLOAD_IDENTITY: {
      const email = extractWorkloadIdentityId(fullname);
      return isValidEmail(email)
        ? { type: AccountType.WORKLOAD_IDENTITY, email }
        : null;
    }
    default:
      return null;
  }
}

function getSpecialAccountByEmail(email: string): {
  type: AccountType;
  email: string;
} | null {
  if (!isValidEmail(email)) return null;

  switch (getAccountTypeByEmail(email)) {
    case AccountType.SERVICE_ACCOUNT:
      return { type: AccountType.SERVICE_ACCOUNT, email };
    case AccountType.WORKLOAD_IDENTITY:
      return { type: AccountType.WORKLOAD_IDENTITY, email };
    default:
      return null;
  }
}

function detectSpecialAccount(input: string): {
  type: AccountType;
  email: string;
} | null {
  const email = input.trim();
  if (!email) return null;

  return getSpecialAccountByFullname(email) ?? getSpecialAccountByEmail(email);
}

type SpecialAccount = {
  type: AccountType;
  fullname: string;
  email: string;
  title?: string;
};

type AccountParent = {
  name: string;
  canListServiceAccounts: boolean;
  canListWorkloadIdentities: boolean;
};

// ---- Sub-components ----

function SelectionCheckbox({ selected }: { selected: boolean }) {
  return (
    <div
      className={cn(
        "size-4 rounded-xs border flex items-center justify-center shrink-0",
        selected
          ? "bg-accent border-accent text-accent-text"
          : "border-control-border"
      )}
    >
      {selected && <Check className="h-3 w-3" />}
    </div>
  );
}

function SpecialAccountOption({
  keyword,
  match,
  selected,
  onToggle,
}: {
  keyword: string;
  match: SpecialAccount;
  selected: boolean;
  onToggle: () => void;
}) {
  const { t } = useTranslation();
  const isServiceAccount = match.type === AccountType.SERVICE_ACCOUNT;
  const Icon = isServiceAccount ? KeyRound : Shield;
  const label = isServiceAccount
    ? t("settings.members.service-account")
    : t("settings.members.workload-identity");

  return (
    <div
      className={cn(
        "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-control-bg",
        selected && "bg-accent/5"
      )}
      onClick={onToggle}
    >
      <SelectionCheckbox selected={selected} />
      <div
        className="size-7 rounded-full flex items-center justify-center text-accent-text text-xs font-medium shrink-0"
        style={{ backgroundColor: getAvatarColor(match.email) }}
      >
        <Icon className="h-3.5 w-3.5" />
      </div>
      <div className="flex flex-col min-w-0">
        <div className="flex items-center gap-x-1">
          <HighlightLabelText
            text={match.title || match.email.split("@")[0]}
            keyword={keyword}
            className="text-sm font-medium truncate"
          />
          <span className="text-xs text-control-light bg-control-bg rounded-xs px-1">
            {label}
          </span>
        </div>
        <HighlightLabelText
          text={match.email}
          keyword={keyword}
          className="text-xs text-control-light truncate"
        />
      </div>
    </div>
  );
}

// ---- Component ----

export function AccountMultiSelect({
  value,
  onChange,
  disabled,
  includeAllUsers,
  excludeAccounts,
  accountParents,
  placeholder,
}: {
  value: string[];
  onChange: (value: string[]) => void;
  disabled?: boolean;
  includeAllUsers?: boolean;
  /** Empty-state text in the trigger; defaults to the generic account label. */
  placeholder?: string;
  /**
   * Binding strings ("user:{email}" / "group:{email}") to hide from the
   * dropdown — accounts that cannot or need not be picked here, such as the
   * caller themselves or accounts already holding a grant. Display-only:
   * chips already in `value` keep rendering and stay removable.
   */
  excludeAccounts?: string[];
  /** Resource parents whose special accounts may be discovered. */
  accountParents?: string[];
}) {
  const { t } = useTranslation();
  const listUsers = useAppStore((state) => state.listUsers);
  const listGroups = useAppStore((state) => state.listGroups);
  const listServiceAccounts = useAppStore((state) => state.listServiceAccounts);
  const listWorkloadIdentities = useAppStore(
    (state) => state.listWorkloadIdentities
  );
  const currentUser = useCurrentUser();

  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);
  const [specialAccounts, setSpecialAccounts] = useState<SpecialAccount[]>([]);

  const uniqueAccountParents = useMemo(
    () => [...new Set(accountParents?.filter(Boolean) ?? [])],
    [accountParents]
  );
  const workspaceAccountParent = useMemo(
    () =>
      uniqueAccountParents.find((parent) =>
        parent.startsWith(workspaceNamePrefix)
      ),
    [uniqueAccountParents]
  );
  const projectAccountParent = useMemo(
    () =>
      uniqueAccountParents.find((parent) =>
        parent.startsWith(projectNamePrefix)
      ),
    [uniqueAccountParents]
  );
  const project = useAppStore((state) =>
    projectAccountParent
      ? state.projectsByName[projectAccountParent]
      : undefined
  );
  const canListWorkspaceServiceAccounts = useAppStore((state) =>
    workspaceAccountParent
      ? state.hasWorkspacePermission("bb.serviceAccounts.list")
      : false
  );
  const canListWorkspaceWorkloadIdentities = useAppStore((state) =>
    workspaceAccountParent
      ? state.hasWorkspacePermission("bb.workloadIdentities.list")
      : false
  );
  const canListProjectServiceAccounts = useAppStore((state) =>
    project
      ? state.hasProjectPermission(project, "bb.serviceAccounts.list")
      : false
  );
  const canListProjectWorkloadIdentities = useAppStore((state) =>
    project
      ? state.hasProjectPermission(project, "bb.workloadIdentities.list")
      : false
  );
  const permittedAccountParents = useMemo((): AccountParent[] => {
    const parents: AccountParent[] = [];
    if (
      workspaceAccountParent &&
      (canListWorkspaceServiceAccounts || canListWorkspaceWorkloadIdentities)
    ) {
      parents.push({
        name: workspaceAccountParent,
        canListServiceAccounts: canListWorkspaceServiceAccounts,
        canListWorkloadIdentities: canListWorkspaceWorkloadIdentities,
      });
    }
    if (
      projectAccountParent &&
      project &&
      (canListProjectServiceAccounts || canListProjectWorkloadIdentities)
    ) {
      parents.push({
        name: projectAccountParent,
        canListServiceAccounts: canListProjectServiceAccounts,
        canListWorkloadIdentities: canListProjectWorkloadIdentities,
      });
    }
    return parents;
  }, [
    workspaceAccountParent,
    canListWorkspaceServiceAccounts,
    canListWorkspaceWorkloadIdentities,
    projectAccountParent,
    project,
    canListProjectServiceAccounts,
    canListProjectWorkloadIdentities,
  ]);

  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  // Cache display labels for selected items so they persist across search changes
  const labelCacheRef = useRef<Map<string, string>>(new Map());

  useEffect(() => {
    const query = search.trim();
    // Over-fetch by the exclusion count: filtering happens after server
    // pagination, so without this a widely-shared exclusion list could
    // consume the entire page and leave nothing pickable.
    const pageSize = getDefaultPagination() + (excludeAccounts?.length ?? 0);
    listUsers({ pageSize, filter: { query } }).then(({ users: fetched }) =>
      setUsers(fetched)
    );
    listGroups({ pageSize, filter: { query } }).then(({ groups: fetched }) =>
      setGroups(fetched)
    );
  }, [search, listUsers, listGroups, excludeAccounts]);

  useEffect(() => {
    if (permittedAccountParents.length === 0) {
      setSpecialAccounts([]);
      return;
    }

    let cancelled = false;
    const query = search.trim();
    const pageSize = getDefaultPagination() + (excludeAccounts?.length ?? 0);
    const params = (parent: string) => ({
      parent,
      pageSize,
      showDeleted: false,
      filter: { query },
      skipCache: true,
    });

    void Promise.allSettled(
      permittedAccountParents.flatMap((parent) => [
        ...(parent.canListServiceAccounts
          ? [listServiceAccounts(params(parent.name))]
          : []),
        ...(parent.canListWorkloadIdentities
          ? [listWorkloadIdentities(params(parent.name))]
          : []),
      ])
    ).then((results) => {
      if (cancelled) return;
      const discovered = new Map<string, SpecialAccount>();
      for (const result of results) {
        if (result.status !== "fulfilled") continue;
        if ("serviceAccounts" in result.value) {
          for (const account of result.value.serviceAccounts) {
            discovered.set(account.name, {
              type: AccountType.SERVICE_ACCOUNT,
              fullname: account.name,
              email: account.email,
              title: account.title,
            });
          }
        } else {
          for (const account of result.value.workloadIdentities) {
            discovered.set(account.name, {
              type: AccountType.WORKLOAD_IDENTITY,
              fullname: account.name,
              email: account.email,
              title: account.title,
            });
          }
        }
      }
      setSpecialAccounts([...discovered.values()]);
    });

    return () => {
      cancelled = true;
    };
  }, [
    search,
    permittedAccountParents,
    listServiceAccounts,
    listWorkloadIdentities,
    excludeAccounts,
  ]);

  const handleClickOutside = useCallback(() => {
    setOpen(false);
    setSearch("");
  }, []);
  useClickOutside(containerRef, open, handleClickOutside);

  // Selected fullnames set for quick lookup
  const selectedFullnames = useMemo(
    () => new Set(value.map(convertMemberToFullname)),
    [value]
  );

  const excludedFullnames = useMemo(
    () => new Set((excludeAccounts ?? []).map(convertMemberToFullname)),
    [excludeAccounts]
  );
  // Filter the rendered lists only — `users`/`groups` stay complete so
  // resolveLabel keeps labeling chips for any binding in `value`.
  const visibleUsers = useMemo(
    () => users.filter((user) => !excludedFullnames.has(`users/${user.email}`)),
    [users, excludedFullnames]
  );
  const visibleGroups = useMemo(
    () => groups.filter((group) => !excludedFullnames.has(group.name)),
    [groups, excludedFullnames]
  );
  const visibleSpecialAccounts = useMemo(
    () =>
      specialAccounts.filter(
        (account) => !excludedFullnames.has(account.fullname)
      ),
    [specialAccounts, excludedFullnames]
  );
  const visibleServiceAccounts = useMemo(
    () =>
      visibleSpecialAccounts.filter(
        (account) => account.type === AccountType.SERVICE_ACCOUNT
      ),
    [visibleSpecialAccounts]
  );
  const visibleWorkloadIdentities = useMemo(
    () =>
      visibleSpecialAccounts.filter(
        (account) => account.type === AccountType.WORKLOAD_IDENTITY
      ),
    [visibleSpecialAccounts]
  );

  const toggle = (fullname: string) => {
    if (disabled) return;
    const binding = convertFullnameToMember(fullname);
    if (selectedFullnames.has(fullname)) {
      labelCacheRef.current.delete(binding);
      onChange(value.filter((v) => v !== binding));
    } else {
      // Cache display label at selection time so it survives search changes
      const label = resolveLabel(fullname);
      if (label) labelCacheRef.current.set(binding, label);
      onChange([...value, binding]);
    }
  };

  const remove = (binding: string) => {
    if (disabled) return;
    labelCacheRef.current.delete(binding);
    onChange(value.filter((v) => v !== binding));
  };

  // Resolve display label from current search results
  const resolveLabel = (fullname: string): string | undefined => {
    if (fullname.startsWith("users/")) {
      const email = extractUserEmail(fullname);
      const user = users.find((u) => u.email === email);
      return user?.title || user?.email;
    }
    if (fullname.startsWith(groupNamePrefix)) {
      const group = groups.find((g) => g.name === fullname);
      return group?.title || fullname;
    }
    const specialAccount = specialAccounts.find(
      (account) => account.fullname === fullname
    );
    if (specialAccount) {
      return specialAccount.title || specialAccount.email;
    }
    return undefined;
  };

  // Detect service account / workload identity typed in search
  const specialAccountMatch = useMemo((): SpecialAccount | null => {
    const match = detectSpecialAccount(search);
    if (!match || !match.email) return null;
    const prefix =
      match.type === AccountType.SERVICE_ACCOUNT
        ? serviceAccountNamePrefix
        : workloadIdentityNamePrefix;
    return { ...match, fullname: `${prefix}${match.email}` };
  }, [search]);

  // Allow selecting arbitrary user emails typed in the search box
  // (for SaaS where admins grant access to emails before signup, or when
  // the user can set IAM but cannot list users/groups)
  const arbitraryEmailMatch = useMemo((): string | null => {
    const trimmed = search.trim();
    if (!trimmed || !isValidEmail(trimmed)) return null;
    // Don't show if it's a service account or workload identity
    if (specialAccountMatch) return null;
    // Don't show if it already matches a fetched user
    if (users.some((u) => u.email === trimmed)) return null;
    return trimmed;
  }, [search, specialAccountMatch, users]);

  const visibleSpecialAccountMatch =
    specialAccountMatch &&
    !excludedFullnames.has(specialAccountMatch.fullname) &&
    !visibleSpecialAccounts.some(
      (account) => account.fullname === specialAccountMatch.fullname
    )
      ? specialAccountMatch
      : null;
  const visibleArbitraryEmailMatch =
    arbitraryEmailMatch &&
    !excludedFullnames.has(`users/${arbitraryEmailMatch}`)
      ? arbitraryEmailMatch
      : null;
  const showAllUsers = includeAllUsers && !search.trim();

  // Label for a selected binding chip — uses cache to survive search changes
  const chipLabel = (binding: string): string => {
    if (binding === ALL_USERS_USER_EMAIL) {
      return t("settings.members.all-users");
    }
    const cached = labelCacheRef.current.get(binding);
    if (cached) return cached;
    const fullname = convertMemberToFullname(binding);
    const specialAccount = detectSpecialAccount(fullname);
    switch (specialAccount?.type) {
      case AccountType.SERVICE_ACCOUNT:
      case AccountType.WORKLOAD_IDENTITY:
        return specialAccount.email;
    }
    return resolveLabel(fullname) || binding;
  };

  return (
    <div
      ref={containerRef}
      className="relative"
      onKeyDown={(event) => {
        if (event.key !== "Escape" || !open) return;
        // Swallow Escape while the dropdown is open: it closes only the
        // dropdown, not the popover (and staged state) around the picker.
        event.stopPropagation();
        event.preventDefault();
        setOpen(false);
        setSearch("");
      }}
    >
      {/* Trigger */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          disabled && "opacity-50 cursor-not-allowed",
          open && "border-accent"
        )}
        onClick={() => {
          if (!disabled) {
            setOpen(!open);
            requestAnimationFrame(() => inputRef.current?.focus());
          }
        }}
      >
        {value.map((binding) => (
          <span
            key={binding}
            className="inline-flex items-center gap-x-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs"
          >
            {chipLabel(binding)}
            {!disabled && (
              <button
                type="button"
                className="hover:text-error"
                onClick={(e) => {
                  e.stopPropagation();
                  remove(binding);
                }}
              >
                <X className="h-3 w-3" />
              </button>
            )}
          </span>
        ))}
        {value.length === 0 && (
          <span className="text-control-placeholder">
            {placeholder ?? t("settings.members.select-account", { count: 2 })}
          </span>
        )}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {/* Dropdown */}
      {open && (
        <div
          className={cn(
            "absolute mt-1 w-full bg-background border border-control-border rounded-sm shadow-lg max-h-72 overflow-auto flex flex-col",
            LAYER_SURFACE_CLASS
          )}
        >
          {/* Search input */}
          <SearchInput
            ref={inputRef}
            wrapperClassName="sticky top-0 bg-background m-2"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("common.search-for-more")}
          />

          <div className="overflow-auto" role="listbox" aria-multiselectable>
            {/* allUsers option */}
            {showAllUsers && (
              <div
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-control-bg",
                  selectedFullnames.has(ALL_USERS_USER_EMAIL) && "bg-accent/5"
                )}
                onClick={() => toggle(ALL_USERS_USER_EMAIL)}
              >
                <SelectionCheckbox
                  selected={selectedFullnames.has(ALL_USERS_USER_EMAIL)}
                />
                {/* Blue avatar circle */}
                <div
                  className="size-7 rounded-full flex items-center justify-center text-accent-text text-xs font-medium shrink-0"
                  style={{ backgroundColor: "#3B82F6" }}
                >
                  <Users className="h-4 w-4" />
                </div>
                <HighlightLabelText
                  text={t("settings.members.all-users")}
                  keyword={search}
                  className="text-sm font-medium"
                />
              </div>
            )}

            {/* Users */}
            {visibleUsers.length > 0 && (
              <div>
                <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-control-bg border-b">
                  {t("common.users")}
                </div>
                {visibleUsers.map((user) => {
                  const fullname = `users/${user.email}`;
                  const selected = selectedFullnames.has(fullname);
                  const isCurrentUser = user.email === currentUser?.email;
                  const displayName = user.title || user.email;
                  const color = getAvatarColor(displayName);
                  const initials = getInitials(displayName);
                  return (
                    <div
                      key={user.name}
                      className={cn(
                        "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-control-bg",
                        selected && "bg-accent/5"
                      )}
                      onClick={() => toggle(fullname)}
                    >
                      <SelectionCheckbox selected={selected} />
                      {/* Avatar */}
                      <div
                        className="size-7 rounded-full flex items-center justify-center text-accent-text text-xs font-medium shrink-0"
                        style={{ backgroundColor: color }}
                      >
                        {initials}
                      </div>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-x-1">
                          <HighlightLabelText
                            text={displayName}
                            keyword={search}
                            className="text-sm font-medium truncate"
                          />
                          {isCurrentUser && (
                            <span className="text-xs text-control-light bg-control-bg rounded-xs px-1">
                              {t("common.you")}
                            </span>
                          )}
                        </div>
                        {user.title && (
                          <HighlightLabelText
                            text={user.email}
                            keyword={search}
                            className="text-xs text-control-light truncate"
                          />
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Groups */}
            {visibleGroups.length > 0 && (
              <div>
                <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-control-bg border-b">
                  {t("common.groups")}
                </div>
                {visibleGroups.map((group) => {
                  const selected = selectedFullnames.has(group.name);
                  return (
                    <div
                      key={group.name}
                      role="option"
                      aria-selected={selected}
                      tabIndex={0}
                      className={cn(
                        "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-control-bg",
                        selected && "bg-accent/5"
                      )}
                      onClick={() => toggle(group.name)}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          toggle(group.name);
                        }
                      }}
                    >
                      <SelectionCheckbox selected={selected} />
                      <div className="size-7 rounded-full bg-control-bg-hover flex items-center justify-center shrink-0">
                        <Users className="h-4 w-4 text-control-light" />
                      </div>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-x-1.5">
                          <HighlightLabelText
                            text={group.title || group.email}
                            keyword={search}
                            className="text-sm font-medium truncate"
                          />
                          <span className="text-xs text-control-light">
                            ({group.members.length}{" "}
                            {t("common.members", {
                              count: group.members.length,
                            })}
                            )
                          </span>
                        </div>
                        <HighlightLabelText
                          text={group.email}
                          keyword={search}
                          className="text-xs text-control-light truncate"
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Service accounts */}
            {(visibleSpecialAccountMatch?.type ===
              AccountType.SERVICE_ACCOUNT ||
              visibleServiceAccounts.length > 0) && (
              <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-control-bg border-b">
                {t("settings.members.service-accounts")}
              </div>
            )}
            {visibleSpecialAccountMatch?.type ===
              AccountType.SERVICE_ACCOUNT && (
              <SpecialAccountOption
                keyword={search}
                match={visibleSpecialAccountMatch}
                selected={selectedFullnames.has(
                  visibleSpecialAccountMatch.fullname
                )}
                onToggle={() => toggle(visibleSpecialAccountMatch.fullname)}
              />
            )}
            {visibleServiceAccounts.map((account) => (
              <SpecialAccountOption
                key={account.fullname}
                keyword={search}
                match={account}
                selected={selectedFullnames.has(account.fullname)}
                onToggle={() => toggle(account.fullname)}
              />
            ))}

            {/* Workload identities */}
            {(visibleSpecialAccountMatch?.type ===
              AccountType.WORKLOAD_IDENTITY ||
              visibleWorkloadIdentities.length > 0) && (
              <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-control-bg border-b">
                {t("settings.members.workload-identities")}
              </div>
            )}
            {visibleSpecialAccountMatch?.type ===
              AccountType.WORKLOAD_IDENTITY && (
              <SpecialAccountOption
                keyword={search}
                match={visibleSpecialAccountMatch}
                selected={selectedFullnames.has(
                  visibleSpecialAccountMatch.fullname
                )}
                onToggle={() => toggle(visibleSpecialAccountMatch.fullname)}
              />
            )}
            {visibleWorkloadIdentities.map((account) => (
              <SpecialAccountOption
                key={account.fullname}
                keyword={search}
                match={account}
                selected={selectedFullnames.has(account.fullname)}
                onToggle={() => toggle(account.fullname)}
              />
            ))}

            {/* Arbitrary email fallback */}
            {visibleArbitraryEmailMatch && (
              <div
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-control-bg",
                  selectedFullnames.has(
                    `users/${visibleArbitraryEmailMatch}`
                  ) && "bg-accent/5"
                )}
                onClick={() => toggle(`users/${visibleArbitraryEmailMatch}`)}
              >
                <SelectionCheckbox
                  selected={selectedFullnames.has(
                    `users/${visibleArbitraryEmailMatch}`
                  )}
                />
                <div
                  className="size-7 rounded-full flex items-center justify-center text-accent-text text-xs font-medium shrink-0"
                  style={{
                    backgroundColor: getAvatarColor(visibleArbitraryEmailMatch),
                  }}
                >
                  {getInitials(visibleArbitraryEmailMatch.split("@")[0])}
                </div>
                <div className="flex flex-col min-w-0">
                  <HighlightLabelText
                    text={visibleArbitraryEmailMatch}
                    keyword={search}
                    className="text-sm font-medium truncate"
                  />
                </div>
              </div>
            )}

            {/* Empty state */}
            {!showAllUsers &&
              visibleUsers.length === 0 &&
              visibleGroups.length === 0 &&
              visibleSpecialAccounts.length === 0 &&
              !visibleSpecialAccountMatch &&
              !visibleArbitraryEmailMatch && (
                <div className="px-3 py-4 text-sm text-center text-control-light">
                  {t("common.no-data")}
                </div>
              )}
          </div>
        </div>
      )}
    </div>
  );
}
