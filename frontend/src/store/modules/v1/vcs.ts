import { defineStore } from "pinia";
import { reactive } from "vue";
import { isEqual, isUndefined } from "lodash-es";
import { externalVersionControlServiceClient } from "@/grpcweb";
import { ExternalVersionControl } from "@/types/proto/v1/externalvs_service";
import { externalVersionControlPrefix } from "./common";
import { VCSId } from "@/types";

export const useVCSV1Store = defineStore("vcs_v1", () => {
  const vcsMapByName = reactive(new Map<string, ExternalVersionControl>());

  const fetchVCSList = async () => {
    const resp =
      await externalVersionControlServiceClient.listExternalVersionControls({});
    for (const vcs of resp.externalVersionControls) {
      vcsMapByName.set(vcs.name, vcs);
    }
    return resp.externalVersionControls;
  };

  const fetchVCSByName = async (name: string) => {
    const vcs =
      await externalVersionControlServiceClient.getExternalVersionControl({
        name,
      });

    vcsMapByName.set(vcs.name, vcs);
    return vcs;
  };

  const fetchVCSByUid = async (vcsId: VCSId) => {
    return fetchVCSByName(`${externalVersionControlPrefix}${vcsId}`);
  };

  const getVCSByUid = (vcsId: VCSId) => {
    return vcsMapByName.get(`${externalVersionControlPrefix}${vcsId}`);
  };

  const getVCSList = () => {
    return [...vcsMapByName.values()];
  };

  const deleteVCS = async (name: string) => {
    await externalVersionControlServiceClient.deleteExternalVersionControl({
      name,
    });
  };

  const createVCS = async (vcs: ExternalVersionControl) => {
    const resp =
      await externalVersionControlServiceClient.createExternalVersionControl({
        externalVersionControl: vcs,
      });
    vcsMapByName.set(resp.name, resp);
    return resp;
  };

  const updateVCS = async (vcs: Partial<ExternalVersionControl>) => {
    if (!vcs.name) {
      return;
    }
    const existed = await fetchVCSByName(vcs.name);
    const updateMask = getUpdateMaskForVCS(existed, vcs);
    if (updateMask.length === 0) {
      return existed;
    }
    const resp =
      await externalVersionControlServiceClient.updateExternalVersionControl({
        externalVersionControl: vcs,
      });
    vcsMapByName.set(resp.name, resp);
    return resp;
  };

  return {
    getVCSByUid,
    getVCSList,
    fetchVCSByName,
    fetchVCSByUid,
    fetchVCSList,
    deleteVCS,
    createVCS,
    updateVCS,
  };
});

const getUpdateMaskForVCS = (
  origin: ExternalVersionControl,
  update: Partial<ExternalVersionControl>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (
    !isUndefined(update.applicationId) &&
    !isEqual(origin.applicationId, update.applicationId)
  ) {
    updateMask.push("application_id");
  }
  return updateMask;
};
