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

export const useServiceAccountStore = defineStore("serviceAccount", () => {
  const actuatorStore = useActuatorV1Store();
  const cacheByName = ref<Map<string, ServiceAccount>>(new Map());

  // List service accounts.
  // parent: empty for workspace-level, "projects/{project}" for project-level
  const listServiceAccounts = async (
    pageSize: number,
    pageToken: string | undefined,
    showDeleted: boolean,
    parent = ""
  ) => {
    const request = create(ListServiceAccountsRequestSchema, {
      parent,
      pageSize,
      pageToken,
      showDeleted,
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
    return cacheByName.value.get(name);
  };

  const getOrFetchServiceAccount = async (name: string, silent = false) => {
    const cached = getServiceAccount(name);
    if (cached) {
      return cached;
    }
    const sa = await fetchServiceAccount(name, silent);
    cacheByName.value.set(name, sa);
    return sa;
  };

  // Create service account.
  // parent: empty for workspace-level, "projects/{project}" for project-level
  const createServiceAccount = async (
    serviceAccountId: string,
    serviceAccount: Partial<ServiceAccount>,
    parent = ""
  ) => {
    const request = create(CreateServiceAccountRequestSchema, {
      parent,
      serviceAccountId,
      serviceAccount: create(
        ServiceAccountSchema,
        serviceAccount as ServiceAccount
      ),
    });
    const sa =
      await serviceAccountServiceClientConnect.createServiceAccount(request);
    cacheByName.value.set(sa.name, sa);
    await actuatorStore.fetchServerInfo();
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
    await actuatorStore.fetchServerInfo();
  };

  const undeleteServiceAccount = async (name: string) => {
    const request = create(UndeleteServiceAccountRequestSchema, {
      name,
    });
    const sa =
      await serviceAccountServiceClientConnect.undeleteServiceAccount(request);
    cacheByName.value.set(sa.name, sa);
    await actuatorStore.fetchServerInfo();
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
