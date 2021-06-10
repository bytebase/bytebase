import axios from "axios";
import { Repository, VCS } from "../../types";

const GITLAB_API_PATH = "api/v4";

const getters = {};

function convertGitLabProject(project: any): Repository {
  return {
    id: project.id,
    name: project.name,
    fullPath: project.path_with_namespace,
    webURL: project.web_url,
  };
}

const actions = {
  async exchangeToken(
    {}: any,
    { vcs, code, redirectURL }: { vcs: VCS; code: string; redirectURL: string }
  ): Promise<string> {
    console.log(
      "req",
      `${vcs.instanceURL}/oauth/token?client_id=${vcs.applicationId}&client_secret=${vcs.secret}&code=${code}&redirect_uri=${redirectURL}&grant_type=authorization_code`
    );
    const data = (
      await axios.post(
        `${vcs.instanceURL}/oauth/token?client_id=${vcs.applicationId}&client_secret=${vcs.secret}&code=${code}&redirect_uri=${redirectURL}&grant_type=authorization_code`
      )
    ).data;
    return data.access_token;
  },

  async fetchProjectList(
    {}: any,
    { vcs, token }: { vcs: VCS; token: string }
  ): Promise<Repository[]> {
    console.log(
      "req",
      `${vcs.instanceURL}/${GITLAB_API_PATH}/projects?membership=true&simple=true`
    );
    const data = (
      await axios.get(
        `${vcs.instanceURL}/${GITLAB_API_PATH}/projects?membership=true&simple=true`,
        {
          headers: {
            Authorization: "Bearer " + token,
          },
        }
      )
    ).data;

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
