import { defineStore } from "pinia";
import axios from "axios";
import { VCSId, OAuthToken, ResourceObject, VCSType } from "@/types";

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
    // exchangeVCSTokenWithID uses "vcsId" to let the backend infer the details from an existing VCS provider.
    async exchangeVCSTokenWithID({
      vcsId,
      code,
    }: {
      vcsId: VCSId;
      code: string;
    }): Promise<OAuthToken> {
      const data = (
        await axios.post(`/api/oauth/vcs/exchange-token`, {
          data: {
            type: "exchangeToken",
            attributes: {
              code,
              vcsId,
            },
          },
        })
      ).data.data;

      const token = convert(data);
      return token;
    },

    // exchangeVCSToken passes client details to the backend to allow the backend directly
    // compose the request to the VCS host. This should only be used in the initial VCS set up.
    async exchangeVCSToken({
      vcsType,
      instanceUrl,
      redirectUrl,
      clientId,
      clientSecret,
      code,
    }: {
      vcsType: VCSType;
      instanceUrl: string;
      redirectUrl: string;
      clientId: string;
      clientSecret: string;
      code: string;
    }): Promise<OAuthToken> {
      const data = (
        await axios.post(`/api/oauth/vcs/exchange-token`, {
          data: {
            type: "exchangeToken",
            attributes: {
              vcsType,
              instanceUrl,
              redirectUrl,
              clientId,
              clientSecret,
              code,
            },
          },
        })
      ).data.data;

      const token = convert(data);
      return token;
    },
  },
});
