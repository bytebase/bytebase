<template>
  <div class="text-lg leading-6 font-medium text-control">Name</div>
  <BBTextField
    class="mt-2 w-full"
    :placeholder="'GitLab self-host'"
    :value="config.name"
    @input="config.name = $event.target.value"
  />
  <div class="mt-4 text-lg leading-6 font-medium text-control">
    GitLab Instance URL <span class="text-red-600">*</span>
  </div>
  <BBTextField
    class="mt-2 w-full"
    :placeholder="'https://gitlab.example.com'"
    :value="config.instanceURL"
    @input="changeUrl($event.target.value)"
  />
  <p v-if="state.showURLError" class="mt-2 text-sm text-error">
    Instance URL must begin with https:// or http://
  </p>
</template>

<script lang="ts">
import { onUnmounted, PropType, reactive } from "@vue/runtime-core";
import isEmpty from "lodash-es/isEmpty";
import { TEXT_VALIDATION_DELAY, VCSConfig } from "../types";
import { isUrl } from "../utils";

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
        !isEmpty(props.config.instanceURL) && !isUrl(props.config.instanceURL),
    });

    onUnmounted(() => {
      clearInterval(state.urlValidationTimer);
    });

    const changeUrl = (value: string) => {
      props.config.instanceURL = value;

      clearInterval(state.urlValidationTimer);
      // If text becomes valid, we immediately clear the error.
      // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
      if (isUrl(props.config.instanceURL)) {
        state.showURLError = false;
      } else {
        state.urlValidationTimer = setTimeout(() => {
          // If error is already displayed, we hide the error only if there is valid input.
          // Otherwise, we hide the error if input is either empty or valid.
          if (state.showURLError) {
            state.showURLError = !isUrl(props.config.instanceURL);
          } else {
            state.showURLError =
              !isEmpty(props.config.instanceURL) &&
              !isUrl(props.config.instanceURL);
          }
        }, TEXT_VALIDATION_DELAY);
      }
    };

    return { state, changeUrl };
  },
};
</script>
