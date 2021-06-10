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
  VCSTokenCreate,
  Repository,
} from "../../types";

function convertRepository(respository: ResourceObject): Repository {
  const creator = respository.attributes.creator as Principal;
  const updater = respository.attributes.updater as Principal;
  return {
    ...(respository.attributes as Omit<
      Repository,
      "id" | "creator" | "updater"
    >),
    id: parseInt(respository.id),
    creator,
    updater,
  };
}

function convert(vcs: ResourceObject): VCS {
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
  repositoryListByVCSId: new Map(),
});

const getters = {
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
  async fetchVCSList({ commit }: any) {
    const path = "/api/vcs";
    const data = (await axios.get(path)).data;
    const vcsList = data.data.map((vcs: ResourceObject) => {
      return convert(vcs);
    });

    commit("setVCSList", vcsList);

    return vcsList;
  },

  async fetchVCSById({ commit }: any, vcsId: VCSId) {
    // include=secret to return username/password when requesting the specific vcs id
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

  async createVCSToken(
    { commit }: any,
    { vcsId, tokenCreate }: { vcsId: VCSId; tokenCreate: VCSTokenCreate }
  ) {
    const data = (
      await axios.post(`/api/vcs/${vcsId}/token`, {
        data: {
          type: "VCSTokenCreate",
          attributes: tokenCreate,
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

  async fetchRepositoryListByVCS({ commit }: any, vcs: VCS) {
    const repositoryList = (
      await axios.get(`/api/vcs/${vcs.id}/repository`)
    ).data.data.map((repository: ResourceObject) => {
      return convertRepository(repository);
    });

    console.log("repo", repositoryList);

    commit("setRepositoryListByVCSId", { vcsId: vcs.id, repositoryList });
    return repositoryList;
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

  setRepositoryListByVCSId(
    state: VCSState,
    {
      vcsId,
      repositoryList,
    }: {
      vcsId: VCSId;
      repositoryList: Repository[];
    }
  ) {
    state.repositoryListByVCSId.set(vcsId, repositoryList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
