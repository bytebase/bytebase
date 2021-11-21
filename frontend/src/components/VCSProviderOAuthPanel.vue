<template>
  <div class="space-y-4">
    <div class="textlabel">
      <template v-if="config.type == 'GITLAB_SELF_HOST'">
        Register Bytebase as a GitLab instance-wide OAuth application.
      </template>
    </div>
    <ol class="textinfolabel space-y-2">
      <template v-if="config.type == 'GITLAB_SELF_HOST'">
        <li>
          1. Login as an Admin user to the GitLab instance. The account must be
          an Admin of the entire GitLab instance (it has a wrench icon on the
          top bar).
          <img class="w-auto" src="../assets/gitlab_admin_area.png" />
        </li>
        <li>
          2. Go to the Admin page by clicking the wrench icon, then navigate to
          "Applications" section and click "New application" button.
          <a
            :href="createAdminApplicationURL"
            target="_blank"
            class="normal-link"
            >Direct link</a
          >
        </li>
        <li>
          3. Create your Bytebase OAuth application with the following info.
          <div class="m-4 flex justify-center">
            <dl
              class="
                divide-y divide-block-border
                border border-block-border
                shadow
                rounded-lg
              "
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
                  {{ redirectURL() }}
                  <button
                    tabindex="-1"
                    class="
                      ml-1
                      text-sm
                      font-medium
                      text-control-light
                      hover:bg-gray-100
                    "
                    @click.prevent="copyRedirecURI"
                  >
                    <svg
                      class="w-6 h-6"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                      ></path>
                    </svg>
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
          4. Paste the Application ID and Secret from that just created
          application into fields below.
        </li>
      </template>
    </ol>
    <div>
      <div class="textlabel">
        Application ID <span class="text-red-600">*</span>
      </div>
      <BBTextField
        class="mt-2 w-full"
        :placeholder="'ex. 5333b60a6c9f234272dac2ee6b3422aaf224e0a66def54e0d243b77bexa8edda'"
        :value="config.applicationID"
        @input="changeApplicatonID($event.target.value)"
      />
      <p v-if="state.showApplicationIDError" class="mt-2 text-sm text-error">
        Application ID must be a 64-character alphanumeric string
      </p>
      <div class="mt-4 textlabel">
        Secret <span class="text-red-600">*</span>
      </div>
      <BBTextField
        class="mt-2 w-full"
        :placeholder="'ex. b9e0efc7a233403799b42620c60ff98c146895a27b6219912a215f4e2251cc3a'"
        :value="config.secret"
        @input="changeSecret($event.target.value)"
      />
      <p v-if="state.showSecretError" class="mt-2 text-sm text-error">
        Secret must be a 64-character alphanumeric string
      </p>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, onUnmounted, PropType, reactive } from "@vue/runtime-core";
import isEmpty from "lodash-es/isEmpty";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import {
  isValidVCSApplicationIDOrSecret,
  TEXT_VALIDATION_DELAY,
  VCSConfig,
  redirectURL,
} from "../types";
import { useStore } from "vuex";

interface LocalState {
  applicationIDValidationTimer?: ReturnType<typeof setTimeout>;
  showApplicationIDError: boolean;
  secretValidationTimer?: ReturnType<typeof setTimeout>;
  showSecretError: boolean;
}

export default {
  name: "VCSProviderOAuthPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const state = reactive<LocalState>({
      showApplicationIDError:
        !isEmpty(props.config.applicationID) &&
        !isValidVCSApplicationIDOrSecret(props.config.applicationID),
      showSecretError:
        !isEmpty(props.config.secret) &&
        !isValidVCSApplicationIDOrSecret(props.config.secret),
    });

    onUnmounted(() => {
      if (state.applicationIDValidationTimer) {
        clearInterval(state.applicationIDValidationTimer);
      }
      if (state.secretValidationTimer) {
        clearInterval(state.secretValidationTimer);
      }
    });

    const createAdminApplicationURL = computed((): string => {
      if (props.config.type == "GITLAB_SELF_HOST") {
        return `${props.config.instanceURL}/admin/applications/new`;
      }
      return "";
    });

    const copyRedirecURI = () => {
      toClipboard(redirectURL()).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `Redirect URI copied to clipboard. Paste to the corresponding field on the OAuth application form.`,
        });
      });
    };

    const changeApplicatonID = (value: string) => {
      props.config.applicationID = value;

      if (state.applicationIDValidationTimer) {
        clearInterval(state.applicationIDValidationTimer);
      }
      // If text becomes valid, we immediately clear the error.
      // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
      if (isValidVCSApplicationIDOrSecret(props.config.applicationID)) {
        state.showApplicationIDError = false;
      } else {
        state.applicationIDValidationTimer = setTimeout(() => {
          // If error is already displayed, we hide the error only if there is valid input.
          // Otherwise, we hide the error if input is either empty or valid.
          if (state.showApplicationIDError) {
            state.showApplicationIDError = !isValidVCSApplicationIDOrSecret(
              props.config.applicationID
            );
          } else {
            state.showApplicationIDError =
              !isValidVCSApplicationIDOrSecret(props.config.applicationID) &&
              !isValidVCSApplicationIDOrSecret(props.config.applicationID);
          }
        }, TEXT_VALIDATION_DELAY);
      }
    };

    const changeSecret = (value: string) => {
      props.config.secret = value;

      if (state.secretValidationTimer) {
        clearInterval(state.secretValidationTimer);
      }
      // If text becomes valid, we immediately clear the error.
      // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
      if (isValidVCSApplicationIDOrSecret(props.config.secret)) {
        state.showSecretError = false;
      } else {
        state.secretValidationTimer = setTimeout(() => {
          // If error is already displayed, we hide the error only if there is valid input.
          // Otherwise, we hide the error if input is either empty or valid.
          if (state.showSecretError) {
            state.showSecretError = !isValidVCSApplicationIDOrSecret(
              props.config.secret
            );
          } else {
            state.showSecretError =
              !isEmpty(props.config.secret) &&
              !isValidVCSApplicationIDOrSecret(props.config.secret);
          }
        }, TEXT_VALIDATION_DELAY);
      }
    };

    return {
      redirectURL,
      state,
      createAdminApplicationURL,
      copyRedirecURI,
      changeApplicatonID,
      changeSecret,
    };
  },
};
</script>
