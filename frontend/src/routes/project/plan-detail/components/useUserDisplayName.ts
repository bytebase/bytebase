import { useEffect } from "react";
import { usePrincipal } from "@/hooks/usePrincipal";
import { useAppStore } from "@/stores/app";
import { AccountType, getAccountTypeByEmail } from "@/types/v1/user";

// A principal's display name for "users/{email}", fetched once if not cached.
// A person reads as the email until loaded, or when it cannot be. A service
// account or workload identity, as CI is, is read from its own service and
// named from its email until then, as usePrincipal does.
export function useUserDisplayName(name: string | undefined): string {
  const email = name?.replace(/^users\//, "") ?? "";
  const isPerson = !email || getAccountTypeByEmail(email) === AccountType.USER;
  const user = useAppStore((state) =>
    name && isPerson ? state.getUserByIdentifier(name) : undefined
  );
  useEffect(() => {
    if (!name || !isPerson || user) return;
    void useAppStore
      .getState()
      .getOrFetchUserByIdentifier({ identifier: name, silent: true })
      .catch(() => {});
  }, [name, isPerson, user]);
  const principal = usePrincipal(isPerson ? undefined : name);
  return (isPerson ? user?.title : principal?.title) || email;
}
