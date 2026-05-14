import { create as createProto } from "@bufbuild/protobuf";
import { instanceRoleServiceClientConnect } from "@/connect";
import { ListInstanceRolesRequestSchema } from "@/types/proto-es/v1/instance_role_service_pb";
import type { AppSliceCreator, InstanceRoleSlice } from "./types";

export const createInstanceRoleSlice: AppSliceCreator<InstanceRoleSlice> = (
  set,
  get
) => ({
  rolesByInstance: {},
  roleRequests: {},

  fetchInstanceRoles: async (instance) => {
    const existing = get().rolesByInstance[instance];
    if (existing) return existing;
    const pending = get().roleRequests[instance];
    if (pending) return pending;

    const request = instanceRoleServiceClientConnect
      .listInstanceRoles(
        createProto(ListInstanceRolesRequestSchema, { parent: instance })
      )
      .then((response) => {
        const roles = response.roles;
        set((state) => {
          const { [instance]: _, ...roleRequests } = state.roleRequests;
          return {
            rolesByInstance: {
              ...state.rolesByInstance,
              [instance]: roles,
            },
            roleRequests,
          };
        });
        return roles;
      })
      .catch(() => {
        set((state) => {
          const { [instance]: _, ...roleRequests } = state.roleRequests;
          return { roleRequests };
        });
        return [];
      });
    set((state) => ({
      roleRequests: {
        ...state.roleRequests,
        [instance]: request,
      },
    }));
    return request;
  },
});
