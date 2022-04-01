<template>
  <div class="space-y-4">
    <div class="textlabel">
      <template v-if="config.type == 'GITLAB_SELF_HOST'">
        {{
          $t(
            "version-control.setting.add-git-provider.oauth-info.register-oauth-application"
          )
        }}
      </template>
    </div>
    <ol class="textinfolabel space-y-2">
      <template v-if="config.type == 'GITLAB_SELF_HOST'">
        <li>
          1.
          {{
            $t(
              "version-control.setting.add-git-provider.oauth-info.login-as-admin"
            )
          }}
          <img class="w-auto" src="../assets/gitlab_admin_area.png" />
        </li>
        <li>
          2.
          {{
            $t(
              "version-control.setting.add-git-provider.oauth-info.visit-admin-page"
            )
          }}
          <a
            :href="createAdminApplicationUrl"
            target="_blank"
            class="normal-link"
            >{{
              $t(
                "version-control.setting.add-git-provider.oauth-info.direct-link"
              )
            }}</a
          >
        </li>
        <li>
          3.
          {{
            $t(
              "version-control.setting.add-git-provider.oauth-info.create-oauth-app"
            )
          }}
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Redirect URI
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Trusted
                </dt>
                <dd class="text-sm text-main">Yes</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Confidential
                </dt>
                <dd class="text-sm text-main">Yes</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Scopes
                </dt>
                <dd class="text-sm text-main">api</dd>
              </div>
            </dl>
          </div>
        </li>
        <li>
          4.
          {{
            $t(
              "version-control.setting.add-git-provider.oauth-info.paste-oauth-info"
            )
          }}
        </li>
      </template>
    </ol>
    <div>
      <div class="textlabel">
        {{ $t("common.application") }} ID <span class="text-red-600">*</span>
      </div>
      <BBTextField
        class="mt-2 w-full"
        :placeholder="'ex. 5333b60a6c9f234272dac2ee6b3422aaf224e0a66def54e0d243b77bexa8edda'"
        :value="config.applicationId"
        @input="(e: any) => changeApplicationId(e.target.value)"
      />
      <p v-if="state.showApplicationIdError" class="mt-2 text-sm text-error">
        {{
          $t(
            "version-control.setting.add-git-provider.oauth-info.application-id-error"
          )
        }}
      </p>
      <div class="mt-4 textlabel">
        Secret <span class="text-red-600">*</span>
      </div>
      <BBTextField
        class="mt-2 w-full"
        :placeholder="'ex. b9e0efc7a233403799b42620c60ff98c146895a27b6219912a215f4e2251cc3a'"
        :value="config.secret"
        @input="(e: any) => changeSecret(e.target.value)"
      />
      <p v-if="state.showSecretError" class="mt-2 text-sm text-error">
        {{
          $t("version-control.setting.add-git-provider.oauth-info.secret-error")
        }}
      </p>
    </div>
  </div>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  onUnmounted,
  PropType,
  reactive,
} from "vue";
import isEmpty from "lodash-es/isEmpty";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import {
  isValidVCSApplicationIdOrSecret,
  TEXT_VALIDATION_DELAY,
  VCSConfig,
  redirectUrl,
} from "../types";
import { useStore } from "vuex";
import { useNotificationStore } from "@/store";

interface LocalState {
  applicationIdValidationTimer?: ReturnType<typeof setTimeout>;
  showApplicationIdError: boolean;
  secretValidationTimer?: ReturnType<typeof setTimeout>;
  showSecretError: boolean;
}

export default defineComponent({
  name: "VCSProviderOAuthPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup(props) {
    const store = useStore();
    const notificationStore = useNotificationStore();
    const state = reactive<LocalState>({
      showApplicationIdError:
        !isEmpty(props.config.applicationId) &&
        !isValidVCSApplicationIdOrSecret(props.config.applicationId),
      showSecretError:
        !isEmpty(props.config.secret) &&
        !isValidVCSApplicationIdOrSecret(props.config.secret),
    });

    onUnmounted(() => {
      if (state.applicationIdValidationTimer) {
        clearInterval(state.applicationIdValidationTimer);
      }
      if (state.secretValidationTimer) {
        clearInterval(state.secretValidationTimer);
      }
    });

    const createAdminApplicationUrl = computed((): string => {
      if (props.config.type == "GITLAB_SELF_HOST") {
        return `${props.config.instanceUrl}/admin/applications/new`;
      }
      return "";
    });

    const copyRedirectURI = () => {
      toClipboard(redirectUrl()).then(() => {
        notificationStore.pushNotification({
          module: "bytebase",
          style: "INFO",
          title: `Redirect URI copied to clipboard. Paste to the corresponding field on the OAuth application form.`,
        });
      });
    };

    const changeApplicationId = (value: string) => {
      // eslint-disable-next-line vue/no-mutating-props
      props.config.applicationId = value;

      if (state.applicationIdValidationTimer) {
        clearInterval(state.applicationIdValidationTimer);
      }
      // If text becomes valid, we immediately clear the error.
      // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
      if (isValidVCSApplicationIdOrSecret(props.config.applicationId)) {
        state.showApplicationIdError = false;
      } else {
        state.applicationIdValidationTimer = setTimeout(() => {
          // If error is already displayed, we hide the error only if there is valid input.
          // Otherwise, we hide the error if input is either empty or valid.
          if (state.showApplicationIdError) {
            state.showApplicationIdError = !isValidVCSApplicationIdOrSecret(
              props.config.applicationId
            );
          } else {
            state.showApplicationIdError =
              !isValidVCSApplicationIdOrSecret(props.config.applicationId) &&
              !isValidVCSApplicationIdOrSecret(props.config.applicationId);
          }
        }, TEXT_VALIDATION_DELAY);
      }
    };

    const changeSecret = (value: string) => {
      // eslint-disable-next-line vue/no-mutating-props
      props.config.secret = value;

      if (state.secretValidationTimer) {
        clearInterval(state.secretValidationTimer);
      }
      // If text becomes valid, we immediately clear the error.
      // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
      if (isValidVCSApplicationIdOrSecret(props.config.secret)) {
        state.showSecretError = false;
      } else {
        state.secretValidationTimer = setTimeout(() => {
          // If error is already displayed, we hide the error only if there is valid input.
          // Otherwise, we hide the error if input is either empty or valid.
          if (state.showSecretError) {
            state.showSecretError = !isValidVCSApplicationIdOrSecret(
              props.config.secret
            );
          } else {
            state.showSecretError =
              !isEmpty(props.config.secret) &&
              !isValidVCSApplicationIdOrSecret(props.config.secret);
          }
        }, TEXT_VALIDATION_DELAY);
      }
    };

    return {
      redirectUrl,
      state,
      createAdminApplicationUrl,
      copyRedirectURI,
      changeApplicationId,
      changeSecret,
    };
  },
});
</script>
