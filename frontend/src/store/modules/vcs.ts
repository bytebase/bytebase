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

function convert(vcs: ResourceObject): VCS {
  return {
    ...(vcs.attributes as Omit<VCS, "id">),
    id: parseInt(vcs.id),
  };
}

const state: () => VCSState = () => ({
  vcsById: new Map(),
  repositoryListByVCSId: new Map(),
});

const getters = {
  convert:
    (state: VCSState, getters: any, rootState: any, rootGetters: any) =>
    (vcs: ResourceObject): VCS => {
      return convert(vcs);
    },

  vcsList: (state: VCSState) => (): VCS[] => {
    const list = [];
    for (const [_, vcs] of state.vcsById) {
      list.push(vcs);
    }
    return list;
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
  async fetchVCSList({ commit }: any) {
    const path = "/api/vcs";
    const data = (await axios.get(path)).data;
    const vcsList = data.data
      .map((vcs: ResourceObject) => {
        return convert(vcs);
      })
      .sort((a: VCS, b: VCS) => {
        return b.createdTs - a.createdTs;
      });

    commit("setVCSList", vcsList);

    return vcsList;
  },

  async fetchVCSById({ commit }: any, vcsId: VCSId) {
    const data = (await axios.get(`/api/vcs/${vcsId}`)).data;
    const vcs = convert(data.data);

    commit("setVCSById", {
      vcsId,
      vcs,
    });
    return vcs;
  },

  async createVCS({ commit }: any, newVCS: VCSCreate) {
    const data = (
      await axios.post(`/api/vcs`, {
        data: {
          type: "VCSCreate",
          attributes: newVCS,
        },
      })
    ).data;
    const createdVCS = convert(data.data);

    commit("setVCSById", {
      vcsId: createdVCS.id,
      vcs: createdVCS,
    });

    return createdVCS;
  },

  async patchVCS(
    { commit }: any,
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
          type: "VCSCreate",
          attributes: vcsPatch,
        },
      })
    ).data;
    const updatedVCS = convert(data.data);

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
