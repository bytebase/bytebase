import axios from "axios";
import { ExternalRepository, VCS, OAuthConfig } from "../../types";

const GITLAB_API_PATH = "api/v4";
const GITLAB_WEBHOOK_PATH = "hook/gitlab";

const getters = {};

function convertGitLabProject(project: any): ExternalRepository {
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
  ): Promise<string> {
    console.log(
      "req",
      `${oAuthConfig.endpoint}?client_id=${oAuthConfig.applicationId}&client_secret=${oAuthConfig.secret}&code=${code}&redirect_uri=${oAuthConfig.redirectURL}&grant_type=authorization_code`
    );
    const data = (
      await axios.post(
        `${oAuthConfig.endpoint}?client_id=${oAuthConfig.applicationId}&client_secret=${oAuthConfig.secret}&code=${code}&redirect_uri=${oAuthConfig.redirectURL}&grant_type=authorization_code`
      )
    ).data;
    return data.access_token;
  },

  async fetchProjectList(
    {}: any,
    { vcs, token }: { vcs: VCS; token: string }
  ): Promise<ExternalRepository[]> {
    console.log(
      "req",
      `${vcs.instanceURL}/${GITLAB_API_PATH}/projects?membership=true&simple=true`
    );
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

  // Create webhook to receive push event
  async createWebhook(
    {}: any,
    {
      vcs,
      projectId,
      branchFilter,
      token,
    }: { vcs: VCS; projectId: string; branchFilter: string; token: string }
  ): Promise<string> {
    const data = (
      await axios.post(
        `${vcs.instanceURL}/${GITLAB_API_PATH}/projects/${projectId}/hooks`,
        {
          url: `${vcs.instanceURL}/${GITLAB_WEBHOOK_PATH}`,
          push_events: true,
          // For now, there is no native dry run DDL support in mysql/postgres. One may wonder if we could wrap the DDL
          // in a transaction and just not commit at the end, unfortunately there are side effects which are hard to control.
          // See https://www.postgresql.org/message-id/CAMsr%2BYGiYQ7PYvYR2Voio37YdCpp79j5S%2BcmgVJMOLM2LnRQcA%40mail.gmail.com
          // So we can't possibly display useful info when reviewing a MR, thus we don't enable this event.
          // Saying that, delivering a souding dry run solution would be great and hopefully we can achieve that one day.
          // merge_requests_events: true,
          push_events_branch_filter: branchFilter,
          // TODO: Be lax for now
          enable_ssl_verification: false,
        },
        {
          headers: {
            Authorization: "Bearer " + token,
          },
        }
      )
    ).data;

    return data.id.toString();
  },
};

const mutations = {};

export default {
  namespaced: true,
  getters,
  actions,
  mutations,
};
