import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { accessGrantServiceClientConnect } from "@/connect";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import {
  AccessGrant_Status,
  ActivateAccessGrantRequestSchema,
  CreateAccessGrantRequestSchema,
  ListAccessGrantsRequestSchema,
  RevokeAccessGrantRequestSchema,
  SearchMyAccessGrantsRequestSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";

export interface AccessFilter {
  name?: string;
  statement?: string;
  creator?: string;
  status?: AccessGrant_Status;
  issue?: string;
  target?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
  expireTsAfter?: number;
  expireTsBefore?: number;
}

const getListAccessFilter = (filter: AccessFilter | undefined): string => {
  if (!filter) return "";
  const parts: string[] = [];

  if (filter.name) {
    parts.push(`name == "${filter.name}"`);
  }
  if (
    filter.status !== undefined &&
    filter.status !== AccessGrant_Status.STATUS_UNSPECIFIED
  ) {
    parts.push(`status == "${AccessGrant_Status[filter.status]}"`);
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
  if (filter.expireTsAfter !== undefined) {
    parts.push(
      `expire_time >= "${new Date(filter.expireTsAfter).toISOString()}"`
    );
  }
  if (filter.expireTsBefore !== undefined) {
    parts.push(
      `expire_time <= "${new Date(filter.expireTsBefore).toISOString()}"`
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
    searchMyAccessGrants,
    createAccessGrant,
    listAccessGrants,
    activateAccessGrant,
    revokeAccessGrant,
  };
});
