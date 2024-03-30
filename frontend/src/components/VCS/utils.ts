import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { VCSUIType } from "@/types";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";

export const vcsListByUIType = computed(
  (): {
    type: VCSProvider_Type;
    uiType: VCSUIType;
    title: string;
  }[] => {
    const { t } = useI18n();

    return [
      {
        type: VCSProvider_Type.GITLAB,
        uiType: "GITLAB_SELF_HOST",
        title: t("gitops.setting.add-git-provider.gitlab-self-host"),
      },
      {
        type: VCSProvider_Type.GITLAB,
        uiType: "GITLAB_COM",
        title: "GitLab.com",
      },
      {
        type: VCSProvider_Type.GITHUB,
        uiType: "GITHUB_COM",
        title: "GitHub.com",
      },
      {
        type: VCSProvider_Type.GITHUB,
        uiType: "GITHUB_ENTERPRISE",
        title: t("gitops.setting.add-git-provider.github-self-host"),
      },
      {
        type: VCSProvider_Type.AZURE_DEVOPS,
        uiType: "AZURE_DEVOPS",
        title: t("gitops.setting.add-git-provider.azure-devops-service"),
      },
      {
        type: VCSProvider_Type.BITBUCKET,
        uiType: "BITBUCKET_ORG",
        title: "Bitbucket.org",
      },
    ];
  }
);
