import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { ref } from "vue";
import { serviceAccountServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import {
  CreateServiceAccountRequestSchema,
  DeleteServiceAccountRequestSchema,
  GetServiceAccountRequestSchema,
  ListServiceAccountsRequestSchema,
  ServiceAccountSchema,
  UndeleteServiceAccountRequestSchema,
  UpdateServiceAccountRequestSchema,
} from "@/types/proto-es/v1/service_account_service_pb";
import {
  type User,
  UserSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
import { useActuatorV1Store } from "./v1/actuator";
import { extractServiceAccountId, serviceAccountNamePrefix } from "./v1/common";

export interface AccountFilter {
  query?: string;
  state?: State;
}

export const getAccountListFilter = (params: AccountFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(name.matches("${search}") || email.matches("${search}"))`);
  }
  if (params.state === State.DELETED) {
    filter.push(`state == "${State[params.state]}"`);
  }
  return filter.join(" && ");
};

export const ensureServiceAccountFullName = (identifier: string) => {
  const id = extractServiceAccountId(identifier);
  return `${serviceAccountNamePrefix}${id}`;
};

export const useServiceAccountStore = defineStore("serviceAccount", () => {
  const actuatorStore = useActuatorV1Store();
  const cacheByName = ref<Map<string, ServiceAccount>>(new Map());

  const listServiceAccounts = async ({
    parent,
    pageSize,
    pageToken,
    showDeleted,
    filter,
  }: {
    parent?: string;
    pageSize: number;
    pageToken: string | undefined;
    showDeleted: boolean;
    filter?: AccountFilter;
  }) => {
    const request = create(ListServiceAccountsRequestSchema, {
      parent: parent ?? "workspaces/-",
      pageSize,
      pageToken,
      showDeleted,
      filter: getAccountListFilter(filter ?? {}),
    });
    return serviceAccountServiceClientConnect.listServiceAccounts(request);
  };

  const fetchServiceAccount = async (name: string, silent = false) => {
    const request = create(GetServiceAccountRequestSchema, {
      name,
    });
    return serviceAccountServiceClientConnect.getServiceAccount(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
  };

  const getServiceAccount = (name: string) => {
    const validName = ensureServiceAccountFullName(name);
    const email = extractServiceAccountId(validName);
    return (
      cacheByName.value.get(validName) ??
      create(ServiceAccountSchema, {
        name,
        email,
        state: State.ACTIVE,
        title: email.split("@")[0],
      })
    );
  };

  const getOrFetchServiceAccount = async (name: string, silent = false) => {
    const validName = ensureServiceAccountFullName(name);
    if (cacheByName.value.has(validName)) {
      return cacheByName.value.get(validName)!;
    }
    const sa = await fetchServiceAccount(validName, silent);
    cacheByName.value.set(sa.name, sa);
    return sa;
  };

  const createServiceAccount = async (
    serviceAccountId: string,
    serviceAccount: Partial<ServiceAccount>,
    parent?: string
  ) => {
    const request = create(CreateServiceAccountRequestSchema, {
      parent: parent ?? "workspaces/-",
      serviceAccountId,
      serviceAccount: create(
        ServiceAccountSchema,
        serviceAccount as ServiceAccount
      ),
    });
    const sa =
      await serviceAccountServiceClientConnect.createServiceAccount(request);
    cacheByName.value.set(sa.name, sa);
    actuatorStore.updateUserStat([
      {
        count: 1,
        state: State.ACTIVE,
        userType: UserType.SERVICE_ACCOUNT,
      },
    ]);
    return sa;
  };

  const updateServiceAccount = async (
    serviceAccount: Partial<ServiceAccount>,
    updateMask: { paths: string[] }
  ) => {
    const request = create(UpdateServiceAccountRequestSchema, {
      serviceAccount: create(
        ServiceAccountSchema,
        serviceAccount as ServiceAccount
      ),
      updateMask,
    });
    const sa =
      await serviceAccountServiceClientConnect.updateServiceAccount(request);
    cacheByName.value.set(sa.name, sa);
    return sa;
  };

  const deleteServiceAccount = async (name: string) => {
    const request = create(DeleteServiceAccountRequestSchema, {
      name,
    });
    await serviceAccountServiceClientConnect.deleteServiceAccount(request);
    const sa = cacheByName.value.get(name);
    if (sa) {
      sa.state = State.DELETED;
    }
    actuatorStore.updateUserStat([
      {
        count: -1,
        state: State.ACTIVE,
        userType: UserType.SERVICE_ACCOUNT,
      },
      {
        count: 1,
        state: State.DELETED,
        userType: UserType.SERVICE_ACCOUNT,
      },
    ]);
  };

  const undeleteServiceAccount = async (name: string) => {
    const request = create(UndeleteServiceAccountRequestSchema, {
      name,
    });
    const sa =
      await serviceAccountServiceClientConnect.undeleteServiceAccount(request);
    cacheByName.value.set(sa.name, sa);
    actuatorStore.updateUserStat([
      {
        count: 1,
        state: State.ACTIVE,
        userType: UserType.SERVICE_ACCOUNT,
      },
      {
        count: -1,
        state: State.DELETED,
        userType: UserType.SERVICE_ACCOUNT,
      },
    ]);
    return sa;
  };

  return {
    listServiceAccounts,
    getServiceAccount,
    getOrFetchServiceAccount,
    createServiceAccount,
    updateServiceAccount,
    deleteServiceAccount,
    undeleteServiceAccount,
  };
});

export const serviceAccountToUser = (sa: ServiceAccount): User => {
  return create(UserSchema, {
    name: `users/${sa.email}`,
    email: sa.email,
    title: sa.title,
    state: sa.state,
    userType: UserType.SERVICE_ACCOUNT,
    serviceKey: sa.serviceKey,
  });
};
