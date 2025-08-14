import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { groupServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import {
  CreateGroupRequestSchema,
  DeleteGroupRequestSchema,
  GetGroupRequestSchema,
  ListGroupsRequestSchema,
  UpdateGroupRequestSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
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
    const request = create(ListGroupsRequestSchema, {});
    // Ignore errors and silent the request.
    const { groups } = await groupServiceClientConnect.listGroups(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
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

    const request = create(GetGroupRequestSchema, {
      name: ensureGroupIdentifier(id),
    });
    const group = await groupServiceClientConnect.getGroup(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
    await composeGroup(group);
    return group;
  };

  const deleteGroup = async (name: string) => {
    const request = create(DeleteGroupRequestSchema, { name });
    await groupServiceClientConnect.deleteGroup(request);
    groupMapByName.delete(name);
  };

  const createGroup = async (group: Group) => {
    const request = create(CreateGroupRequestSchema, {
      group: group,
      groupEmail: extractGroupEmail(group.name),
    });
    const response = await groupServiceClientConnect.createGroup(request);
    await composeGroup(response);
    return response;
  };

  const updateGroup = async (group: Group) => {
    const request = create(UpdateGroupRequestSchema, {
      group: group,
      updateMask: { paths: ["title", "description", "members"] },
      allowMissing: false,
    });
    const response = await groupServiceClientConnect.updateGroup(request);
    await composeGroup(response);
    return response;
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

export const useGroupList = () => {
  const groupStore = useGroupStore();
  return computed(() => groupStore.groupList);
};
