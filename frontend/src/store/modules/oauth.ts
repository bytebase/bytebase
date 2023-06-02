import { defineStore } from "pinia";
import axios from "axios";
import { VCSId, OAuthToken, ResourceObject } from "@/types";
import {
  ExternalVersionControl_Type,
  externalVersionControl_TypeToJSON,
} from "@/types/proto/v1/externalvs_service";

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
      code,
      vcsId,
      clientId,
      clientSecret,
    }: {
      code: string;
      vcsId: VCSId;
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
              clientId,
              clientSecret,
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
      clientId,
      clientSecret,
      code,
    }: {
      vcsType: ExternalVersionControl_Type;
      instanceUrl: string;
      clientId: string;
      clientSecret: string;
      code: string;
    }): Promise<OAuthToken> {
      const data = (
        await axios.post(`/api/oauth/vcs/exchange-token`, {
          data: {
            type: "exchangeToken",
            attributes: {
              // TODO(ed): change vcs type
              vcsType: externalVersionControl_TypeToJSON(vcsType),
              instanceUrl,
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
