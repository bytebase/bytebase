<template>
  <BBAttention
    v-if="showAttention"
    :style="'WARN'"
    :description="attentionText"
  />
  <BBStepTab
    class="mt-4"
    :step-item-list="stepList"
    :allow-next="allowNext"
    :finish-title="'Confirm and add'"
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

<script lang="ts">
import { computed } from "@vue/runtime-core";
import { reactive } from "@vue/reactivity";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import isEmpty from "lodash-es/isEmpty";
import { BBStepTabItem } from "../bbkit/types";
import VCSProviderBasicInfoPanel from "./VCSProviderBasicInfoPanel.vue";
import VCSProviderOAuthPanel from "./VCSProviderOAuthPanel.vue";
import VCSProviderConfirmPanel from "./VCSProviderConfirmPanel.vue";
import {
  isValidVCSApplicationIdOrSecret,
  VCSConfig,
  VCSCreate,
  VCS,
  openWindowForOAuth,
  OAuthWindowEventPayload,
  OAuthWindowEvent,
  OAuthConfig,
  redirectURL,
  OAuthToken,
} from "../types";
import { isURL } from "../utils";

const BASIC_INFO_STEP = 0;
const OAUTH_INFO_STEP = 1;
const CONFIRM_STEP = 2;

const stepList: BBStepTabItem[] = [
  { title: "Basic info" },
  { title: "OAuth application info" },
  { title: "Confirm" },
];

interface LocalState {
  config: VCSConfig;
  currentStep: number;
  oAuthResultCallback?: (token: OAuthToken | undefined) => void;
}

export default {
  name: "VCSSetupWizard",
  components: {
    VCSProviderBasicInfoPanel,
    VCSProviderOAuthPanel,
    VCSProviderConfirmPanel,
  },
  setup() {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      config: {
        type: "GITLAB_SELF_HOST",
        name: "GitLab self-host",
        instanceURL: "",
        applicationId: "",
        secret: "",
      },
      currentStep: 0,
    });

    const eventListener = (event: Event) => {
      const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
      if (isEmpty(payload.error)) {
        if (state.config.type == "GITLAB_SELF_HOST") {
          const oAuthConfig: OAuthConfig = {
            endpoint: `${state.config.instanceURL}/oauth/token`,
            applicationId: state.config.applicationId,
            secret: state.config.secret,
            redirectURL: redirectURL(),
          };
          store
            .dispatch("gitlab/exchangeToken", {
              oAuthConfig,
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

      window.removeEventListener(OAuthWindowEvent, eventListener);
    };

    const allowNext = computed((): boolean => {
      if (state.currentStep == BASIC_INFO_STEP) {
        return isURL(state.config.instanceURL);
      } else if (state.currentStep == OAUTH_INFO_STEP) {
        return (
          isValidVCSApplicationIdOrSecret(state.config.applicationId) &&
          isValidVCSApplicationIdOrSecret(state.config.secret)
        );
      }
      return true;
    });

    const attentionText = computed((): string => {
      if (state.config.type == "GITLAB_SELF_HOST") {
        return "You need to be an Admin of your chosen GitLab instance to configure this. Otherwise, you need to ask your GitLab instance Admin to register Bytebase as a GitLab instance-wide OAuth application, then provide you that Application ID and Secret to fill at the 'OAuth application info' step.";
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
        const newWindow = openWindowForOAuth(
          `${state.config.instanceURL}/oauth/authorize`,
          state.config.applicationId
        );
        if (newWindow) {
          state.oAuthResultCallback = (token: OAuthToken | undefined) => {
            if (token) {
              state.currentStep = newStep;
              allowChangeCallback();
              store.dispatch("notification/pushNotification", {
                module: "bytebase",
                style: "SUCCESS",
                title: "Verified OAuth info is correct",
              });
            } else {
              var description = "";
              if (state.config.type == "GITLAB_SELF_HOST") {
                // If application id mismatches, the OAuth workflow will stop early.
                // So the only possibility to reach here is we have a matching application id, while
                // we failed to exchange a token, and it's likely we are requesting with a wrong secret.
                description =
                  "Please make sure Secret matches the one from your GitLab instance Application.";
              }
              store.dispatch("notification/pushNotification", {
                module: "bytebase",
                style: "CRITICAL",
                title: "Failed to setup OAuth",
                description: description,
              });
            }
          };
          window.addEventListener(OAuthWindowEvent, eventListener, false);
        }
      } else {
        state.currentStep = newStep;
        allowChangeCallback();
      }
    };

    const tryFinishSetup = (allowChangeCallback: () => void) => {
      const vcsCreate: VCSCreate = {
        ...state.config,
      };
      store.dispatch("vcs/createVCS", vcsCreate).then((vcs: VCS) => {
        allowChangeCallback();
        router.push({
          name: "setting.workspace.version-control",
        });
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: `Successfully added Git provider '${vcs.name}'`,
        });
      });
    };

    const cancelSetup = () => {
      router.push({
        name: "setting.workspace.version-control",
      });
    };

    return {
      stepList,
      state,
      allowNext,
      attentionText,
      showAttention,
      tryChangeStep,
      tryFinishSetup,
      cancelSetup,
    };
  },
};
</script>
