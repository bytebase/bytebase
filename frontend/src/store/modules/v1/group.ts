import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { groupServiceClient } from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import type { Group } from "@/types/proto/v1/group";
import { hasWorkspacePermissionV2 } from "@/utils";
import { groupNamePrefix } from "./common";

export const extractGroupEmail = (emailResource: string) => {
  const matches = emailResource.match(/^(?:group:|groups\/)(.+)$/);
  return matches?.[1] ?? emailResource;
};

export const useGroupStore = defineStore("group", () => {
  const currentUser = useCurrentUserV1();
  const groupMapByName = reactive(new Map<string, Group>());
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
  const getGroupName = (email: string) => `${groupNamePrefix}${email}`;

  const getGroupByEmail = (email: string) => {
    return groupMapByName.get(getGroupName(email));
  };

  const getGroupByIdentifier = (id: string) => {
    return getGroupByEmail(extractGroupEmail(id));
  };

  const fetchGroupList = async () => {
    if (!hasWorkspacePermissionV2(currentUser.value, "bb.groups.list")) {
      return [];
    }

    const { groups } = await groupServiceClient.listGroups({});
    resetCache();
    for (const group of groups) {
      groupMapByName.set(group.name, group);
    }
    return groups;
  };

  const getOrFetchGroupByEmail = async (email: string) => {
    if (!hasWorkspacePermissionV2(currentUser.value, "bb.groups.get")) {
      return;
    }

    if (getGroupByEmail(email)) {
      return getGroupByEmail(email);
    }

    const group = await groupServiceClient.getGroup({
      name: getGroupName(email),
    });
    groupMapByName.set(group.name, group);
    return group;
  };

  const createGroup = async (group: Group) => {
    const resp = await groupServiceClient.createGroup({ group });
    groupMapByName.set(resp.name, resp);
    return resp;
  };

  const deleteGroup = async (name: string) => {
    await groupServiceClient.deleteGroup({ name });
    groupMapByName.delete(name);
  };

  const updateGroup = async (group: Group, updateMask: string[]) => {
    const updated = await groupServiceClient.updateGroup({
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
