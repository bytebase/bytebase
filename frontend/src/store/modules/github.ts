import axios from "axios";
import {
  ExternalRepositoryInfo,
  VCS,
  OAuthConfig,
  OAuthToken,
} from "../../types";

const getters = {};

function convertGitHubProject(project: any): ExternalRepositoryInfo {
  return {
    externalId: project.id.toString(),
    name: project.name,
    fullPath: project.path_with_namespace,
    webUrl: project.web_url,
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
        // FIXME: This is damn unsafe, only OK with development, a real solution must use backend to make reqeusts to the GitHub.com
        `https://cors-anywhere.herokuapp.com/${oAuthConfig.endpoint}?client_id=${oAuthConfig.applicationId}&client_secret=${oAuthConfig.secret}&code=${code}&redirect_uri=${oAuthConfig.redirectUrl}&grant_type=authorization_code`
      )
    ).data;

    // cors-anywhere.herokuapp.com does not pass headers, so we only have URL parameters right now
    const params = new URLSearchParams(data);
    const accessToken = params.get("access_token");
    const oAuthToken: OAuthToken = {
      accessToken: accessToken == null ? "" : accessToken,
      expiresTs: 0,
      refreshToken: "",
    };
    return oAuthToken;
  },

  async fetchProjectList(
    {}: any,
    { vcs, token }: { vcs: VCS; token: string }
  ): Promise<ExternalRepositoryInfo[]> {
    // FIXME: Send request to backend to circumvent CORS
    return [
      {
        externalId: "470746482",
        name: "bytebase-test",
        fullPath: "unknwon/bytebase-test",
        webUrl: "https://github.com/unknwon/bytebase-test",
      },
    ];
    // const data = (
    //   await axios.get(`https://api.github.com/user/repos`, {
    //     headers: {
    //       Authorization: "Bearer " + token,
    //     },
    //   })
    // ).data;

    // // TODO: Filter out repositories without write permissions
    // // "permissions":{"admin":true,"maintain":true,"push":true,"triage":true,"pull":true}
    // return data.map((item: any) => convertGitHubProject(item));
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
