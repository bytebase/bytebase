import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { groupServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { isValidProjectName } from "@/types";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import {
  BatchGetGroupsRequestSchema,
  CreateGroupRequestSchema,
  DeleteGroupRequestSchema,
  GetGroupRequestSchema,
  ListGroupsRequestSchema,
  UpdateGroupRequestSchema,
} from "@/types/proto-es/v1/group_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { groupNamePrefix } from "./common";

export interface GroupFilter {
  query?: string;
  project?: string;
}

const getListGroupFilter = (params: GroupFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(title.matches("${search}") || email.matches("${search}"))`);
  }
  if (isValidProjectName(params.project)) {
    filter.push(`project == "${params.project}"`);
  }

  return filter.join(" && ");
};

export const extractGroupEmail = (emailResource: string) => {
  const matches = emailResource.match(/^(?:group:|groups\/)(.+)$/);
  return matches?.[1] ?? emailResource;
};

export const ensureGroupIdentifier = (id: string) => {
  const email = extractGroupEmail(id);
  return `${groupNamePrefix}${email}`;
};

export const useGroupStore = defineStore("group", () => {
  const groupMapByName = reactive(new Map<string, Group>());

  // Getters
  const groupList = computed(() => {
    return orderBy(
      Array.from(groupMapByName.values()),
      (group) => group.name,
      "asc"
    );
  });

  const getGroupByIdentifier = (id: string) => {
    return groupMapByName.get(ensureGroupIdentifier(id));
  };

  const fetchGroupList = async (params: {
    pageSize: number;
    pageToken?: string;
    filter?: GroupFilter;
  }): Promise<{
    groups: Group[];
    nextPageToken: string;
  }> => {
    const request = create(ListGroupsRequestSchema, {
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      filter: getListGroupFilter(params.filter ?? {}),
    });
    // Ignore errors and silent the request.
    const { groups, nextPageToken } =
      await groupServiceClientConnect.listGroups(request, {
        contextValues: createContextValues().set(silentContextKey, true),
      });
    for (const group of groups) {
      groupMapByName.set(group.name, group);
    }
    return { groups, nextPageToken };
  };

  const batchFetchGroups = async (groupNameList: string[]) => {
    if (groupNameList.length === 0) {
      return [];
    }
    const request = create(BatchGetGroupsRequestSchema, {
      names: groupNameList.map(ensureGroupIdentifier),
    });
    const { groups } = await groupServiceClientConnect.batchGetGroups(request, {
      contextValues: createContextValues().set(silentContextKey, true),
    });
    for (const group of groups) {
      groupMapByName.set(group.name, group);
    }
    return groups;
  };

  const batchGetOrFetchGroups = async (groupNames: string[]) => {
    const validGroupList = uniq(groupNames).filter((groupName) => !!groupName);
    const pendingFetch = validGroupList.filter((groupName) => {
      const group = getGroupByIdentifier(groupName);
      if (group) {
        return false;
      }
      return true;
    });
    await batchFetchGroups(pendingFetch);
    return validGroupList.map(getGroupByIdentifier);
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
    groupMapByName.set(group.name, group);
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
    groupMapByName.set(response.name, response);
    return response;
  };

  const updateGroup = async (group: Group) => {
    const request = create(UpdateGroupRequestSchema, {
      group: group,
      updateMask: { paths: ["title", "description", "members"] },
      allowMissing: false,
    });
    const response = await groupServiceClientConnect.updateGroup(request);
    groupMapByName.set(response.name, response);
    return response;
  };

  return {
    groupList,
    fetchGroupList,
    batchGetOrFetchGroups,
    getGroupByIdentifier,
    getOrFetchGroupByIdentifier,
    deleteGroup,
    updateGroup,
    createGroup,
  };
});
