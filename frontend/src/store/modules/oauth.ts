import axios from "axios";
import { VCSId, OAuthToken, ResourceObject } from "../../types";

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
    }: {
      vcsId: VCSId;
      code: string;
    }
  ): Promise<OAuthToken> {
    const data = (
      await axios.post(`/api/oauth/vcs/${vcsId}/exchange-token`, null, {
        headers: { code },
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
