<template>
  <BBAttention
    v-if="showAttention"
    type="warning"
    :description="attentionText"
    :link="link"
  />
  <StepTab
    class="mt-4 mb-8"
    :current-index="state.currentStep"
    :step-list="stepList"
    :allow-next="allowNext"
    :show-cancel="showCancel"
    :finish-title="$t('common.confirm-and-add')"
    @update:current-index="tryChangeStep"
    @finish="tryFinishSetup"
    @cancel="cancelSetup"
  >
    <template #0>
      <VCSProviderBasicInfoPanel :config="state.config" />
    </template>
    <template #1>
      <VCSProviderOAuthPanel :config="state.config" />
    </template>
  </StepTab>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { StepTab } from "@/components/v2";
import { WORKSPACE_ROUTE_GITOPS } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useVCSV1Store } from "@/store";
import { VCSConfig } from "@/types";
import {
  VCSProvider,
  VCSProvider_Type,
} from "@/types/proto/v1/vcs_provider_service";
import VCSProviderBasicInfoPanel from "./VCSProviderBasicInfoPanel.vue";
import VCSProviderOAuthPanel from "./VCSProviderOAuthPanel.vue";

withDefaults(
  defineProps<{
    showCancel?: boolean;
  }>(),
  {
    showCancel: true,
  }
);

const CONFIRM_STEP = 1;

interface LocalState {
  config: VCSConfig;
  currentStep: number;
}

const { t } = useI18n();
const router = useRouter();
const vcsV1Store = useVCSV1Store();

const stepList = [
  { title: t("gitops.setting.add-git-provider.basic-info.self") },
  { title: t("common.confirm") },
];

const state = reactive<LocalState>({
  config: {
    type: VCSProvider_Type.GITLAB,
    uiType: "GITLAB_SELF_HOST",
    name: t("gitops.setting.add-git-provider.gitlab-self-host"),
    instanceUrl: "",
    accessToken: "",
  },
  currentStep: 0,
});

const allowNext = computed((): boolean => {
  return true;
});

const attentionText = computed((): string => {
  if (state.config.type === VCSProvider_Type.GITLAB) {
    if (state.config.uiType == "GITLAB_SELF_HOST") {
      return t(
        "gitops.setting.add-git-provider.gitlab-self-host-admin-requirement"
      );
    }
    return t("gitops.setting.add-git-provider.gitlab-com-admin-requirement");
  } else if (state.config.type === VCSProvider_Type.GITHUB) {
    return t("gitops.setting.add-git-provider.github-com-admin-requirement");
  } else if (state.config.type === VCSProvider_Type.BITBUCKET) {
    return t("gitops.setting.add-git-provider.bitbucket-admin-requirement");
  } else if (state.config.type === VCSProvider_Type.AZURE_DEVOPS) {
    return t("gitops.setting.add-git-provider.azure-admin-requirement");
  }
  return "";
});

const link = computed((): string => {
  if (state.config.type === VCSProvider_Type.GITLAB) {
    if (state.config.uiType == "GITLAB_SELF_HOST") {
      return "https://www.bytebase.com/docs/vcs-integration/self-host-gitlab/?source=console";
    }
    return "https://www.bytebase.com/docs/vcs-integration/gitlab-com/?source=console";
  } else if (state.config.type === VCSProvider_Type.GITHUB) {
    if (state.config.uiType == "GITHUB_COM") {
      return "https://www.bytebase.com/docs/vcs-integration/github-com/?source=console";
    }
    return "https://www.bytebase.com/docs/vcs-integration/github-enterprise/?source=console";
  } else if (state.config.type === VCSProvider_Type.BITBUCKET) {
    return "https://www.bytebase.com/docs/vcs-integration/bitbucket-org/?source=console";
  } else if (state.config.type === VCSProvider_Type.AZURE_DEVOPS) {
    return "https://www.bytebase.com/docs/vcs-integration/azure-devops/?source=console";
  }
  return "";
});

const showAttention = computed((): boolean => {
  return state.currentStep != CONFIRM_STEP;
});

const tryChangeStep = (nextStepIndex: number) => {
  state.currentStep = nextStepIndex;
};

const tryFinishSetup = () => {
  vcsV1Store
    .createVCS({
      name: "",
      title: state.config.name,
      type: state.config.type,
      url: state.config.instanceUrl,
      accessToken: state.config.accessToken,
    })
    .then((vcs: VCSProvider) => {
      router.push({
        name: WORKSPACE_ROUTE_GITOPS,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("gitops.setting.add-git-provider.add-success", {
          vcs: vcs.title,
        }),
      });
    });
};

const cancelSetup = () => {
  router.push({
    name: WORKSPACE_ROUTE_GITOPS,
  });
};
</script>
