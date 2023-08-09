<template>
  <BBAttention
    v-if="showAttention"
    :style="'WARN'"
    :description="attentionText"
  />
  <BBStepTab
    class="mt-4 mb-8"
    :step-item-list="stepList"
    :allow-next="allowNext"
    :finish-title="$t('common.confirm-and-add')"
    @try-change-step="tryChangeStep"
    @try-finish="tryFinishSetup"
    @cancel="cancelSetup"
  >
    <template #0>
      <VCSProviderBasicInfoPanel :config="state.config" />
    </template>
    <template #1>
      <VCSProviderOAuthPanel :config="state.config" />
    </template>
    <template #2>
      <VCSProviderConfirmPanel :config="state.config" />
    </template>
  </BBStepTab>
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import { reactive, computed, onUnmounted, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { pushNotification, useVCSV1Store } from "@/store";
import {
  OAuthToken,
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";
import { BBStepTabItem } from "../bbkit/types";
import {
  isValidVCSApplicationIdOrSecret,
  VCSConfig,
  openWindowForOAuth,
  OAuthWindowEventPayload,
} from "../types";
import { isUrl } from "../utils";
import VCSProviderBasicInfoPanel from "./VCSProviderBasicInfoPanel.vue";
import VCSProviderConfirmPanel from "./VCSProviderConfirmPanel.vue";
import VCSProviderOAuthPanel from "./VCSProviderOAuthPanel.vue";

const BASIC_INFO_STEP = 0;
const OAUTH_INFO_STEP = 1;
const CONFIRM_STEP = 2;

interface LocalState {
  config: VCSConfig;
  currentStep: number;
  oAuthResultCallback?: (token: OAuthToken | undefined) => void;
}

const { t } = useI18n();
const router = useRouter();
const vcsV1Store = useVCSV1Store();

const stepList: BBStepTabItem[] = [
  { title: t("gitops.setting.add-git-provider.basic-info.self") },
  { title: t("gitops.setting.add-git-provider.oauth-info.self") },
  { title: t("common.confirm") },
];

const state = reactive<LocalState>({
  config: {
    type: ExternalVersionControl_Type.GITLAB,
    uiType: "GITLAB_SELF_HOST",
    name: t("gitops.setting.add-git-provider.gitlab-self-host"),
    instanceUrl: "",
    applicationId: "",
    secret: "",
  },
  currentStep: 0,
});

onMounted(() => {
  window.addEventListener("bb.oauth.register-vcs", eventListener, false);
});

onUnmounted(() => {
  window.removeEventListener("bb.oauth.register-vcs", eventListener);
});

const eventListener = (event: Event) => {
  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (isEmpty(payload.error)) {
    if (
      state.config.type === ExternalVersionControl_Type.GITLAB ||
      state.config.type === ExternalVersionControl_Type.GITHUB ||
      state.config.type === ExternalVersionControl_Type.BITBUCKET ||
      state.config.type === ExternalVersionControl_Type.AZURE_DEVOPS
    ) {
      vcsV1Store
        .exchangeToken({
          vcsType: state.config.type,
          instanceUrl: state.config.instanceUrl,
          clientId: state.config.applicationId,
          clientSecret: state.config.secret,
          code: payload.code,
        })
        .then((token: OAuthToken) => {
          state.oAuthResultCallback!(token);
        })
        .catch(() => {
          state.oAuthResultCallback!(undefined);
        });
    }
  } else {
    state.oAuthResultCallback!(undefined);
  }
};

const allowNext = computed((): boolean => {
  if (state.currentStep == BASIC_INFO_STEP) {
    return isUrl(state.config.instanceUrl);
  } else if (state.currentStep == OAUTH_INFO_STEP) {
    return (
      isValidVCSApplicationIdOrSecret(
        state.config.type,
        state.config.applicationId
      ) &&
      isValidVCSApplicationIdOrSecret(state.config.type, state.config.secret)
    );
  }
  return true;
});

const attentionText = computed((): string => {
  if (state.config.type === ExternalVersionControl_Type.GITLAB) {
    if (state.config.uiType == "GITLAB_SELF_HOST") {
      return t(
        "gitops.setting.add-git-provider.gitlab-self-host-admin-requirement"
      );
    }
    return t("gitops.setting.add-git-provider.gitlab-com-admin-requirement");
  } else if (state.config.type === ExternalVersionControl_Type.GITHUB) {
    return t("gitops.setting.add-git-provider.github-com-admin-requirement");
  } else if (state.config.type === ExternalVersionControl_Type.BITBUCKET) {
    return t("gitops.setting.add-git-provider.bitbucket-admin-requirement");
  } else if (state.config.type === ExternalVersionControl_Type.AZURE_DEVOPS) {
    return t("gitops.setting.add-git-provider.azure-admin-requirement");
  }
  return "";
});

const showAttention = computed((): boolean => {
  return state.currentStep != CONFIRM_STEP;
});

const tryChangeStep = (
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  // If we are trying to move from OAuth step to Confirm step, we first verify
  // the OAuth info is correct. We achieve this by:
  // 1. Kicking of the OAuth workflow to verify the current user can login to the GitLab instance and the application id is correct.
  // 2. If step 1 succeeds, we will get a code, we use this code together with the secret to exchange for the access token. (see eventListener)
  if (state.currentStep == OAUTH_INFO_STEP && newStep > oldStep) {
    let authorizeUrl = `${state.config.instanceUrl}/oauth/authorize`;
    if (state.config.type === ExternalVersionControl_Type.GITHUB) {
      authorizeUrl = `${state.config.instanceUrl}/login/oauth/authorize`;
    } else if (state.config.type === ExternalVersionControl_Type.BITBUCKET) {
      authorizeUrl = `https://bitbucket.org/site/oauth2/authorize`;
    } else if (state.config.type === ExternalVersionControl_Type.AZURE_DEVOPS) {
      authorizeUrl = "https://app.vssps.visualstudio.com/oauth2/authorize";
    }
    const newWindow = openWindowForOAuth(
      authorizeUrl,
      state.config.applicationId,
      "bb.oauth.register-vcs",
      state.config.type
    );
    if (newWindow) {
      state.oAuthResultCallback = (token: OAuthToken | undefined) => {
        if (token) {
          state.currentStep = newStep;
          allowChangeCallback();
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("gitops.setting.add-git-provider.oauth-info-correct"),
          });
        } else {
          let description = "";
          if (state.config.type == ExternalVersionControl_Type.GITLAB) {
            // If application id mismatches, the OAuth workflow will stop early.
            // So the only possibility to reach here is we have a matching application id, while
            // we failed to exchange a token, and it's likely we are requesting with a wrong secret.
            description = t(
              "gitops.setting.add-git-provider.check-oauth-info-match"
            );
          }
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: "Failed to setup OAuth",
            description: description,
          });
        }
      };
    }
  } else {
    state.currentStep = newStep;
    allowChangeCallback();
  }
};

const tryFinishSetup = (allowChangeCallback: () => void) => {
  vcsV1Store
    .createVCS({
      name: "",
      title: state.config.name,
      type: state.config.type,
      url: state.config.instanceUrl,
      applicationId: state.config.applicationId,
      secret: state.config.secret,
      apiUrl: "",
    })
    .then((vcs: ExternalVersionControl) => {
      allowChangeCallback();
      router.push({
        name: "setting.workspace.gitops",
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
    name: "setting.workspace.gitops",
  });
};
</script>
