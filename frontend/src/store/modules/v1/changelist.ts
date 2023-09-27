import { defineStore } from "pinia";
import { reactive } from "vue";
import { changelistServiceClient } from "@/grpcweb";
import {
  Changelist,
  CreateChangelistRequest,
  DeepPartial,
  ListChangelistsRequest,
} from "@/types/proto/v1/changelist_service";

export const useChangelistStore = defineStore("changelist", () => {
  const changelistMapByName = reactive(new Map<string, Changelist>());

  const upsertChangelistMap = async (changelists: Changelist[]) => {
    for (let i = 0; i < changelists.length; i++) {
      const changelist = changelists[i];
      changelistMapByName.set(changelist.name, changelist);
    }
  };

  const getChangelistByName = (name: string) => {
    return changelistMapByName.get(name);
  };

  const fetchChangelistByName = async (name: string, silent = false) => {
    const changelist = await changelistServiceClient.getChangelist(
      { name },
      { silent }
    );
    await upsertChangelistMap([changelist]);
    return changelist;
  };

  const getOrFetchChangelistByName = async (name: string, silent = false) => {
    const cachedData = changelistMapByName.get(name);
    if (cachedData) {
      return cachedData;
    }

    return fetchChangelistByName(name, silent);
  };

  const createChangelist = async (request: CreateChangelistRequest) => {
    const created = await changelistServiceClient.createChangelist(request);
    await upsertChangelistMap([created]);
    return created;
  };

  const fetchChangelists = async (
    request: DeepPartial<ListChangelistsRequest>
  ) => {
    const response = await changelistServiceClient.listChangelists(request);
    await upsertChangelistMap(response.changelists);
    return response;
  };

  return {
    getChangelistByName,
    fetchChangelistByName,
    getOrFetchChangelistByName,
    createChangelist,
    fetchChangelists,
  };
});
