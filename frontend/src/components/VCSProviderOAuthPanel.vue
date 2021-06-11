<template>
  <div class="space-y-4">
    <div class="textlabel">
      <template v-if="config.type == 'GITLAB_SELF_HOST'">
        Register Bytebase as a GitLab system OAuth application.
      </template>
    </div>
    <ol class="textinfolabel space-y-2">
      <template v-if="config.type == 'GITLAB_SELF_HOST'">
        <li>
          1. Login as an Admin user to the GitLab instance. The account must be
          an Admin of the entire GitLab instance (not just a particular
          project).
        </li>
        <li>
          2. Visit the Admin page, then navigate to "Applications" section and
          click "New application" button.
          <a
            :href="createAdminApplicationURL"
            target="_blank"
            class="normal-link"
            >Visit now</a
          >
        </li>
        <li>
          3. Create our Bytebase OAuth application with the following info.
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
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Redirect URI
                </dt>
                <dd class="text-sm text-main">{{ redirectURL() }}</dd>
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
        :placeholder="'ex. 5333b60a6c9f234272dac2ee6b3422aaf224e0a66def54e0d243b77be7a8edda'"
        :value="config.applicationId"
        @input="changeApplicatonId($event.target.value)"
      />
      <p v-if="state.showApplicationIdError" class="mt-2 text-sm text-error">
        Application ID must be a 64-character alphanumeric string
      </p>
      <div class="mt-4 textlabel">
        Secret <span class="text-red-600">*</span>
      </div>
      <BBTextField
        class="mt-2 w-full"
        :placeholder="'ex. b9e0efc7a233403799b42620c60ff98c146895a27b6219912ad15f4e2251cc3a'"
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
import {
  isValidVCSApplicationIdOrSecret,
  TEXT_VALIDATION_DELAY,
  VCSConfig,
  redirectURL,
} from "../types";

interface LocalState {
  applicationIdValidationTimer?: ReturnType<typeof setTimeout>;
  showApplicationIdError: boolean;
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
    const state = reactive<LocalState>({
      showApplicationIdError:
        !isEmpty(props.config.applicationId) &&
        !isValidVCSApplicationIdOrSecret(props.config.applicationId),
      showSecretError:
        !isEmpty(props.config.secret) &&
        !isValidVCSApplicationIdOrSecret(props.config.secret),
    });

    onUnmounted(() => {
      clearInterval(state.applicationIdValidationTimer);
      clearInterval(state.secretValidationTimer);
    });

    const createAdminApplicationURL = computed((): string => {
      if (props.config.type == "GITLAB_SELF_HOST") {
        return `${props.config.instanceURL}/admin/applications/new`;
      }
      return "";
    });

    const changeApplicatonId = (value: string) => {
      props.config.applicationId = value;

      clearInterval(state.applicationIdValidationTimer);
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
      props.config.secret = value;

      clearInterval(state.secretValidationTimer);
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
      redirectURL,
      state,
      createAdminApplicationURL,
      changeApplicatonId,
      changeSecret,
    };
  },
};
</script>
