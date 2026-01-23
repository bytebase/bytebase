import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { ref } from "vue";
import { workloadIdentityServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type User,
  UserSchema,
  UserType,
} from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  CreateWorkloadIdentityRequestSchema,
  DeleteWorkloadIdentityRequestSchema,
  GetWorkloadIdentityRequestSchema,
  ListWorkloadIdentitiesRequestSchema,
  UndeleteWorkloadIdentityRequestSchema,
  UpdateWorkloadIdentityRequestSchema,
  WorkloadIdentitySchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { useActuatorV1Store } from "./v1/actuator";

export const useWorkloadIdentityStore = defineStore("workloadIdentity", () => {
  const actuatorStore = useActuatorV1Store();
  const cacheByName = ref<Map<string, WorkloadIdentity>>(new Map());

  const listWorkloadIdentities = async (
    pageSize: number,
    pageToken: string | undefined,
    showDeleted: boolean
  ) => {
    const request = create(ListWorkloadIdentitiesRequestSchema, {
      pageSize,
      pageToken,
      showDeleted,
    });
    return workloadIdentityServiceClientConnect.listWorkloadIdentities(request);
  };

  const fetchWorkloadIdentity = async (name: string, silent = false) => {
    const request = create(GetWorkloadIdentityRequestSchema, {
      name,
    });
    return workloadIdentityServiceClientConnect.getWorkloadIdentity(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
  };

  const getWorkloadIdentity = (name: string) => {
    return cacheByName.value.get(name);
  };

  const getOrFetchWorkloadIdentity = async (
    name: string,
    silent = false
  ): Promise<WorkloadIdentity> => {
    const cached = getWorkloadIdentity(name);
    if (cached) {
      return cached;
    }
    const wi = await fetchWorkloadIdentity(name, silent);
    cacheByName.value.set(name, wi);
    return wi;
  };

  const createWorkloadIdentity = async (
    workloadIdentityId: string,
    workloadIdentity: Partial<WorkloadIdentity>
  ) => {
    const request = create(CreateWorkloadIdentityRequestSchema, {
      workloadIdentityId,
      workloadIdentity: create(
        WorkloadIdentitySchema,
        workloadIdentity as WorkloadIdentity
      ),
    });
    const wi =
      await workloadIdentityServiceClientConnect.createWorkloadIdentity(
        request
      );
    cacheByName.value.set(wi.name, wi);
    actuatorStore.updateUserStat([
      {
        count: 1,
        state: State.ACTIVE,
        userType: UserType.WORKLOAD_IDENTITY,
      },
    ]);
    return wi;
  };

  const updateWorkloadIdentity = async (
    workloadIdentity: Partial<WorkloadIdentity>,
    updateMask: { paths: string[] }
  ) => {
    const request = create(UpdateWorkloadIdentityRequestSchema, {
      workloadIdentity: create(
        WorkloadIdentitySchema,
        workloadIdentity as WorkloadIdentity
      ),
      updateMask,
    });
    const wi =
      await workloadIdentityServiceClientConnect.updateWorkloadIdentity(
        request
      );
    cacheByName.value.set(wi.name, wi);
    return wi;
  };

  const deleteWorkloadIdentity = async (name: string) => {
    const request = create(DeleteWorkloadIdentityRequestSchema, {
      name,
    });
    await workloadIdentityServiceClientConnect.deleteWorkloadIdentity(request);
    const wi = cacheByName.value.get(name);
    if (wi) {
      wi.state = State.DELETED;
    }
    actuatorStore.updateUserStat([
      {
        count: -1,
        state: State.ACTIVE,
        userType: UserType.WORKLOAD_IDENTITY,
      },
      {
        count: 1,
        state: State.DELETED,
        userType: UserType.WORKLOAD_IDENTITY,
      },
    ]);
  };

  const undeleteWorkloadIdentity = async (name: string) => {
    const request = create(UndeleteWorkloadIdentityRequestSchema, {
      name,
    });
    const wi =
      await workloadIdentityServiceClientConnect.undeleteWorkloadIdentity(
        request
      );
    cacheByName.value.set(wi.name, wi);
    actuatorStore.updateUserStat([
      {
        count: 1,
        state: State.ACTIVE,
        userType: UserType.WORKLOAD_IDENTITY,
      },
      {
        count: -1,
        state: State.DELETED,
        userType: UserType.WORKLOAD_IDENTITY,
      },
    ]);
    return wi;
  };

  return {
    listWorkloadIdentities,
    getWorkloadIdentity,
    getOrFetchWorkloadIdentity,
    createWorkloadIdentity,
    updateWorkloadIdentity,
    deleteWorkloadIdentity,
    undeleteWorkloadIdentity,
  };
});

export const workloadIdentityToUser = (wi: WorkloadIdentity): User => {
  return create(UserSchema, {
    name: `users/${wi.email}`,
    email: wi.email,
    title: wi.title,
    state: wi.state,
    userType: UserType.WORKLOAD_IDENTITY,
    workloadIdentityConfig: wi.workloadIdentityConfig,
  });
};
