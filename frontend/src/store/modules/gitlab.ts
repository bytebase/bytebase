import axios from "axios";
import {
  ExternalRepositoryInfo,
  VCS,
  OAuthConfig,
  OAuthToken,
} from "../../types";

const GITLAB_API_PATH = "api/v4";

const getters = {};

function convertGitLabProject(project: any): ExternalRepositoryInfo {
  return {
    externalId: project.id.toString(),
    name: project.name,
    fullPath: project.path_with_namespace,
    webURL: project.web_url,
  };
}

const actions = {
  async exchangeToken(
    {}: any,
    {
      oAuthConfig,
      code,
    }: {
      oAuthConfig: OAuthConfig;
      code: string;
    }
  ): Promise<OAuthToken> {
    const data = (
      await axios.post(
        `${oAuthConfig.endpoint}?client_id=${oAuthConfig.applicationId}&client_secret=${oAuthConfig.secret}&code=${code}&redirect_uri=${oAuthConfig.redirectURL}&grant_type=authorization_code`
      )
    ).data;

    const oAuthToken: OAuthToken = {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      // For GitLab, as of 13.12, the default config won't expire the access token, thus this field is 0.
      // see https://gitlab.com/gitlab-org/gitlab/-/issues/21745.
      expiresTs: data.expires_in == 0 ? 0 : data.created_at + data.expires_in,
    };
    return oAuthToken;
  },

  async fetchProjectList(
    {}: any,
    { vcs, token }: { vcs: VCS; token: string }
  ): Promise<ExternalRepositoryInfo[]> {
    // We will use user's token to create webhook in the project, which requires the token owner to
    // be at least the project maintainer(40)
    const data = (
      await axios.get(
        `${vcs.instanceURL}/${GITLAB_API_PATH}/projects?membership=true&simple=true&min_access_level=40`,
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
