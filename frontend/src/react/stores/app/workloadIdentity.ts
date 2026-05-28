import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { workloadIdentityServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import { type User, UserSchema } from "@/types/proto-es/v1/user_service_pb";
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
import { buildAccountListFilter } from "./serviceAccount";
import type { AppSliceCreator, WorkloadIdentitySlice } from "./types";

const workloadIdentityNamePrefix = "workloadIdentities/";

export const extractWorkloadIdentityId = (identifier: string) => {
  const matches = identifier.match(
    /^(?:workloadIdentity:|workloadIdentities\/)(.+)$/
  );
  return matches?.[1] ?? identifier;
};

export const ensureWorkloadIdentityFullName = (identifier: string) => {
  const id = extractWorkloadIdentityId(identifier);
  return `${workloadIdentityNamePrefix}${id}`;
};

export const workloadIdentityToUser = (wi: WorkloadIdentity): User => {
  return createProto(UserSchema, {
    name: `users/${wi.email}`,
    email: wi.email,
    title: wi.title,
    state: wi.state,
  });
};

export const createWorkloadIdentitySlice: AppSliceCreator<
  WorkloadIdentitySlice
> = (set, get) => ({
  workloadIdentitiesByName: {},
  workloadIdentityRequests: {},

  listWorkloadIdentities: async (params) => {
    const response =
      await workloadIdentityServiceClientConnect.listWorkloadIdentities(
        createProto(ListWorkloadIdentitiesRequestSchema, {
          parent: params.parent,
          pageSize: params.pageSize,
          pageToken: params.pageToken,
          showDeleted: params.showDeleted,
          filter: buildAccountListFilter(params.filter ?? {}),
        })
      );
    set((state) => ({
      workloadIdentitiesByName: {
        ...state.workloadIdentitiesByName,
        ...Object.fromEntries(
          response.workloadIdentities.map((wi) => [wi.name, wi])
        ),
      },
    }));
    return {
      workloadIdentities: response.workloadIdentities,
      nextPageToken: response.nextPageToken,
    };
  },

  fetchWorkloadIdentity: async (name, silent = false) => {
    const validName = ensureWorkloadIdentityFullName(name);
    const existing = get().workloadIdentitiesByName[validName];
    if (existing) return existing;
    const pending = get().workloadIdentityRequests[validName];
    if (pending) return pending;

    const request = workloadIdentityServiceClientConnect
      .getWorkloadIdentity(
        createProto(GetWorkloadIdentityRequestSchema, { name: validName }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      )
      .then((workloadIdentity) => {
        set((state) => {
          const { [validName]: _, ...workloadIdentityRequests } =
            state.workloadIdentityRequests;
          return {
            workloadIdentitiesByName: {
              ...state.workloadIdentitiesByName,
              [workloadIdentity.name]: workloadIdentity,
            },
            workloadIdentityRequests,
          };
        });
        return workloadIdentity;
      })
      .catch(() => {
        set((state) => {
          const { [validName]: _, ...workloadIdentityRequests } =
            state.workloadIdentityRequests;
          return { workloadIdentityRequests };
        });
        return undefined;
      });
    set((state) => ({
      workloadIdentityRequests: {
        ...state.workloadIdentityRequests,
        [validName]: request,
      },
    }));
    return request;
  },

  getWorkloadIdentity: (name) => {
    const validName = ensureWorkloadIdentityFullName(name);
    const email = extractWorkloadIdentityId(validName);
    return (
      get().workloadIdentitiesByName[validName] ??
      createProto(WorkloadIdentitySchema, {
        name: validName,
        email,
        state: State.ACTIVE,
        title: email.split("@")[0],
      })
    );
  },

  createWorkloadIdentity: async (
    workloadIdentityId,
    workloadIdentity,
    parent
  ) => {
    const wi =
      await workloadIdentityServiceClientConnect.createWorkloadIdentity(
        createProto(CreateWorkloadIdentityRequestSchema, {
          parent,
          workloadIdentityId,
          workloadIdentity: createProto(
            WorkloadIdentitySchema,
            workloadIdentity as WorkloadIdentity
          ),
        })
      );
    set((state) => ({
      workloadIdentitiesByName: {
        ...state.workloadIdentitiesByName,
        [wi.name]: wi,
      },
    }));
    return wi;
  },

  updateWorkloadIdentity: async (workloadIdentity, updateMask) => {
    const wi =
      await workloadIdentityServiceClientConnect.updateWorkloadIdentity(
        createProto(UpdateWorkloadIdentityRequestSchema, {
          workloadIdentity: createProto(
            WorkloadIdentitySchema,
            workloadIdentity as WorkloadIdentity
          ),
          updateMask,
        })
      );
    set((state) => ({
      workloadIdentitiesByName: {
        ...state.workloadIdentitiesByName,
        [wi.name]: wi,
      },
    }));
    return wi;
  },

  deleteWorkloadIdentity: async (name) => {
    const validName = ensureWorkloadIdentityFullName(name);
    await workloadIdentityServiceClientConnect.deleteWorkloadIdentity(
      createProto(DeleteWorkloadIdentityRequestSchema, { name: validName })
    );
    set((state) => {
      const cached = state.workloadIdentitiesByName[validName];
      if (!cached) return {};
      return {
        workloadIdentitiesByName: {
          ...state.workloadIdentitiesByName,
          [validName]: { ...cached, state: State.DELETED },
        },
      };
    });
  },

  undeleteWorkloadIdentity: async (name) => {
    const wi =
      await workloadIdentityServiceClientConnect.undeleteWorkloadIdentity(
        createProto(UndeleteWorkloadIdentityRequestSchema, { name })
      );
    set((state) => ({
      workloadIdentitiesByName: {
        ...state.workloadIdentitiesByName,
        [wi.name]: wi,
      },
    }));
    return wi;
  },
});
