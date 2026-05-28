import { create as createProto } from "@bufbuild/protobuf";
import { roleServiceClientConnect } from "@/connect";
import {
  DeleteRoleRequestSchema,
  ListRolesRequestSchema,
  UpdateRoleRequestSchema,
} from "@/types/proto-es/v1/role_service_pb";
import type { AppSliceCreator, RoleSlice } from "./types";

export const createRoleSlice: AppSliceCreator<RoleSlice> = (set, get) => ({
  roleList: [],

  listRoles: async () => {
    const response = await roleServiceClientConnect.listRoles(
      createProto(ListRolesRequestSchema, {})
    );
    set({ roleList: response.roles });
    return response.roles;
  },

  getRoleByName: (name) => get().roleList.find((role) => role.name === name),

  upsertRole: async (role) => {
    const response = await roleServiceClientConnect.updateRole(
      createProto(UpdateRoleRequestSchema, {
        role,
        updateMask: {
          paths: ["title", "description", "permissions"],
        },
        allowMissing: true,
      })
    );
    set((state) => {
      const index = state.roleList.findIndex((r) => r.name === role.name);
      if (index < 0) {
        return { roleList: [...state.roleList, response] };
      }
      const roleList = [...state.roleList];
      roleList.splice(index, 1, response);
      return { roleList };
    });
    return response;
  },

  deleteRole: async (role) => {
    await roleServiceClientConnect.deleteRole(
      createProto(DeleteRoleRequestSchema, { name: role.name })
    );
    set((state) => ({
      roleList: state.roleList.filter((r) => r.name !== role.name),
    }));
  },
});
