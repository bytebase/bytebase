import { create as createProto } from "@bufbuild/protobuf";
import { accessGrantServiceClientConnect } from "@/connect";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import {
  AccessGrant_Status,
  ActivateAccessGrantRequestSchema,
  CreateAccessGrantRequestSchema,
  GetAccessGrantRequestSchema,
  ListAccessGrantsRequestSchema,
  RevokeAccessGrantRequestSchema,
  SearchMyAccessGrantsRequestSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import type {
  AccessGrantFilter,
  AccessGrantSlice,
  AppSliceCreator,
} from "./types";

export const buildAccessGrantFilter = (
  filter: AccessGrantFilter | undefined,
  now = new Date()
): string => {
  if (!filter) return "";
  const parts: string[] = [];

  if (filter.name) {
    parts.push(`name == "${filter.name}"`);
  }
  if (filter.status !== undefined && filter.status.length > 0) {
    const statusFilter: string[] = [];
    for (const status of filter.status) {
      switch (status) {
        case "ACTIVE":
          statusFilter.push(
            `(status == "${AccessGrant_Status[AccessGrant_Status.ACTIVE]}" && expire_time > "${now.toISOString()}")`
          );
          break;
        case "EXPIRED":
          statusFilter.push(`expire_time <= "${now.toISOString()}"`);
          break;
        default:
          statusFilter.push(
            `status == "${status as keyof typeof AccessGrant_Status}"`
          );
      }
    }
    parts.push(`(${statusFilter.join(" || ")})`);
  }
  if (filter.statement) {
    parts.push(`query.contains("${filter.statement.trim()}")`);
  }
  if (filter.statementExact !== undefined) {
    // Use JSON.stringify so internal quotes / backslashes / newlines in
    // the SQL are escaped safely into the CEL string literal.
    parts.push(`query == ${JSON.stringify(filter.statementExact.trim())}`);
  }
  if (filter.creator) {
    parts.push(`creator == "${filter.creator}"`);
  }
  if (filter.issue) {
    parts.push(`issue == "${filter.issue}"`);
  }
  if (filter.target) {
    parts.push(`target == "${filter.target}"`);
  }
  if (filter.createdTsAfter !== undefined) {
    parts.push(
      `create_time >= "${new Date(filter.createdTsAfter).toISOString()}"`
    );
  }
  if (filter.createdTsBefore !== undefined) {
    parts.push(
      `create_time <= "${new Date(filter.createdTsBefore).toISOString()}"`
    );
  }
  if (filter.unmask !== undefined) {
    parts.push(`unmask == ${filter.unmask}`);
  }
  if (filter.export !== undefined) {
    parts.push(`export == ${filter.export}`);
  }
  return parts.join(" && ");
};

const upsertAccessGrants = (
  set: Parameters<AppSliceCreator<AccessGrantSlice>>[0],
  accessGrants: AccessGrant[]
) => {
  set((state) => ({
    accessGrantsByName: {
      ...state.accessGrantsByName,
      ...Object.fromEntries(accessGrants.map((grant) => [grant.name, grant])),
    },
  }));
};

export const createAccessGrantSlice: AppSliceCreator<AccessGrantSlice> = (
  set,
  get
) => ({
  accessGrantsByName: {},
  accessGrantRequests: {},

  fetchAccessGrant: async (name) => {
    const existing = get().accessGrantsByName[name];
    if (existing) return existing;
    const pending = get().accessGrantRequests[name];
    if (pending) return pending;

    const request = accessGrantServiceClientConnect
      .getAccessGrant(createProto(GetAccessGrantRequestSchema, { name }))
      .then((grant) => {
        set((state) => {
          const { [name]: _, ...accessGrantRequests } =
            state.accessGrantRequests;
          return {
            accessGrantsByName: {
              ...state.accessGrantsByName,
              [grant.name]: grant,
            },
            accessGrantRequests,
          };
        });
        return grant;
      })
      .catch(() => {
        set((state) => {
          const { [name]: _, ...accessGrantRequests } =
            state.accessGrantRequests;
          return { accessGrantRequests };
        });
        return undefined;
      });
    set((state) => ({
      accessGrantRequests: { ...state.accessGrantRequests, [name]: request },
    }));
    return request;
  },

  searchMyAccessGrants: async (params) => {
    const response = await accessGrantServiceClientConnect.searchMyAccessGrants(
      createProto(SearchMyAccessGrantsRequestSchema, {
        parent: params.parent,
        filter: buildAccessGrantFilter(params.filter),
        pageSize: params.pageSize ?? 0,
        pageToken: params.pageToken ?? "",
        orderBy: params.orderBy ?? "",
      })
    );
    upsertAccessGrants(set, response.accessGrants);
    return {
      accessGrants: response.accessGrants,
      nextPageToken: response.nextPageToken,
    };
  },

  listAccessGrants: async (params) => {
    const response = await accessGrantServiceClientConnect.listAccessGrants(
      createProto(ListAccessGrantsRequestSchema, {
        parent: params.parent,
        filter: buildAccessGrantFilter(params.filter),
        pageSize: params.pageSize ?? 0,
        pageToken: params.pageToken ?? "",
        orderBy: params.orderBy ?? "",
      })
    );
    upsertAccessGrants(set, response.accessGrants);
    return {
      accessGrants: response.accessGrants,
      nextPageToken: response.nextPageToken,
    };
  },

  createAccessGrant: async (parent, accessGrant) => {
    const grant = await accessGrantServiceClientConnect.createAccessGrant(
      createProto(CreateAccessGrantRequestSchema, { parent, accessGrant })
    );
    upsertAccessGrants(set, [grant]);
    return grant;
  },

  activateAccessGrant: async (name) => {
    const grant = await accessGrantServiceClientConnect.activateAccessGrant(
      createProto(ActivateAccessGrantRequestSchema, { name })
    );
    upsertAccessGrants(set, [grant]);
    return grant;
  },

  revokeAccessGrant: async (name) => {
    const grant = await accessGrantServiceClientConnect.revokeAccessGrant(
      createProto(RevokeAccessGrantRequestSchema, { name })
    );
    upsertAccessGrants(set, [grant]);
    return grant;
  },
});
