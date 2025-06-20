import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { groupServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { Group } from "@/types/proto/v1/group_service";
import {
  CreateGroupRequestSchema,
  DeleteGroupRequestSchema,
  GetGroupRequestSchema,
  ListGroupsRequestSchema,
  UpdateGroupRequestSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { convertNewGroupToOld, convertOldGroupToNew } from "@/utils/v1/group-conversions";
import { useUserStore } from "../user";
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
    await useUserStore().batchGetUsers(group.members.map((m) => m.member));
    groupMapByName.set(group.name, group);
  };

  const getGroupByIdentifier = (id: string) => {
    return groupMapByName.get(ensureGroupIdentifier(id));
  };

  const fetchGroupList = async () => {
    if (!hasWorkspacePermissionV2("bb.groups.list")) {
      return [];
    }

    const request = create(ListGroupsRequestSchema, {});
    const { groups } = await groupServiceClientConnect.listGroups(request);
    resetCache();
    const oldGroups = groups.map(convertNewGroupToOld);
    for (const group of oldGroups) {
      await composeGroup(group);
    }
    return oldGroups;
  };

  const getOrFetchGroupByIdentifier = async (id: string) => {
    if (!hasWorkspacePermissionV2("bb.groups.get")) {
      return;
    }

    const existed = getGroupByIdentifier(id);
    if (existed) {
      return existed;
    }

    const request = create(GetGroupRequestSchema, {
      name: ensureGroupIdentifier(id),
    });
    const newGroup = await groupServiceClientConnect.getGroup(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
    const group = convertNewGroupToOld(newGroup);
    await composeGroup(group);
    return group;
  };

  const deleteGroup = async (name: string) => {
    const request = create(DeleteGroupRequestSchema, { name });
    await groupServiceClientConnect.deleteGroup(request);
    groupMapByName.delete(name);
  };

  const createGroup = async (group: Group) => {
    const newGroup = convertOldGroupToNew(group);
    const request = create(CreateGroupRequestSchema, {
      group: newGroup,
      groupEmail: extractGroupEmail(group.name),
    });
    const response = await groupServiceClientConnect.createGroup(request);
    const oldGroup = convertNewGroupToOld(response);
    await composeGroup(oldGroup);
    return oldGroup;
  };

  const updateGroup = async (group: Group) => {
    const newGroup = convertOldGroupToNew(group);
    const request = create(UpdateGroupRequestSchema, {
      group: newGroup,
      updateMask: { paths: ["title", "description", "members"] },
      allowMissing: false,
    });
    const response = await groupServiceClientConnect.updateGroup(request);
    const updated = convertNewGroupToOld(response);
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
