import { defineStore } from "pinia";
import { ref } from "vue";
import { roleServiceClient } from "@/grpcweb";
import { ComposedRole } from "@/types/iam/role";
import { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName } from "@/utils";
import { useGracefulRequest } from "./utils";

export const useRoleStore = defineStore("role", () => {
  const roleList = ref<ComposedRole[]>([]);

  const fetchRoleList = async () => {
    const { roles } = await roleServiceClient.listRoles({});
    roleList.value = roles as ComposedRole[];
    return roleList.value;
  };

  const getRoleByName = (name: string) => {
    return roleList.value.find((r) => r.name === name) || Role.fromPartial({});
  };

  const upsertRole = async (role: Role) => {
    const existedRole = roleList.value.find((r) => r.name === role.name);
    if (existedRole) {
      // update
      const updated = (await roleServiceClient.updateRole({
        role,
        updateMask: ["title", "description"],
      })) as ComposedRole;
      return updated;
    } else {
      // create
      const created = (await roleServiceClient.createRole({
        role,
        roleId: extractRoleResourceName(role.name),
      })) as ComposedRole;
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
