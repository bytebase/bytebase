import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { groupServiceClient } from "@/grpcweb";
import type { Group } from "@/types/proto/v1/group_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { batchGetOrFetchUsers } from "../user";
import { groupNamePrefix } from "./common";

export const extractGroupEmail = (emailResource: string) => {
  const matches = emailResource.match(/^(?:group:|groups\/)(.+)$/);
  return matches?.[1] ?? emailResource;
};

const ensureGroupIdentifier = (id: string) => {
  const email = extractGroupEmail(id);
  return `${groupNamePrefix}${email}`;
};

export const useGroupStore = defineStore("group", () => {
  const groupMapByName = reactive(new Map<string, Group>());
  const resetCache = () => {
    groupMapByName.clear();
  };

  // Getters
  const groupList = computed(() => {
    return orderBy(
      Array.from(groupMapByName.values()),
      (group) => group.name,
      "asc"
    );
  });

  const composeGroup = async (group: Group) => {
    await batchGetOrFetchUsers(group.members.map((m) => m.member));
    groupMapByName.set(group.name, group);
  };

  const getGroupByIdentifier = (id: string) => {
    return groupMapByName.get(ensureGroupIdentifier(id));
  };

  const fetchGroupList = async () => {
    if (!hasWorkspacePermissionV2("bb.groups.list")) {
      return [];
    }

    const { groups } = await groupServiceClient.listGroups({});
    resetCache();
    for (const group of groups) {
      await composeGroup(group);
    }
    return groups;
  };

  const getOrFetchGroupByIdentifier = async (id: string) => {
    if (!hasWorkspacePermissionV2("bb.groups.get")) {
      return;
    }

    const existed = getGroupByIdentifier(id);
    if (existed) {
      return existed;
    }

    const group = await groupServiceClient.getGroup(
      {
        name: ensureGroupIdentifier(id),
      },
      { silent: true }
    );
    await composeGroup(group);
    return group;
  };

  const deleteGroup = async (name: string) => {
    await groupServiceClient.deleteGroup({ name });
    groupMapByName.delete(name);
  };

  const createGroup = async (group: Group) => {
    const resp = await groupServiceClient.createGroup({
      group,
      groupEmail: extractGroupEmail(group.name),
    });
    await composeGroup(resp);
    return resp;
  };

  const updateGroup = async (group: Group) => {
    const updated = await groupServiceClient.updateGroup({
      group,
      updateMask: ["title", "description", "members"],
      allowMissing: false,
    });
    await composeGroup(updated);
    return updated;
  };

  return {
    groupList,
    fetchGroupList,
    getGroupByIdentifier,
    getOrFetchGroupByIdentifier,
    deleteGroup,
    updateGroup,
    createGroup,
  };
});
