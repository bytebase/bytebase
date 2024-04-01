import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { VCSUIType } from "@/types";
import { VCSType } from "@/types/proto/v1/common";

export const vcsListByUIType = computed(
  (): {
    type: VCSType;
    uiType: VCSUIType;
    title: string;
  }[] => {
    const { t } = useI18n();

    return [
      {
        type: VCSType.GITLAB,
        uiType: "GITLAB_SELF_HOST",
        title: t("gitops.setting.add-git-provider.gitlab-self-host"),
      },
      {
        type: VCSType.GITLAB,
        uiType: "GITLAB_COM",
        title: "GitLab.com",
      },
      {
        type: VCSType.GITHUB,
        uiType: "GITHUB_COM",
        title: "GitHub.com",
      },
      {
        type: VCSType.GITHUB,
        uiType: "GITHUB_ENTERPRISE",
        title: t("gitops.setting.add-git-provider.github-self-host"),
      },
      {
        type: VCSType.AZURE_DEVOPS,
        uiType: "AZURE_DEVOPS",
        title: t("gitops.setting.add-git-provider.azure-devops-service"),
      },
      {
        type: VCSType.BITBUCKET,
        uiType: "BITBUCKET_ORG",
        title: "Bitbucket.org",
      },
    ];
  }
);
