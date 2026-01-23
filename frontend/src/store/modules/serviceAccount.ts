import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { serviceAccountServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { serviceAccountBindingPrefix } from "@/types";
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

const ensureServiceAccountFullName = (identifier: string) => {
  const id = extractServiceAccountId(identifier);
  return `${serviceAccountNamePrefix}${id}`;
};

export const useServiceAccountStore = defineStore("serviceAccount", () => {
  const actuatorStore = useActuatorV1Store();
  const cacheByName = ref<Map<string, ServiceAccount>>(new Map());

  const listServiceAccounts = async (
    pageSize: number,
    pageToken: string | undefined,
    showDeleted: boolean
  ) => {
    const request = create(ListServiceAccountsRequestSchema, {
      pageSize,
      pageToken,
      showDeleted,
    });
    return serviceAccountServiceClientConnect.listServiceAccounts(request);
  };

  const fetchServiceAccount = async (name: string, silent = false) => {
    const request = create(GetServiceAccountRequestSchema, {
      name: ensureServiceAccountFullName(name),
    });
    return serviceAccountServiceClientConnect.getServiceAccount(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
  };

  const getServiceAccount = (name: string) => {
    return cacheByName.value.get(ensureServiceAccountFullName(name));
  };

  const getOrFetchServiceAccount = async (name: string, silent = false) => {
    const cached = getServiceAccount(name);
    if (cached) {
      return cached;
    }
    const sa = await fetchServiceAccount(name, silent);
    cacheByName.value.set(sa.name, sa);
    return sa;
  };

  const batchGetOrFetchServiceAccounts = async (nameList: string[]) => {
    const validList = uniq(nameList).filter(
      (name) =>
        Boolean(name) &&
        (name.startsWith(serviceAccountNamePrefix) ||
          name.startsWith(serviceAccountBindingPrefix))
    );
    try {
      const pendingFetch = validList
        .filter((name) => {
          return getServiceAccount(name) === undefined;
        })
        .map((name) => ensureServiceAccountFullName(name));

      const resp =
        await serviceAccountServiceClientConnect.batchGetServiceAccounts(
          {
            names: pendingFetch,
          },
          {
            contextValues: createContextValues().set(silentContextKey, true),
          }
        );
      for (const sa of resp.serviceAccounts) {
        cacheByName.value.set(sa.name, sa);
      }
    } catch {}
  };

  const createServiceAccount = async (
    serviceAccountId: string,
    serviceAccount: Partial<ServiceAccount>
  ) => {
    const request = create(CreateServiceAccountRequestSchema, {
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
    batchGetOrFetchServiceAccounts,
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
