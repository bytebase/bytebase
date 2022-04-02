import axios from "axios";
import {
  ExternalRepositoryInfo,
  VCS,
  OAuthConfig,
  OAuthToken,
} from "../../types";

const getters = {};

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
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
