import axios from "axios";
import {
  VCSId,
  VCS,
  VCSCreate,
  VCSState,
  ResourceObject,
  ResourceIdentifier,
  unknown,
  VCSPatch,
  empty,
  EMPTY_ID,
  Principal,
  Repository,
  Project,
} from "../../types";

function convertRepository(
  respository: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Repository {
  const creator = respository.attributes.creator as Principal;
  const updater = respository.attributes.updater as Principal;

  const projectId = (
    respository.relationships!.project.data as ResourceIdentifier
  ).id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectId);

  for (const item of includedList || []) {
    if (item.type == "project" && item.id == projectId) {
      project = rootGetters["project/convert"](item, includedList);
    }
  }

  return {
    ...(respository.attributes as Omit<
      Repository,
      "id" | "creator" | "updater" | "project"
    >),
    id: parseInt(respository.id),
    creator,
    updater,
    project,
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

  repositoryListByVCSId:
    (state: VCSState) =>
    (vcsId: VCSId): Repository[] => {
      return state.repositoryListByVCSId.get(vcsId) || [];
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

  async fetchRepositoryListByVCS({ commit, rootGetters }: any, vcs: VCS) {
    const data = (await axios.get(`/api/vcs/${vcs.id}/repository`)).data;

    const repositoryList = data.data.map((repository: ResourceObject) => {
      return convertRepository(repository, data.included, rootGetters);
    });

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
