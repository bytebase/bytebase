import { defineStore } from "pinia";
import axios from "axios";
import {
  VCSId,
  VCS,
  VCSCreate,
  VCSState,
  ResourceObject,
  unknown,
  VCSPatch,
  empty,
  EMPTY_ID,
} from "../../types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(vcs: ResourceObject, includedList: ResourceObject[]): VCS {
  return {
    ...(vcs.attributes as Omit<VCS, "id" | "creator" | "updater">),
    id: parseInt(vcs.id),
    creator: getPrincipalFromIncludedList(
      vcs.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      vcs.relationships!.updater.data,
      includedList
    ),
  };
}

export const useVcsStore = defineStore("vcs", {
  state: (): VCSState => ({
    vcsById: new Map(),
    // repositoryListByVCSId: new Map(),
  }),

  actions: {
    convert(vcs: ResourceObject, includedList: ResourceObject[]): VCS {
      return convert(vcs, includedList);
    },

    getVCSList(): VCS[] {
      const list = [];
      for (const [_, vcs] of this.vcsById) {
        list.push(vcs);
      }
      return list;
    },

    getVCSById(vcsId: VCSId): VCS {
      if (vcsId == EMPTY_ID) {
        return empty("VCS") as VCS;
      }

      return this.vcsById.get(vcsId) || (unknown("VCS") as VCS);
    },

    setVCSList(vcsList: VCS[]) {
      vcsList.forEach((vcs) => {
        this.vcsById.set(vcs.id, vcs);
      });
    },

    setVCSById({ vcsId, vcs }: { vcsId: VCSId; vcs: VCS }) {
      this.vcsById.set(vcsId, vcs);
    },

    async fetchVCSList() {
      const path = "/api/vcs";
      const data = (await axios.get(path)).data;
      const vcsList = data.data
        .map((vcs: ResourceObject) => {
          return convert(vcs, data.included);
        })
        .sort((a: VCS, b: VCS) => {
          return b.createdTs - a.createdTs;
        });

      this.setVCSList(vcsList);

      return vcsList;
    },

    async fetchVCSById(vcsId: VCSId) {
      const data = (await axios.get(`/api/vcs/${vcsId}`)).data;
      const vcs = convert(data.data, data.included);

      this.setVCSById({
        vcsId,
        vcs,
      });
      return vcs;
    },

    async createVCS(newVCS: VCSCreate) {
      const data = (
        await axios.post(`/api/vcs`, {
          data: {
            type: "VCSCreate",
            attributes: newVCS,
          },
        })
      ).data;
      const createdVCS = convert(data.data, data.included);

      this.setVCSById({
        vcsId: createdVCS.id,
        vcs: createdVCS,
      });

      return createdVCS;
    },

    async patchVCS({ vcsId, vcsPatch }: { vcsId: VCSId; vcsPatch: VCSPatch }) {
      const data = (
        await axios.patch(`/api/vcs/${vcsId}`, {
          data: {
            type: "VCSPatch",
            attributes: vcsPatch,
          },
        })
      ).data;
      const updatedVCS = convert(data.data, data.included);

      this.setVCSById({
        vcsId: updatedVCS.id,
        vcs: updatedVCS,
      });

      return updatedVCS;
    },

    async deleteVCSById(vcsId: VCSId) {
      await axios.delete(`/api/vcs/${vcsId}`);

      this.vcsById.delete(vcsId);
    },
  },
});
