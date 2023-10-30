import { defineStore } from "pinia";
import { reactive } from "vue";
import { changelistServiceClient } from "@/grpcweb";
import {
  Changelist,
  Changelist_Change as Change,
  CreateChangelistRequest,
  DeepPartial,
  ListChangelistsRequest,
} from "@/types/proto/v1/changelist_service";
import {
  ResourceComposer,
  isBranchChangeSource,
  isChangeHistoryChangeSource,
} from "@/utils";
import { useSchemaDesignStore } from "../schemaDesign";
import { useChangeHistoryStore } from "./changeHistory";
import { useSheetV1Store } from "./sheet";

export const useChangelistStore = defineStore("changelist", () => {
  const changelistMapByName = reactive(new Map<string, Changelist>());

  const upsertChangelistMap = async (
    changelists: Changelist[],
    compose: boolean
  ) => {
    for (let i = 0; i < changelists.length; i++) {
      const changelist = changelists[i];
      if (compose) {
        await composeChangelist(changelist);
      }
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
    await upsertChangelistMap([changelist], true /* compose */);
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
    await upsertChangelistMap([created], true /* compose */);
    return created;
  };

  const fetchChangelists = async (
    request: DeepPartial<ListChangelistsRequest>
  ) => {
    const response = await changelistServiceClient.listChangelists(request);
    await upsertChangelistMap(response.changelists, false /* !compose */);
    return response;
  };

  const patchChangelist = async (
    changelist: DeepPartial<Changelist>,
    updateMask: string[]
  ) => {
    const updated = await changelistServiceClient.updateChangelist({
      changelist,
      updateMask,
    });
    await upsertChangelistMap([updated], true /* compose */);
    return updated;
  };

  const deleteChangelist = async (name: string) => {
    await changelistServiceClient.deleteChangelist({
      name,
    });
    changelistMapByName.delete(name);
  };

  const composeChangelist = async (changelist: Changelist) => {
    const composer = new ResourceComposer();
    const { changes } = changelist;
    for (let i = 0; i < changes.length; i++) {
      composeChange(changes[i], composer);
    }

    await composer.compose();
  };
  const composeChange = (change: Change, composer: ResourceComposer) => {
    const { sheet, source } = change;
    if (isChangeHistoryChangeSource(change)) {
      composer.collect(source, () =>
        useChangeHistoryStore().getOrFetchChangeHistoryByName(source)
      );
    } else if (isBranchChangeSource(change)) {
      composer.collect(source, () =>
        useSchemaDesignStore().fetchSchemaDesignByName(
          source,
          true /* useCache */
        )
      );
    } else {
      // Raw SQL, no need to compose
    }
    composer.collect(sheet, () =>
      useSheetV1Store().getOrFetchSheetByName(sheet)
    );
  };

  return {
    getChangelistByName,
    fetchChangelistByName,
    getOrFetchChangelistByName,
    createChangelist,
    fetchChangelists,
    patchChangelist,
    deleteChangelist,
  };
});
