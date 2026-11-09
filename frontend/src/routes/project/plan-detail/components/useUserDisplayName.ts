import { useEffect } from "react";
import { useAppStore } from "@/stores/app";

// A user's display name for "users/{email}", fetched once if not cached; the
// email until the user loads, or when it cannot.
export function useUserDisplayName(name: string | undefined): string {
  const user = useAppStore((state) =>
    name ? state.getUserByIdentifier(name) : undefined
  );
  useEffect(() => {
    if (!name || user) return;
    void useAppStore
      .getState()
      .getOrFetchUserByIdentifier({ identifier: name, silent: true })
      .catch(() => {});
  }, [name, user]);
  const email = name?.replace(/^users\//, "") ?? "";
  return user?.title || email;
}
