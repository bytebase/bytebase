import axios from "axios";
import { ExternalRepositoryInfo, VCS, OAuthToken } from "../../types";

const getters = {};

function convertGitLabProject(project: any): ExternalRepositoryInfo {
  const attributes = project.attributes;
  return {
    externalId: project.id.toString(),
    name: attributes.name,
    fullPath: attributes.fullPath,
    webUrl: attributes.webUrl,
  };
}

const actions = {
  // TODO(zilong): here we still store the access token at the frontend, we may move this to the backend
  async fetchProjectList(
    {}: any,
    { vcs, token }: { vcs: VCS; token: OAuthToken }
  ): Promise<ExternalRepositoryInfo[]> {
    const data = (
      await axios.get(`/api/vcs/${vcs.id}/external-repository`, {
        headers: {
          accessToken: token.accessToken,
          refreshToken: token.refreshToken,
        },
      })
    ).data.data;
    return data.map((item: any) => convertGitLabProject(item));
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
