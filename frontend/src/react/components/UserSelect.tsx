import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Combobox, type ComboboxOption } from "@/react/components/ui/combobox";
import { useUserStore } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { ALL_USERS_USER_EMAIL } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { getDefaultPagination, isValidEmail } from "@/utils";

interface UserSelectProps {
  /** Selected user's full name, e.g. "users/foo@bar.com" (or empty). */
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  className?: string;
  /** Include the `users/allUsers` synthetic account in the list. */
  includeAllUsers?: boolean;
}

function toOption(user: User): ComboboxOption {
  const label = user.title || user.email;
  return {
    value: user.name,
    label,
    description: user.email && user.email !== label ? user.email : undefined,
  };
}

export function UserSelect({
  value,
  onChange,
  disabled,
  placeholder,
  className,
  includeAllUsers,
}: UserSelectProps) {
  const userStore = useUserStore();
  const [users, setUsers] = useState<User[]>([]);
  const [selectedUser, setSelectedUser] = useState<User | undefined>(undefined);
  const [search, setSearch] = useState("");
  // Track the latest in-flight request to ignore stale responses when the
  // user types quickly (a slow A can arrive after a fast B).
  const searchSeqRef = useRef(0);

  // Hydrate the selected user so its label renders even when it isn't in
  // the current search results.
  useEffect(() => {
    if (!value) {
      setSelectedUser(undefined);
      return;
    }
    let cancelled = false;
    userStore.getOrFetchUserByIdentifier({ identifier: value }).then((user) => {
      if (!cancelled) setSelectedUser(user);
    });
    return () => {
      cancelled = true;
    };
  }, [value, userStore]);

  const handleSearch = useCallback(
    (query: string) => {
      setSearch(query);
      const seq = ++searchSeqRef.current;
      userStore
        .fetchUserList({
          pageSize: getDefaultPagination(),
          filter: { query: query.trim() },
        })
        .then(({ users: fetched }) => {
          if (seq === searchSeqRef.current) setUsers(fetched);
        });
    },
    [userStore]
  );

  const options: ComboboxOption[] = useMemo(() => {
    const list: ComboboxOption[] = [];
    const seen = new Set<string>();
    if (includeAllUsers) {
      list.push({
        value: `${userNamePrefix}${ALL_USERS_USER_EMAIL}`,
        label: ALL_USERS_USER_EMAIL,
      });
      seen.add(`${userNamePrefix}${ALL_USERS_USER_EMAIL}`);
    }
    if (selectedUser && !seen.has(selectedUser.name)) {
      list.push(toOption(selectedUser));
      seen.add(selectedUser.name);
    }
    for (const user of users) {
      if (seen.has(user.name)) continue;
      list.push(toOption(user));
      seen.add(user.name);
    }
    // Allow selecting arbitrary emails typed in the search box — same pattern
    // as AccountMultiSelect. Supports SaaS flows where admins grant access to
    // emails before signup (invited-but-not-yet-signed-in accounts).
    const trimmed = search.trim();
    if (isValidEmail(trimmed) && !users.some((u) => u.email === trimmed)) {
      const customName = `${userNamePrefix}${trimmed}`;
      if (!seen.has(customName)) {
        list.push({ value: customName, label: trimmed });
      }
    }
    return list;
  }, [users, selectedUser, includeAllUsers, search]);

  return (
    <Combobox
      value={value}
      onChange={onChange}
      options={options}
      onSearch={handleSearch}
      placeholder={placeholder}
      disabled={disabled}
      className={className}
      portal
    />
  );
}
