import axios from "axios";
import { VCSId, OAuthToken, ResourceObject } from "@/types";
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
    async exchangeVCSToken({ vcsId, code }: { vcsId: VCSId; code: string }) {
      const data = (
        await axios.post(`/api/oauth/vcs/${vcsId}/exchange-token`, null, {
          headers: { code },
        })
      ).data.data;

      const token = convert(data);
      return token;
    },
  },
});
