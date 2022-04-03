import axios from "axios";
import { VCSId, OAuthConfig, OAuthToken, ResourceObject } from "../../types";

const convert = (raw: ResourceObject): OAuthToken => {
  const attr = raw.attributes;
  return {
    accessToken: attr.accessToken as string,
    refreshToken: attr.refreshToken as string,
    expiresTs: attr.expiresTs as number,
  };
};

const actions = {
  async exchangeVCSToken(
    {}: any,
    {
      vcsId,
      code,
      oauthConfig,
    }: {
      vcsId: VCSId;
      code: string;
      oauthConfig: OAuthConfig;
    }
  ): Promise<OAuthToken> {
    const data = (
      await axios.post(`/api/oauth/vcs/exchange-token`, {
        data: {
          type: "exchangeToken",
          attributes: {
            vcsId,
            code,
            config: oauthConfig,
          },
        },
      })
    ).data.data;

    const token = convert(data);
    return token;
  },
};
export default {
  namespaced: true,
  actions,
};
