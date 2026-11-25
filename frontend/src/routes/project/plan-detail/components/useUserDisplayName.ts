import { useEffect } from "react";
import { usePrincipal } from "@/hooks/usePrincipal";
import { useAppStore } from "@/stores/app";
import { AccountType, getAccountTypeByEmail } from "@/types/v1/user";

// A principal's display name for "users/{email}", fetched once if not cached;
// the email until it loads, or when it cannot. A service account or workload
// identity, as CI is, is read from its own service.
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
