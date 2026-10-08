import { useEffect } from "react";
import { useAppStore } from "@/stores/app";
import { ensureServiceAccountFullName } from "@/stores/app/serviceAccount";
import { ensureWorkloadIdentityFullName } from "@/stores/app/workloadIdentity";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { AccountType, getAccountTypeByEmail } from "@/types/v1/user";
import { ensureUserFullName } from "@/utils/v1/user";

interface Principal {
  email: string;
  title: string;
}

type AppState = ReturnType<typeof useAppStore.getState>;

// The user service refuses service accounts and workload identities, which
// have services of their own; the account type read from the email decides
// where a principal is read and fetched.
const PRINCIPAL_SERVICES: Record<
  AccountType,
  {
    name: (email: string) => string;
    title: (state: AppState, name: string) => string | undefined;
    fetch: (state: AppState, name: string) => Promise<unknown>;
  }
> = {
  [AccountType.SERVICE_ACCOUNT]: {
    name: ensureServiceAccountFullName,
    title: (state, name) => state.getServiceAccount(name).title,
    fetch: (state, name) => state.fetchServiceAccount(name, true),
  },
  [AccountType.WORKLOAD_IDENTITY]: {
    name: ensureWorkloadIdentityFullName,
    title: (state, name) => state.getWorkloadIdentity(name).title,
    fetch: (state, name) => state.fetchWorkloadIdentity(name, true),
  },
  [AccountType.USER]: {
    name: ensureUserFullName,
    title: (state, name) => state.getUserByIdentifier(name)?.title,
    fetch: (state, name) =>
      state.getOrFetchUserByIdentifier({ identifier: name }),
  },
};

/**
 * Whoever a `users/{email}` identifier names: a person, a service account, or
 * a workload identity. Until a record arrives the store names a service
 * account or workload identity from its email, as the member list does, and
 * a person has no title.
 */
export function usePrincipal(
  identifier: string | undefined
): Principal | undefined {
  const email = identifier ? extractUserEmail(identifier) : undefined;
  const service =
    PRINCIPAL_SERVICES[email ? getAccountTypeByEmail(email) : AccountType.USER];
  const name = email === undefined ? undefined : service.name(email);
  const title = useAppStore((state) =>
    name === undefined ? undefined : service.title(state, name)
  );
  useEffect(() => {
    if (name !== undefined) {
      void service.fetch(useAppStore.getState(), name);
    }
  }, [name, service]);
  if (email === undefined) {
    return undefined;
  }
  return { email, title: title ?? "" };
}
