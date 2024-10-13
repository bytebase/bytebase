import { defineStore } from "pinia";
import { ref } from "vue";
import { roleServiceClient } from "@/grpcweb";
import type { Role } from "@/types/proto/v1/role_service";
import { useGracefulRequest } from "./utils";

export const useRoleStore = defineStore("role", () => {
  const roleList = ref<Role[]>([]);

  const fetchRoleList = async () => {
    const { roles } = await roleServiceClient.listRoles({});
    roleList.value = roles as Role[];
    return roleList.value;
  };

  const getRoleByName = (name: string) => {
    return roleList.value.find((r) => r.name === name);
  };

  const upsertRole = async (role: Role) => {
    const updated = await roleServiceClient.updateRole({
      role,
      updateMask: ["title", "description", "permissions"],
      allowMissing: true,
    });
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
    getRoleByName,
    upsertRole,
    deleteRole,
  };
});
