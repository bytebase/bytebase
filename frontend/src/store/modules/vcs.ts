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
  Principal,
} from "../../types";

function convert(
  vcs: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): VCS {
  const creator = vcs.attributes.creator as Principal;
  const updater = vcs.attributes.updater as Principal;
  return {
    ...(vcs.attributes as Omit<VCS, "id" | "creator" | "updater">),
    id: parseInt(vcs.id),
    creator,
    updater,
  };
}

const state: () => VCSState = () => ({
  vcsById: new Map(),
});

const getters = {
  convert:
    (state: VCSState, getters: any, rootState: any, rootGetters: any) =>
    (vcs: ResourceObject, includedList: ResourceObject[]): VCS => {
      return convert(vcs, includedList, rootGetters);
    },

  vcsList: (state: VCSState) => (): VCS[] => {
    const list = [];
    for (const [_, vcs] of state.vcsById) {
      list.push(vcs);
    }
    return list.sort((a: VCS, b: VCS) => {
      return b.createdTs - a.createdTs;
    });
  },

  vcsById:
    (state: VCSState) =>
    (vcsId: VCSId): VCS => {
      if (vcsId == EMPTY_ID) {
        return empty("VCS") as VCS;
      }

      return state.vcsById.get(vcsId) || (unknown("VCS") as VCS);
    },
};

const actions = {
  async fetchVCSList({ commit, rootGetters }: any) {
    const path = "/api/vcs";
    const data = (await axios.get(path)).data;
    const vcsList = data.data.map((vcs: ResourceObject) => {
      return convert(vcs, data.included, rootGetters);
    });

    commit("setVCSList", vcsList);

    return vcsList;
  },

  async fetchVCSById({ commit, rootGetters }: any, vcsId: VCSId) {
    // include=secret to return username/password when requesting the specific vcs id
    const data = (await axios.get(`/api/vcs/${vcsId}`)).data;
    const vcs = convert(data.data, data.included, rootGetters);

    commit("setVCSById", {
      vcsId,
      vcs,
    });
    return vcs;
  },

  async createVCS({ commit, rootGetters }: any, newVCS: VCSCreate) {
    const data = (
      await axios.post(`/api/vcs`, {
        data: {
          type: "VCSCreate",
          attributes: newVCS,
        },
      })
    ).data;
    const createdVCS = convert(data.data, data.included, rootGetters);

    commit("setVCSById", {
      vcsId: createdVCS.id,
      vcs: createdVCS,
    });

    return createdVCS;
  },

  async patchVCS(
    { commit, rootGetters }: any,
    {
      vcsId,
      vcsPatch,
    }: {
      vcsId: VCSId;
      vcsPatch: VCSPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/vcs/${vcsId}`, {
        data: {
          type: "vcsPatch",
          attributes: vcsPatch,
        },
      })
    ).data;
    const updatedVCS = convert(data.data, data.included, rootGetters);

    commit("setVCSById", {
      vcsId: updatedVCS.id,
      vcs: updatedVCS,
    });

    return updatedVCS;
  },

  async deleteVCSById(
    { commit }: { state: VCSState; commit: any },
    vcsId: VCSId
  ) {
    await axios.delete(`/api/vcs/${vcsId}`);

    commit("deleteVCSById", vcsId);
  },
};

const mutations = {
  setVCSList(state: VCSState, vcsList: VCS[]) {
    vcsList.forEach((vcs) => {
      state.vcsById.set(vcs.id, vcs);
    });
  },

  setVCSById(
    state: VCSState,
    {
      vcsId,
      vcs,
    }: {
      vcsId: VCSId;
      vcs: VCS;
    }
  ) {
    state.vcsById.set(vcsId, vcs);
  },

  deleteVCSById(state: VCSState, vcsId: VCSId) {
    state.vcsById.delete(vcsId);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
