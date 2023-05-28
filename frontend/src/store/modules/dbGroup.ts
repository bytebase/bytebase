import { projectServiceClient } from "@/grpcweb";
import { DatabaseGroup } from "@/types/proto/v1/project_service";
import { defineStore } from "pinia";

const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapById = new Map<string, DatabaseGroup>();
  const dbGroupListMapByProjectId = new Map<string, DatabaseGroup[]>();

  const getOrFetchDBGroupListByProjectId = async (projectId: string) => {
    const cached = dbGroupListMapByProjectId.get(projectId);
    if (cached) return cached;

    const dbGroupList = await projectServiceClient(projectId);
    dbGroupListMapByProjectId.set(projectId, dbGroupList);
    return dbGroupList;
  };

  return {
    getOrFetchDBGroupListByProjectId,
  };
});
