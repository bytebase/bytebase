import { uniqBy } from "lodash-es";
import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { vcsProviderServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import type { VCSId } from "@/types";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { vcsProviderPrefix } from "./common";

type VCSProviderCacheKey = [string /* vcs name */];

export const useVCSV1Store = defineStore("vcs_v1", () => {
  const cacheByName = useCache<VCSProviderCacheKey, VCSProvider | undefined>(
    "bb.vcs-provider.by-name"
  );

  const vcsList = computed(() => {
    const list = Array.from(cacheByName.entityCacheMap.values())
      .map((entry) => entry.entity)
      .filter((vcs): vcs is VCSProvider => vcs !== undefined);
    return uniqBy(list, (vcs) => vcs.name);
  });

  const searchVCSProviderRepositories = async (vcsName: string) => {
    const resp = await vcsProviderServiceClient.searchVCSProviderRepositories({
      name: vcsName,
    });
    return resp.repositories;
  };

  const fetchVCSList = async () => {
    const resp = await vcsProviderServiceClient.listVCSProviders({});
    return resp.vcsProviders;
  };

  const getOrFetchVCSList = async () => {
    if (vcsList.value.length > 0) {
      return vcsList.value;
    }
    const list = await fetchVCSList();
    for (const vcs of list) {
      cacheByName.setEntity([vcs.name], vcs);
    }
    return list;
  };

  const fetchVCSByName = async (name: string, silent = false) => {
    const vcs = await vcsProviderServiceClient.getVCSProvider(
      {
        name,
      },
      { silent }
    );

    return vcs;
  };

  const getOrFetchVCSByName = async (name: string) => {
    const entity = cacheByName.getEntity([name]);
    if (entity) {
      return entity;
    }

    const vcs = await vcsProviderServiceClient.getVCSProvider({
      name,
    });
    cacheByName.setEntity([vcs.name], vcs);
    return vcs;
  };

  const getVCSById = (vcsId: VCSId) => {
    return getVCSByName(`${vcsProviderPrefix}${vcsId}`);
  };

  const getVCSByName = (vcs: string) => {
    return cacheByName.getEntity([vcs]);
  };

  const deleteVCS = async (name: string) => {
    await vcsProviderServiceClient.deleteVCSProvider({
      name,
    });
    cacheByName.invalidateEntity([name]);
  };

  const createVCS = async (resourceId: string, vcs: VCSProvider) => {
    const provider = await vcsProviderServiceClient.createVCSProvider({
      vcsProvider: vcs,
      vcsProviderId: resourceId,
    });
    cacheByName.setEntity([provider.name], provider);
    return provider;
  };

  const updateVCS = async (vcs: Partial<VCSProvider>) => {
    if (!vcs.name) {
      return;
    }
    const existed = await fetchVCSByName(vcs.name);
    const updateMask = getUpdateMaskForVCS(existed, vcs);
    if (updateMask.length === 0) {
      return existed;
    }
    const provider = await vcsProviderServiceClient.updateVCSProvider({
      vcsProvider: vcs,
      updateMask,
    });
    cacheByName.setEntity([provider.name], provider);
    return provider;
  };

  return {
    vcsList,
    searchVCSProviderRepositories,
    getVCSById,
    getVCSByName,
    getOrFetchVCSByName,
    getOrFetchVCSList,
    deleteVCS,
    createVCS,
    updateVCS,
  };
});

const getUpdateMaskForVCS = (
  origin: VCSProvider,
  update: Partial<VCSProvider>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (
    !isUndefined(update.accessToken) &&
    !isEqual(origin.accessToken, update.accessToken)
  ) {
    updateMask.push("access_token");
  }
  return updateMask;
};
