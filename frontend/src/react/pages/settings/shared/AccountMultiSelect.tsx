import { Check, ChevronDown, Search, Users, X } from "lucide-react";
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

// ---- Avatar helpers ----

const AVATAR_COLORS = [
  "#F59E0B",
  "#10B981",
  "#8B5CF6",
  "#EC4899",
  "#06B6D4",
  "#EF4444",
];

function getAvatarColor(name: string) {
  let hash = 0;
  for (let i = 0; i < name.length; i++)
    hash = (hash * 31 + name.charCodeAt(i)) | 0;
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

function getInitials(name: string) {
  return name
    .split(/\s+/)
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

// ---- Conversion helpers ----

// binding  →  fullname
// "user:alice@example.com"  →  "users/alice@example.com"
// "group:eng@example.com"   →  "groups/eng@example.com"
// "allUsers"                →  "allUsers"
function bindingToFullname(binding: string): string {
  if (binding === ALL_USERS_USER_EMAIL) return ALL_USERS_USER_EMAIL;
  if (binding.startsWith("user:"))
    return `users/${binding.slice("user:".length)}`;
  if (binding.startsWith("group:"))
    return `${groupNamePrefix}${binding.slice("group:".length)}`;
  return binding;
}

// fullname  →  binding
function fullnameToBinding(fullname: string): string {
  if (fullname === ALL_USERS_USER_EMAIL) return ALL_USERS_USER_EMAIL;
  if (fullname.startsWith("users/"))
    return `${userBindingPrefix}${fullname.slice("users/".length)}`;
  if (fullname.startsWith(groupNamePrefix))
    return `group:${fullname.slice(groupNamePrefix.length)}`;
  return fullname;
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
      onChange(value.filter((v) => v !== binding));
    } else {
      onChange([...value, binding]);
    }
  };

  const remove = (binding: string) => {
    if (disabled) return;
    onChange(value.filter((v) => v !== binding));
  };

  // Label for a selected binding chip
  const chipLabel = (binding: string): string => {
    if (binding === ALL_USERS_USER_EMAIL) {
      return t("settings.members.all-users");
    }
    const fullname = bindingToFullname(binding);
    if (fullname.startsWith("users/")) {
      const email = extractUserEmail(fullname);
      const user = users.find((u) => u.email === email);
      return user?.title || user?.email || email;
    }
    if (fullname.startsWith(groupNamePrefix)) {
      const email = fullname.slice(groupNamePrefix.length);
      const group = groups.find((g) => g.email === email);
      return group?.title || email;
    }
    return binding;
  };

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          disabled && "opacity-50 cursor-not-allowed",
          open && "ring-2 ring-accent border-accent"
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
        {!open && value.length === 0 && (
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
                <div
                  className={cn(
                    "h-4 w-4 rounded-xs border flex items-center justify-center shrink-0",
                    selectedFullnames.has(ALL_USERS_USER_EMAIL)
                      ? "bg-accent border-accent text-white"
                      : "border-control-border"
                  )}
                >
                  {selectedFullnames.has(ALL_USERS_USER_EMAIL) && (
                    <Check className="h-3 w-3" />
                  )}
                </div>
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
                            <span className="text-xs text-control-light bg-gray-100 rounded px-1">
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
                      <div className="h-7 w-7 rounded-full bg-gray-200 flex items-center justify-center shrink-0">
                        <Users className="h-4 w-4 text-control-light" />
                      </div>
                      <div className="flex flex-col min-w-0">
                        <span className="text-sm font-medium truncate">
                          {group.title || group.email}
                        </span>
                        <span className="text-xs text-control-light">
                          {t("common.members", { count: group.members.length })}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Empty state */}
            {!includeAllUsers && users.length === 0 && groups.length === 0 && (
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
