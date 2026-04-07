import {
  Check,
  ChevronDown,
  KeyRound,
  Search,
  Shield,
  Users,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useCurrentUserV1, useGroupStore, useUserStore } from "@/store";
import { extractUserEmail, groupNamePrefix } from "@/store/modules/v1/common";
import { ALL_USERS_USER_EMAIL, userBindingPrefix } from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { isValidEmail } from "@/utils";

import { getAvatarColor, getInitials } from "./UserAvatar";

// ---- Account type detection ----

const SERVICE_ACCOUNT_SUFFIX = "@service.bytebase.com";
const WORKLOAD_IDENTITY_SUFFIX = "@workload.bytebase.com";

type SpecialAccountType = "serviceAccount" | "workloadIdentity" | null;

function detectSpecialAccount(input: string): SpecialAccountType {
  const lower = input.toLowerCase();
  if (
    lower.endsWith(SERVICE_ACCOUNT_SUFFIX) ||
    lower.startsWith("serviceaccounts/") ||
    lower.startsWith("serviceaccount:")
  )
    return "serviceAccount";
  if (
    lower.endsWith(WORKLOAD_IDENTITY_SUFFIX) ||
    lower.startsWith("workloadidentities/") ||
    lower.startsWith("workloadidentity:")
  )
    return "workloadIdentity";
  return null;
}

function extractSpecialEmail(input: string): string {
  if (input.includes("/")) return input.split("/").slice(1).join("/");
  if (input.includes(":")) return input.split(":").slice(1).join(":");
  return input;
}

// ---- Conversion helpers ----

// binding  →  fullname
function bindingToFullname(binding: string): string {
  if (binding === ALL_USERS_USER_EMAIL) return ALL_USERS_USER_EMAIL;
  if (binding.startsWith("user:"))
    return `users/${binding.slice("user:".length)}`;
  if (binding.startsWith("group:"))
    return `${groupNamePrefix}${binding.slice("group:".length)}`;
  if (binding.startsWith("serviceAccount:"))
    return `serviceAccounts/${binding.slice("serviceAccount:".length)}`;
  if (binding.startsWith("workloadIdentity:"))
    return `workloadIdentities/${binding.slice("workloadIdentity:".length)}`;
  return binding;
}

// fullname  →  binding
function fullnameToBinding(fullname: string): string {
  if (fullname === ALL_USERS_USER_EMAIL) return ALL_USERS_USER_EMAIL;
  if (fullname.startsWith("users/"))
    return `${userBindingPrefix}${fullname.slice("users/".length)}`;
  if (fullname.startsWith(groupNamePrefix))
    return `group:${fullname.slice(groupNamePrefix.length)}`;
  if (fullname.startsWith("serviceAccounts/"))
    return `serviceAccount:${fullname.slice("serviceAccounts/".length)}`;
  if (fullname.startsWith("workloadIdentities/"))
    return `workloadIdentity:${fullname.slice("workloadIdentities/".length)}`;
  return fullname;
}

// ---- Sub-components ----

function SelectionCheckbox({ selected }: { selected: boolean }) {
  return (
    <div
      className={cn(
        "h-4 w-4 rounded-xs border flex items-center justify-center shrink-0",
        selected
          ? "bg-accent border-accent text-white"
          : "border-control-border"
      )}
    >
      {selected && <Check className="h-3 w-3" />}
    </div>
  );
}

function SpecialAccountOption({
  match,
  selected,
  onToggle,
}: {
  match: { type: SpecialAccountType; email: string };
  selected: boolean;
  onToggle: () => void;
}) {
  const { t } = useTranslation();
  const isServiceAccount = match.type === "serviceAccount";
  const Icon = isServiceAccount ? KeyRound : Shield;
  const label = isServiceAccount
    ? t("settings.members.service-account")
    : t("settings.members.workload-identity");

  return (
    <div
      className={cn(
        "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-gray-50",
        selected && "bg-accent/5"
      )}
      onClick={onToggle}
    >
      <SelectionCheckbox selected={selected} />
      <div
        className="h-7 w-7 rounded-full flex items-center justify-center text-white text-xs font-medium shrink-0"
        style={{ backgroundColor: getAvatarColor(match.email) }}
      >
        <Icon className="h-3.5 w-3.5" />
      </div>
      <div className="flex flex-col min-w-0">
        <div className="flex items-center gap-x-1">
          <span className="text-sm font-medium truncate">
            {match.email.split("@")[0]}
          </span>
          <span className="text-xs text-control-light bg-gray-100 rounded-xs px-1">
            {label}
          </span>
        </div>
        <span className="text-xs text-control-light truncate">
          {match.email}
        </span>
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
}: {
  value: string[];
  onChange: (value: string[]) => void;
  disabled?: boolean;
  includeAllUsers?: boolean;
}) {
  const { t } = useTranslation();
  const userStore = useUserStore();
  const groupStore = useGroupStore();
  const currentUser = useVueState(() => useCurrentUserV1().value);

  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [users, setUsers] = useState<User[]>([]);
  const [groups, setGroups] = useState<Group[]>([]);

  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Cache display labels for selected items so they persist across search changes
  const labelCacheRef = useRef<Map<string, string>>(new Map());

  // Fetch on search change (debounced 300ms)
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      const query = search.trim();
      userStore
        .fetchUserList({ pageSize: 50, filter: { query } })
        .then(({ users: fetched }) => setUsers(fetched));
      groupStore
        .fetchGroupList({ pageSize: 50, filter: { query } })
        .then(({ groups: fetched }) => setGroups(fetched));
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [search, userStore, groupStore]);

  const handleClickOutside = useCallback(() => {
    setOpen(false);
    setSearch("");
  }, []);
  useClickOutside(containerRef, open, handleClickOutside);

  // Selected fullnames set for quick lookup
  const selectedFullnames = useMemo(
    () => new Set(value.map(bindingToFullname)),
    [value]
  );

  const toggle = (fullname: string) => {
    if (disabled) return;
    const binding = fullnameToBinding(fullname);
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
      const email = fullname.slice(groupNamePrefix.length);
      const group = groups.find((g) => g.email === email);
      return group?.title || email;
    }
    return undefined;
  };

  // Detect service account / workload identity typed in search
  const specialAccountMatch = useMemo((): {
    type: SpecialAccountType;
    email: string;
    fullname: string;
  } | null => {
    const trimmed = search.trim();
    if (!trimmed) return null;
    const type = detectSpecialAccount(trimmed);
    if (!type) return null;
    const email = extractSpecialEmail(trimmed);
    if (!email) return null;
    const prefix =
      type === "serviceAccount" ? "serviceAccounts/" : "workloadIdentities/";
    return { type, email, fullname: `${prefix}${email}` };
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

  // Label for a selected binding chip — uses cache to survive search changes
  const chipLabel = (binding: string): string => {
    if (binding === ALL_USERS_USER_EMAIL) {
      return t("settings.members.all-users");
    }
    const cached = labelCacheRef.current.get(binding);
    if (cached) return cached;
    const fullname = bindingToFullname(binding);
    if (fullname.startsWith("serviceAccounts/"))
      return fullname.slice("serviceAccounts/".length);
    if (fullname.startsWith("workloadIdentities/"))
      return fullname.slice("workloadIdentities/".length);
    return resolveLabel(fullname) || binding;
  };

  return (
    <div ref={containerRef} className="relative">
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
            className="inline-flex items-center gap-x-1 rounded-xs bg-gray-100 px-1.5 py-0.5 text-xs"
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
            {t("settings.members.select-account", { count: 2 })}
          </span>
        )}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-control-border rounded-sm shadow-lg max-h-72 overflow-auto flex flex-col">
          {/* Search input */}
          <div className="flex items-center gap-x-2 px-3 py-2 border-b sticky top-0 bg-white">
            <Search className="h-4 w-4 text-control-light shrink-0" />
            <input
              ref={inputRef}
              className="flex-1 outline-hidden text-sm bg-transparent"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("common.search-for-more")}
            />
          </div>

          <div className="overflow-auto">
            {/* allUsers option */}
            {includeAllUsers && (
              <div
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-gray-50",
                  selectedFullnames.has(ALL_USERS_USER_EMAIL) && "bg-accent/5"
                )}
                onClick={() => toggle(ALL_USERS_USER_EMAIL)}
              >
                <SelectionCheckbox
                  selected={selectedFullnames.has(ALL_USERS_USER_EMAIL)}
                />
                {/* Blue avatar circle */}
                <div
                  className="h-7 w-7 rounded-full flex items-center justify-center text-white text-xs font-medium shrink-0"
                  style={{ backgroundColor: "#3B82F6" }}
                >
                  <Users className="h-4 w-4" />
                </div>
                <span className="text-sm font-medium">
                  {t("settings.members.all-users")}
                </span>
              </div>
            )}

            {/* Users */}
            {users.length > 0 && (
              <div>
                <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-gray-50 border-b">
                  {t("common.users")}
                </div>
                {users.map((user) => {
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
                        "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-gray-50",
                        selected && "bg-accent/5"
                      )}
                      onClick={() => toggle(fullname)}
                    >
                      <SelectionCheckbox selected={selected} />
                      {/* Avatar */}
                      <div
                        className="h-7 w-7 rounded-full flex items-center justify-center text-white text-xs font-medium shrink-0"
                        style={{ backgroundColor: color }}
                      >
                        {initials}
                      </div>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-x-1">
                          <span className="text-sm font-medium truncate">
                            {displayName}
                          </span>
                          {isCurrentUser && (
                            <span className="text-xs text-control-light bg-gray-100 rounded-xs px-1">
                              {t("common.you")}
                            </span>
                          )}
                        </div>
                        {user.title && (
                          <span className="text-xs text-control-light truncate">
                            {user.email}
                          </span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Groups */}
            {groups.length > 0 && (
              <div>
                <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-gray-50 border-b">
                  {t("common.groups")}
                </div>
                {groups.map((group) => {
                  const fullname = `${groupNamePrefix}${group.email}`;
                  const selected = selectedFullnames.has(fullname);
                  return (
                    <div
                      key={group.name}
                      className={cn(
                        "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-gray-50",
                        selected && "bg-accent/5"
                      )}
                      onClick={() => toggle(fullname)}
                    >
                      <SelectionCheckbox selected={selected} />
                      <div className="h-7 w-7 rounded-full bg-gray-200 flex items-center justify-center shrink-0">
                        <Users className="h-4 w-4 text-control-light" />
                      </div>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-x-1.5">
                          <span className="text-sm font-medium truncate">
                            {group.title || group.email}
                          </span>
                          <span className="text-xs text-control-light">
                            ({group.members.length}{" "}
                            {t("common.members", {
                              count: group.members.length,
                            })}
                            )
                          </span>
                        </div>
                        <span className="text-xs text-control-light truncate">
                          {group.email}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Service account / workload identity match */}
            {specialAccountMatch && (
              <SpecialAccountOption
                match={specialAccountMatch}
                selected={selectedFullnames.has(specialAccountMatch.fullname)}
                onToggle={() => toggle(specialAccountMatch.fullname)}
              />
            )}

            {/* Arbitrary email fallback */}
            {arbitraryEmailMatch && (
              <div
                className={cn(
                  "flex items-center gap-x-3 px-3 py-2 cursor-pointer hover:bg-gray-50",
                  selectedFullnames.has(`users/${arbitraryEmailMatch}`) &&
                    "bg-accent/5"
                )}
                onClick={() => toggle(`users/${arbitraryEmailMatch}`)}
              >
                <SelectionCheckbox
                  selected={selectedFullnames.has(
                    `users/${arbitraryEmailMatch}`
                  )}
                />
                <div
                  className="h-7 w-7 rounded-full flex items-center justify-center text-white text-xs font-medium shrink-0"
                  style={{
                    backgroundColor: getAvatarColor(arbitraryEmailMatch),
                  }}
                >
                  {getInitials(arbitraryEmailMatch.split("@")[0])}
                </div>
                <div className="flex flex-col min-w-0">
                  <span className="text-sm font-medium truncate">
                    {arbitraryEmailMatch}
                  </span>
                </div>
              </div>
            )}

            {/* Empty state */}
            {!includeAllUsers &&
              users.length === 0 &&
              groups.length === 0 &&
              !specialAccountMatch &&
              !arbitraryEmailMatch && (
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
