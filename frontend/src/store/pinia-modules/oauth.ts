import axios from "axios";
import { VCSId, OAuthToken, ResourceObject, VCSType } from "@/types";
import { defineStore } from "pinia";

const convert = (raw: ResourceObject): OAuthToken => {
  const attr = raw.attributes;
  return {
    accessToken: attr.accessToken as string,
    refreshToken: attr.refreshToken as string,
    expiresTs: attr.expiresTs as number,
  };
};

export const useOAuthStore = defineStore("oauth", {
  actions: {
    // Either pass the "vcsId" to let the backend infer the details from an existing VCS provider
    // or pass "vcsType", "instanceURL", "clientId" and "clientSecret" to allow the backend directly
    // compose the request to the VCS host.
    async exchangeVCSToken({
      code,
      vcsId,
      vcsType,
      instanceUrl,
      clientId,
      clientSecret,
    }: {
      code: string;
      vcsId?: VCSId;
      vcsType?: VCSType;
      instanceUrl?: string;
      clientId?: string;
      clientSecret?: string;
    }): Promise<OAuthToken> {
      const data = (
        await axios.post(`/api/oauth/vcs/exchange-token`, {
          data: {
            type: "exchangeToken",
            attributes: {
              code,
              vcsId,
              vcsType,
              instanceUrl,
              clientId,
              clientSecret,
            },
          },
        })
      ).data.data;

      const token = convert(data);
      return token;
    },
  },
});
