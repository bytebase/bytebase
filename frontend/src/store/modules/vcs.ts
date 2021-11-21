import axios from "axios";
import {
  VCSID,
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
  vcsByID: new Map(),
  repositoryListByVCSID: new Map(),
});

const getters = {
  convert:
    (state: VCSState, getters: any, rootState: any, rootGetters: any) =>
    (vcs: ResourceObject): VCS => {
      return convert(vcs);
    },

  vcsList: (state: VCSState) => (): VCS[] => {
    const list = [];
    for (const [_, vcs] of state.vcsByID) {
      list.push(vcs);
    }
    return list;
  },

  vcsByID:
    (state: VCSState) =>
    (vcsID: VCSID): VCS => {
      if (vcsID == EMPTY_ID) {
        return empty("VCS") as VCS;
      }

      return state.vcsByID.get(vcsID) || (unknown("VCS") as VCS);
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

  async fetchVCSByID({ commit }: any, vcsID: VCSID) {
    const data = (await axios.get(`/api/vcs/${vcsID}`)).data;
    const vcs = convert(data.data);

    commit("setVCSByID", {
      vcsID,
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

    commit("setVCSByID", {
      vcsID: createdVCS.id,
      vcs: createdVCS,
    });

    return createdVCS;
  },

  async patchVCS(
    { commit }: any,
    {
      vcsID,
      vcsPatch,
    }: {
      vcsID: VCSID;
      vcsPatch: VCSPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/vcs/${vcsID}`, {
        data: {
          type: "VCSCreate",
          attributes: vcsPatch,
        },
      })
    ).data;
    const updatedVCS = convert(data.data);

    commit("setVCSByID", {
      vcsID: updatedVCS.id,
      vcs: updatedVCS,
    });

    return updatedVCS;
  },

  async deleteVCSByID(
    { commit }: { state: VCSState; commit: any },
    vcsID: VCSID
  ) {
    await axios.delete(`/api/vcs/${vcsID}`);

    commit("deleteVCSByID", vcsID);
  },
};

const mutations = {
  setVCSList(state: VCSState, vcsList: VCS[]) {
    vcsList.forEach((vcs) => {
      state.vcsByID.set(vcs.id, vcs);
    });
  },

  setVCSByID(
    state: VCSState,
    {
      vcsID,
      vcs,
    }: {
      vcsID: VCSID;
      vcs: VCS;
    }
  ) {
    state.vcsByID.set(vcsID, vcs);
  },

  deleteVCSByID(state: VCSState, vcsID: VCSID) {
    state.vcsByID.delete(vcsID);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
