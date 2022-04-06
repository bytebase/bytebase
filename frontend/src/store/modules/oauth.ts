import axios from "axios";
import { VCSId, OAuthToken, ResourceObject, VCSType } from "../../types";

const convert = (raw: ResourceObject): OAuthToken => {
  const attr = raw.attributes;
  return {
    accessToken: attr.accessToken as string,
    refreshToken: attr.refreshToken as string,
    expiresTs: attr.expiresTs as number,
  };
};

const actions = {
  // Either pass the "vcsId" to let the backend infer the details from an exsiting VCS provider
  // or pass "vcsType", "instanceURL", "clientId", "clientSecret" to make the backend only to
  // pass on the request to the code host.
  async exchangeVCSToken(
    {}: any,
    {
      code,
      vcsId,
      vcsType,
      instanceURL,
      clientId,
      clientSecret,
    }: {
      code: string;
      vcsId?: VCSId;
      vcsType?: VCSType;
      instanceURL?: string;
      clientId?: string;
      clientSecret?: string;
    }
  ): Promise<OAuthToken> {
    const data = (
      await axios.post(`/api/oauth/vcs/exchange-token`, {
        data: {
          type: "exchangeToken",
          attributes: {
            code,
            vcsId,
            vcsType,
            instanceURL,
            clientId,
            clientSecret,
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
