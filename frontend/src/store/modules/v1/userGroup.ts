import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { userGroupServiceClient } from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import type { UserGroup } from "@/types/proto/v1/user_group";
import { hasWorkspacePermissionV2 } from "@/utils";
import { userGroupNamePrefix } from "./common";

export const extractGroupEmail = (emailResource: string) => {
  const matches = emailResource.match(/^(?:group:|groups\/)(.+)$/);
  return matches?.[1] ?? emailResource;
};

export const useUserGroupStore = defineStore("user_group", () => {
  const currentUser = useCurrentUserV1();
  const groupMapByName = reactive(new Map<string, UserGroup>());
  const resetCache = () => {
    groupMapByName.clear();
  };

  // Getters
  const groupList = computed(() => {
    return orderBy(
      Array.from(groupMapByName.values()),
      (group) => group.createTime,
      "desc"
    );
  });

  // Actions
  const getGroupName = (email: string) => `${userGroupNamePrefix}${email}`;

  const getGroupByEmail = (email: string) => {
    return groupMapByName.get(getGroupName(email));
  };

  const getGroupByIdentifier = (id: string) => {
    return getGroupByEmail(extractGroupEmail(id));
  };

  const fetchGroupList = async () => {
    if (!hasWorkspacePermissionV2(currentUser.value, "bb.userGroups.list")) {
      return [];
    }

    const { groups } = await userGroupServiceClient.listUserGroups({});
    resetCache();
    for (const group of groups) {
      groupMapByName.set(group.name, group);
    }
    return groups;
  };

  const getOrFetchGroupByEmail = async (email: string) => {
    if (!hasWorkspacePermissionV2(currentUser.value, "bb.userGroups.get")) {
      return;
    }

    if (getGroupByEmail(email)) {
      return getGroupByEmail(email);
    }

    const group = await userGroupServiceClient.getUserGroup({
      name: getGroupName(email),
    });
    groupMapByName.set(group.name, group);
    return group;
  };

  const createGroup = async (group: UserGroup) => {
    const resp = await userGroupServiceClient.createUserGroup({ group });
    groupMapByName.set(resp.name, resp);
    return resp;
  };

  const deleteGroup = async (name: string) => {
    await userGroupServiceClient.deleteUserGroup({ name });
    groupMapByName.delete(name);
  };

  const updateGroup = async (group: UserGroup, updateMask: string[]) => {
    const updated = await userGroupServiceClient.updateUserGroup({
      group,
      updateMask,
    });
    groupMapByName.set(updated.name, updated);
    return updated;
  };

  return {
    groupList,
    fetchGroupList,
    getGroupByEmail,
    getGroupByIdentifier,
    getOrFetchGroupByEmail,
    deleteGroup,
    updateGroup,
    createGroup,
  };
});
