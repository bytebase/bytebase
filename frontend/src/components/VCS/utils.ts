import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { VCSUIType } from "@/types";
import { ExternalVersionControl_Type } from "@/types/proto/v1/externalvs_service";

export const vcsListByUIType = computed(
  (): {
    type: ExternalVersionControl_Type;
    uiType: VCSUIType;
    title: string;
  }[] => {
    const { t } = useI18n();

    return [
      {
        type: ExternalVersionControl_Type.GITLAB,
        uiType: "GITLAB_SELF_HOST",
        title: t("gitops.setting.add-git-provider.gitlab-self-host"),
      },
      {
        type: ExternalVersionControl_Type.GITLAB,
        uiType: "GITLAB_COM",
        title: "GitLab.com",
      },
      {
        type: ExternalVersionControl_Type.GITHUB,
        uiType: "GITHUB_COM",
        title: "GitHub.com",
      },
      {
        type: ExternalVersionControl_Type.GITHUB,
        uiType: "GITHUB_ENTERPRISE",
        title: t("gitops.setting.add-git-provider.github-self-host-ee"),
      },
      {
        type: ExternalVersionControl_Type.AZURE_DEVOPS,
        uiType: "AZURE_DEVOPS",
        title: t("gitops.setting.add-git-provider.azure-devops-service"),
      },
      {
        type: ExternalVersionControl_Type.BITBUCKET,
        uiType: "BITBUCKET_ORG",
        title: "Bitbucket.org",
      },
    ];
  }
);
