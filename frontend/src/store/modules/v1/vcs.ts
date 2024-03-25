import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { vcsProviderServiceClient } from "@/grpcweb";
import { VCSId } from "@/types";
import { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { vcsProviderPrefix } from "./common";

export const useVCSV1Store = defineStore("vcs_v1", () => {
  const vcsMapByName = reactive(new Map<string, VCSProvider>());

  const listVCSExternalProjects = async (vcsName: string) => {
    const resp = await vcsProviderServiceClient.searchVCSProviderProjects({
      name: vcsName,
    });
    return resp.projects;
  };

  const fetchVCSList = async () => {
    const resp = await vcsProviderServiceClient.listVCSProviders({});
    for (const vcs of resp.vcsProviders) {
      vcsMapByName.set(vcs.name, vcs);
    }
    return resp.vcsProviders;
  };

  const fetchVCSByName = async (name: string, silent = false) => {
    const vcs = await vcsProviderServiceClient.getVCSProvider({
      name,
    });

    vcsMapByName.set(vcs.name, vcs);
    return vcs;
  };

  const fetchVCSById = async (vcsId: VCSId) => {
    return fetchVCSByName(`${vcsProviderPrefix}${vcsId}`);
  };

  const getVCSById = (vcsId: VCSId) => {
    return vcsMapByName.get(`${vcsProviderPrefix}${vcsId}`);
  };

  const getVCSList = () => {
    return [...vcsMapByName.values()];
  };

  const deleteVCS = async (name: string) => {
    await vcsProviderServiceClient.deleteVCSProvider({
      name,
    });
    vcsMapByName.delete(name);
  };

  const createVCS = async (resourceId: string, vcs: VCSProvider) => {
    const resp = await vcsProviderServiceClient.createVCSProvider({
      vcsProvider: vcs,
      vcsProviderId: resourceId,
    });
    vcsMapByName.set(resp.name, resp);
    return resp;
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
    const resp = await vcsProviderServiceClient.updateVCSProvider({
      vcsProvider: vcs,
      updateMask,
    });
    vcsMapByName.set(resp.name, resp);
    return resp;
  };

  return {
    listVCSExternalProjects,
    getVCSById,
    getVCSList,
    fetchVCSByName,
    fetchVCSById,
    fetchVCSList,
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
