import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
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
import { type AccessGrantFilterStatus } from "@/utils";

export interface AccessFilter {
  name?: string;
  statement?: string;
  creator?: string;
  status?: AccessGrantFilterStatus[];
  issue?: string;
  target?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
}

const getListAccessFilter = (filter: AccessFilter | undefined): string => {
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
            `(status == "${AccessGrant_Status[AccessGrant_Status.ACTIVE]}" && expire_time > "${new Date().toISOString()}")`
          );
          break;
        case "EXPIRED":
          statusFilter.push(`expire_time <= "${new Date().toISOString()}"`);
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
    parts.push(`query.matches("${filter.statement.trim()}")`);
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

  return parts.join(" && ");
};

interface ListAccessGrantsParams {
  parent: string;
  filter?: AccessFilter;
  pageSize?: number;
  pageToken?: string;
  orderBy?: string;
}

export const useAccessGrantStore = defineStore("accessGrant", () => {
  const getAccessGrant = async (name: string) => {
    return await accessGrantServiceClientConnect.getAccessGrant(
      create(GetAccessGrantRequestSchema, { name })
    );
  };

  const searchMyAccessGrants = async (params: ListAccessGrantsParams) => {
    return await accessGrantServiceClientConnect.searchMyAccessGrants(
      create(SearchMyAccessGrantsRequestSchema, {
        parent: params.parent,
        filter: getListAccessFilter(params.filter),
        pageSize: params.pageSize ?? 0,
        pageToken: params.pageToken ?? "",
        orderBy: params.orderBy ?? "",
      })
    );
  };

  const createAccessGrant = async (
    parent: string,
    accessGrant: AccessGrant
  ) => {
    return await accessGrantServiceClientConnect.createAccessGrant(
      create(CreateAccessGrantRequestSchema, {
        parent,
        accessGrant,
      })
    );
  };

  const listAccessGrants = async (params: ListAccessGrantsParams) => {
    return await accessGrantServiceClientConnect.listAccessGrants(
      create(ListAccessGrantsRequestSchema, {
        parent: params.parent,
        filter: getListAccessFilter(params.filter),
        pageSize: params.pageSize ?? 0,
        pageToken: params.pageToken ?? "",
        orderBy: params.orderBy ?? "",
      })
    );
  };

  const activateAccessGrant = async (name: string) => {
    return await accessGrantServiceClientConnect.activateAccessGrant(
      create(ActivateAccessGrantRequestSchema, { name })
    );
  };

  const revokeAccessGrant = async (name: string) => {
    return await accessGrantServiceClientConnect.revokeAccessGrant(
      create(RevokeAccessGrantRequestSchema, { name })
    );
  };

  return {
    getAccessGrant,
    searchMyAccessGrants,
    createAccessGrant,
    listAccessGrants,
    activateAccessGrant,
    revokeAccessGrant,
  };
});
