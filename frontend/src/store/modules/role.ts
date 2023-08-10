import { defineStore } from "pinia";
import { ref } from "vue";
import { roleServiceClient } from "@/grpcweb";
import { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName } from "@/utils";
import { useGracefulRequest } from "./utils";

export const useRoleStore = defineStore("role", () => {
  const roleList = ref<Role[]>([]);

  const fetchRoleList = async () => {
    const response = await roleServiceClient.listRoles({});
    roleList.value = response.roles;
    return roleList.value;
  };

  const upsertRole = async (role: Role) => {
    const existedRole = roleList.value.find((r) => r.name === role.name);
    if (existedRole) {
      // update
      const updated = await roleServiceClient.updateRole({
        role,
        updateMask: ["title", "description"],
      });
      Object.assign(existedRole, updated);
    } else {
      // create
      const created = await roleServiceClient.createRole({
        role,
        roleId: extractRoleResourceName(role.name),
      });
      Object.assign(role, created);
      roleList.value.push(role);
    }
  };

  const deleteRole = async (role: Role) => {
    await useGracefulRequest(async () => {
      await roleServiceClient.deleteRole({
        name: role.name,
      });
      const index = roleList.value.findIndex((r) => r.name === role.name);
      if (index >= 0) {
        roleList.value.splice(index, 1);
      }
    });
  };

  return {
    roleList,
    fetchRoleList,
    upsertRole,
    deleteRole,
  };
});
