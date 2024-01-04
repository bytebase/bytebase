import { defineStore } from "pinia";
import { ref } from "vue";
import { roleServiceClient } from "@/grpcweb";
import { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName } from "@/utils";
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
    const existedRole = roleList.value.find((r) => r.name === role.name);
    if (existedRole) {
      // update
      const updated = await roleServiceClient.updateRole({
        role,
        updateMask: ["title", "description"],
      });
      const index = roleList.value.findIndex((r) => r.name === role.name);
      if (index >= 0) {
        roleList.value.splice(index, 1, updated);
      }
      return updated;
    } else {
      // create
      const created = await roleServiceClient.createRole({
        role,
        roleId: extractRoleResourceName(role.name),
      });
      roleList.value.push(created);
      return created;
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
    getRoleByName,
    upsertRole,
    deleteRole,
  };
});
