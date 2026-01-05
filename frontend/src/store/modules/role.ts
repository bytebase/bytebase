import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { ref } from "vue";
import { roleServiceClientConnect } from "@/connect";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import {
  DeleteRoleRequestSchema,
  ListRolesRequestSchema,
  UpdateRoleRequestSchema,
} from "@/types/proto-es/v1/role_service_pb";
import { useGracefulRequest } from "./utils";

export const useRoleStore = defineStore("role", () => {
  const roleList = ref<Role[]>([]);

  const fetchRoleList = async () => {
    const request = create(ListRolesRequestSchema, {});
    const response = await roleServiceClientConnect.listRoles(request);
    roleList.value = response.roles;
    return roleList.value;
  };

  const getRoleByName = (name: string) => {
    return roleList.value.find((r) => r.name === name);
  };

  const upsertRole = async (role: Role) => {
    const request = create(UpdateRoleRequestSchema, {
      role: role,
      updateMask: {
        paths: ["title", "description", "permissions"],
      },
      allowMissing: true,
    });
    const response = await roleServiceClientConnect.updateRole(request);
    const index = roleList.value.findIndex((r) => r.name === role.name);
    if (index >= 0) {
      roleList.value.splice(index, 1, response);
    } else {
      roleList.value.push(response);
    }
    return response;
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
