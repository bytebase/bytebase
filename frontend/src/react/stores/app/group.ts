import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { groupServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { isValidProjectName } from "@/react/lib/resourceName";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import {
  BatchGetGroupsRequestSchema,
  CreateGroupRequestSchema,
  DeleteGroupRequestSchema,
  GetGroupRequestSchema,
  ListGroupsRequestSchema,
  UpdateGroupRequestSchema,
} from "@/types/proto-es/v1/group_service_pb";
import type { AppSliceCreator, GroupFilter, GroupSlice } from "./types";
import { toError } from "./utils";

const groupNamePrefix = "groups/";

export const extractGroupEmail = (emailResource: string) => {
  const matches = emailResource.match(/^(?:group:|groups\/)(.+)$/);
  return matches?.[1] ?? emailResource;
};

export const ensureGroupIdentifier = (id: string) => {
  const email = extractGroupEmail(id);
  return `${groupNamePrefix}${email}`;
};

export const buildGroupFilter = (params: GroupFilter) => {
  const filter = [];
  const search = params.query?.trim()?.toLowerCase();
  if (search) {
    filter.push(`(title.contains("${search}") || email.contains("${search}"))`);
  }
  if (isValidProjectName(params.project)) {
    filter.push(`project == "${params.project}"`);
  }
  return filter.join(" && ");
};

export const createGroupSlice: AppSliceCreator<GroupSlice> = (set, get) => ({
  groupsByName: {},
  groupRequests: {},
  groupErrorsByName: {},

  listGroups: async ({ pageSize, pageToken, filter }) => {
    if (!get().hasWorkspacePermission("bb.groups.list")) {
      return { groups: [], nextPageToken: "" };
    }
    const response = await groupServiceClientConnect.listGroups(
      createProto(ListGroupsRequestSchema, {
        pageSize,
        pageToken,
        filter: buildGroupFilter(filter ?? {}),
      }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    set((state) => ({
      groupsByName: {
        ...state.groupsByName,
        ...Object.fromEntries(
          response.groups.map((group) => [group.name, group])
        ),
      },
    }));
    return { groups: response.groups, nextPageToken: response.nextPageToken };
  },

  batchFetchGroups: async (names) => {
    const validNames = uniq(names).filter(Boolean).map(ensureGroupIdentifier);
    if (validNames.length === 0) return [];
    const response = await groupServiceClientConnect.batchGetGroups(
      createProto(BatchGetGroupsRequestSchema, { names: validNames }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    set((state) => ({
      groupsByName: {
        ...state.groupsByName,
        ...Object.fromEntries(
          response.groups.map((group) => [group.name, group])
        ),
      },
    }));
    return response.groups;
  },

  batchGetOrFetchGroups: async (names) => {
    const validNames = uniq(names).filter(Boolean).map(ensureGroupIdentifier);
    const missing = validNames.filter((name) => !get().groupsByName[name]);
    if (missing.length > 0) {
      await get().batchFetchGroups(missing);
    }
    return validNames.map((name) => get().groupsByName[name]);
  },

  fetchGroup: async (id) => {
    if (!get().hasWorkspacePermission("bb.groups.get")) return undefined;
    const name = ensureGroupIdentifier(id);
    const existing = get().groupsByName[name];
    if (existing) return existing;
    const pending = get().groupRequests[name];
    if (pending) return pending;

    const request = groupServiceClientConnect
      .getGroup(createProto(GetGroupRequestSchema, { name }), {
        contextValues: createContextValues().set(silentContextKey, true),
      })
      .then((group: Group) => {
        set((state) => {
          const { [name]: _, ...groupRequests } = state.groupRequests;
          return {
            groupsByName: { ...state.groupsByName, [group.name]: group },
            groupErrorsByName: {
              ...state.groupErrorsByName,
              [name]: undefined,
            },
            groupRequests,
          };
        });
        return group;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...groupRequests } = state.groupRequests;
          return {
            groupErrorsByName: {
              ...state.groupErrorsByName,
              [name]: toError(error),
            },
            groupRequests,
          };
        });
        return undefined;
      });
    set((state) => ({
      groupRequests: { ...state.groupRequests, [name]: request },
    }));
    return request;
  },

  getGroupByIdentifier: (id) => get().groupsByName[ensureGroupIdentifier(id)],

  createGroup: async (group) => {
    const response = await groupServiceClientConnect.createGroup(
      createProto(CreateGroupRequestSchema, {
        group,
        groupEmail: extractGroupEmail(group.name),
      })
    );
    set((state) => ({
      groupsByName: { ...state.groupsByName, [response.name]: response },
    }));
    return response;
  },

  updateGroup: async (group) => {
    const response = await groupServiceClientConnect.updateGroup(
      createProto(UpdateGroupRequestSchema, {
        group,
        updateMask: { paths: ["title", "description", "members"] },
        allowMissing: false,
      })
    );
    set((state) => ({
      groupsByName: { ...state.groupsByName, [response.name]: response },
    }));
    return response;
  },

  deleteGroup: async (name) => {
    await groupServiceClientConnect.deleteGroup(
      createProto(DeleteGroupRequestSchema, { name })
    );
    set((state) => {
      const { [name]: _, ...groupsByName } = state.groupsByName;
      return { groupsByName };
    });
  },
});
