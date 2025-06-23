import { defineStore } from "pinia";
import { ref } from "vue";
import { create } from "@bufbuild/protobuf";
import { roleServiceClientConnect } from "@/grpcweb";
import type { Role } from "@/types/proto/v1/role_service";
import { 
  ListRolesRequestSchema,
  UpdateRoleRequestSchema,
  DeleteRoleRequestSchema
} from "@/types/proto-es/v1/role_service_pb";
import { convertNewRoleToOld, convertOldRoleToNew } from "@/utils/v1/role-conversions";
import { useGracefulRequest } from "./utils";

export const useRoleStore = defineStore("role", () => {
  const roleList = ref<Role[]>([]);

  const fetchRoleList = async () => {
    const request = create(ListRolesRequestSchema, {});
    const response = await roleServiceClientConnect.listRoles(request);
    roleList.value = response.roles.map(convertNewRoleToOld);
    return roleList.value;
  };

  const getRoleByName = (name: string) => {
    return roleList.value.find((r) => r.name === name);
  };

  const upsertRole = async (role: Role) => {
    const newRole = convertOldRoleToNew(role);
    const request = create(UpdateRoleRequestSchema, {
      role: newRole,
      updateMask: {
        paths: ["title", "description", "permissions"],
      },
      allowMissing: true,
    });
    const response = await roleServiceClientConnect.updateRole(request);
    const updated = convertNewRoleToOld(response);
    const index = roleList.value.findIndex((r) => r.name === role.name);
    if (index >= 0) {
      roleList.value.splice(index, 1, updated);
    } else {
      roleList.value.push(updated);
    }
    return updated;
  };

  const deleteRole = async (role: Role) => {
    await useGracefulRequest(async () => {
      const request = create(DeleteRoleRequestSchema, {
        name: role.name,
      });
      await roleServiceClientConnect.deleteRole(request);
      const index = roleList.value.findIndex((r) => r.name === role.name);
      if (index >= 0) {
        roleList.value.splice(index, 1);
      }
    });
  };

  return {
    roleList,
    fetchRoleList,
    getRoleByName,
    upsertRole,
    deleteRole,
  };
});
