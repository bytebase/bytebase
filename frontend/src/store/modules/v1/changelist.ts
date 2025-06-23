import { defineStore } from "pinia";
import { reactive } from "vue";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { changelistServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type {
  Changelist,
  Changelist_Change as Change,
  CreateChangelistRequest,
  DeepPartial,
  ListChangelistsRequest,
} from "@/types/proto/v1/changelist_service";
import {
  GetChangelistRequestSchema,
  CreateChangelistRequestSchema,
  ListChangelistsRequestSchema,
  UpdateChangelistRequestSchema,
  DeleteChangelistRequestSchema,
} from "@/types/proto-es/v1/changelist_service_pb";
import { ResourceComposer, isChangelogChangeSource } from "@/utils";
import {
  convertNewChangelistToOld,
  convertOldChangelistToNew,
} from "@/utils/v1/changelist-conversions";
import { useUserStore } from "../user";
import { useChangelogStore } from "./changelog";
import { useSheetV1Store } from "./sheet";

export const useChangelistStore = defineStore("changelist", () => {
  const changelistMapByName = reactive(new Map<string, Changelist>());

  const upsertChangelistMap = async (
    changelists: Changelist[],
    compose: boolean
  ) => {
    await useUserStore().batchGetUsers(
      changelists.map((changelist) => changelist.creator)
    );
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
    const request = create(GetChangelistRequestSchema, { name });
    const changelist = await changelistServiceClientConnect.getChangelist(
      request,
      { contextValues: createContextValues().set(silentContextKey, silent) }
    );
    const oldChangelist = convertNewChangelistToOld(changelist);
    await upsertChangelistMap([oldChangelist], true /* compose */);
    return oldChangelist;
  };

  const getOrFetchChangelistByName = async (name: string, silent = false) => {
    const cachedData = changelistMapByName.get(name);
    if (cachedData) {
      return cachedData;
    }

    return fetchChangelistByName(name, silent);
  };

  const createChangelist = async (request: CreateChangelistRequest) => {
    // Convert the changelist in the request if it exists
    const connectRequest = create(CreateChangelistRequestSchema, {
      ...request,
      changelist: request.changelist ? convertOldChangelistToNew(request.changelist) : undefined,
    });
    const created = await changelistServiceClientConnect.createChangelist(connectRequest);
    const oldCreated = convertNewChangelistToOld(created);
    await upsertChangelistMap([oldCreated], true /* compose */);
    return oldCreated;
  };

  const fetchChangelists = async (
    request: DeepPartial<ListChangelistsRequest>
  ) => {
    const connectRequest = create(ListChangelistsRequestSchema, request);
    const response = await changelistServiceClientConnect.listChangelists(connectRequest);
    const oldChangelists = response.changelists.map(convertNewChangelistToOld);
    await upsertChangelistMap(oldChangelists, false /* !compose */);
    return {
      ...response,
      changelists: oldChangelists,
    };
  };

  const patchChangelist = async (
    changelist: DeepPartial<Changelist>,
    updateMask: string[]
  ) => {
    const connectRequest = create(UpdateChangelistRequestSchema, {
      changelist: changelist as any, // Type assertion needed for partial type compatibility
      updateMask: { paths: updateMask },
    });
    const updated = await changelistServiceClientConnect.updateChangelist(connectRequest);
    const oldUpdated = convertNewChangelistToOld(updated);
    await upsertChangelistMap([oldUpdated], true /* compose */);
    return oldUpdated;
  };

  const deleteChangelist = async (name: string) => {
    const connectRequest = create(DeleteChangelistRequestSchema, { name });
    await changelistServiceClientConnect.deleteChangelist(connectRequest);
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
    if (isChangelogChangeSource(change)) {
      composer.collect(source, () =>
        useChangelogStore().getOrFetchChangelogByName(source)
      );
    } else {
      // Raw SQL, no need to compose
    }
    composer.collect(sheet, () =>
      // Use any (basic or full) view of sheets here to save data size
      useSheetV1Store().getOrFetchSheetByName(sheet)
    );
  };

  return {
    getChangelistByName,
    getOrFetchChangelistByName,
    createChangelist,
    fetchChangelists,
    patchChangelist,
    deleteChangelist,
  };
});
