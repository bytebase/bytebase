<template>
  <div class="textlabel">
    {{ instanceURLLabel }} <span class="text-red-600">*</span>
  </div>
  <p class="mt-1 textinfolabel">
    The VCS instance URL. Make sure this instance and Bytebase are network
    accessible from each other.
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
  <div class="mt-4 textlabel">Display Name</div>
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
  name: "VCSProviderInfoPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      showURLError:
        !isEmpty(props.config.instanceURL) && !isURL(props.config.instanceURL),
    });

    onUnmounted(() => {
      clearInterval(state.urlValidationTimer);
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

      clearInterval(state.urlValidationTimer);
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
