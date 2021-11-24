<template>
  <div class="textlabel">
    Choose Git provider <span class="text-red-600">*</span>
  </div>
  <div class="pt-4 radio-set-row">
    <div class="radio space-x-2">
      <input
        v-model="config.type"
        name="Self-host GitLab"
        tabindex="-1"
        type="radio"
        class="btn"
        value="GITLAB_SELF_HOST"
      />
      <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      <label class="whitespace-nowrap"
        >Self-host GitLab Enterprise Edition (EE) or Community Edition (CE)
      </label>
    </div>
  </div>
  <div class="mt-4 relative">
    <div class="relative flex justify-start">
      <span class="pr-2 bg-white text-xs text-control-light">
        Coming later
      </span>
    </div>
  </div>
  <div class="mt-2 flex flex-row itmes-center space-x-4 text-xs">
    <div class="flex flex-row space-x-2 items-center text-control">
      <div class="h-5 w-5">
        <img src="../assets/gitlab-logo.svg" />
      </div>
      <label class="whitespace-nowrap">GitLab.com </label>
    </div>
    <div class="flex flex-row space-x-2 items-center text-control">
      <div class="h-5 w-5">
        <img src="../assets/github-logo.svg" />
      </div>
      <label class="whitespace-nowrap">GitHub Enterprise </label>
    </div>
    <div class="flex flex-row space-x-2 items-center text-control">
      <div class="h-5 w-5">
        <img src="../assets/github-logo.svg" />
      </div>
      <label class="whitespace-nowrap">GitHub.com </label>
    </div>
  </div>
  <div class="mt-6 pt-6 border-t border-block-border textlabel">
    {{ instanceURLLabel }} <span class="text-red-600">*</span>
  </div>
  <p class="mt-1 textinfolabel">
    The VCS instance URL. Make sure this instance and Bytebase are network
    reachable from each other.
  </p>
  <BBTextField
    class="mt-2 w-full"
    :placeholder="'https://gitlab.example.com'"
    :value="config.instanceURL"
    @input="changeURL($event.target.value)"
  />
  <p v-if="state.showURLError" class="mt-2 text-sm text-error">
    Instance URL must begin with https:// or http://
  </p>
  <div class="mt-4 textlabel">Display name</div>
  <p class="mt-1 textinfolabel">
    An optional display name to help identifying among different configs using
    the same Git provider.
  </p>
  <BBTextField
    class="mt-2 w-full"
    :placeholder="namePlaceholder"
    :value="config.name"
    @input="config.name = $event.target.value"
  />
</template>

<script lang="ts">
import { computed, onUnmounted, PropType, reactive } from "@vue/runtime-core";
import isEmpty from "lodash-es/isEmpty";
import { TEXT_VALIDATION_DELAY, VCSConfig } from "../types";
import { isURL } from "../utils";

interface LocalState {
  urlValidationTimer?: ReturnType<typeof setTimeout>;
  showURLError: boolean;
}

export default {
  name: "VCSProviderBasicInfoPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup(props) {
    const state = reactive<LocalState>({
      showURLError:
        !isEmpty(props.config.instanceURL) && !isURL(props.config.instanceURL),
    });

    onUnmounted(() => {
      if (state.urlValidationTimer) {
        clearInterval(state.urlValidationTimer);
      }
    });

    const namePlaceholder = computed((): string => {
      if (props.config.type == "GITLAB_SELF_HOST") {
        return "GitLab self-host";
      }
      return "";
    });

    const instanceURLLabel = computed((): string => {
      if (props.config.type == "GITLAB_SELF_HOST") {
        return "GitLab Instance URL";
      }
      return "";
    });

    const changeURL = (value: string) => {
      props.config.instanceURL = value;

      if (state.urlValidationTimer) {
        clearInterval(state.urlValidationTimer);
      }
      // If text becomes valid, we immediately clear the error.
      // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
      if (isURL(props.config.instanceURL)) {
        state.showURLError = false;
      } else {
        state.urlValidationTimer = setTimeout(() => {
          // If error is already displayed, we hide the error only if there is valid input.
          // Otherwise, we hide the error if input is either empty or valid.
          if (state.showURLError) {
            state.showURLError = !isURL(props.config.instanceURL);
          } else {
            state.showURLError =
              !isEmpty(props.config.instanceURL) &&
              !isURL(props.config.instanceURL);
          }
        }, TEXT_VALIDATION_DELAY);
      }
    };

    return { state, namePlaceholder, instanceURLLabel, changeURL };
  },
};
</script>
