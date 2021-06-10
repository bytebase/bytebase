<template>
  <div class="flex flex-row radio-set">
    <div class="radio space-x-2">
      <input
        name="Self-host GitLab"
        tabindex="-1"
        type="radio"
        class="btn"
        value="GITLAB_SELF_HOST"
        v-model="config.type"
      />
      <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      <label class="whitespace-nowrap"
        >Self-host GitLab Enterprise Edition (EE) or Community Edition (CE)
      </label>
    </div>
  </div>
  <div class="mt-4 relative">
    <div class="absolute inset-0 flex items-center" aria-hidden="true">
      <div class="w-full border-t border-block-border"></div>
    </div>
    <div class="relative flex justify-start">
      <span class="pr-2 bg-white text-sm text-control-light">
        Coming later
      </span>
    </div>
  </div>
  <div class="mt-2 flex flex-row itmes-center space-x-4">
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
</template>

<script lang="ts">
import { PropType } from "@vue/runtime-core";
import { VCSConfig } from "../types";
export default {
  name: "VCSProviderSelectionPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup(props, ctx) {
    const selectGitLabOnPrem = () => {
      props.config.type = "GITLAB_SELF_HOST";
    };
    return { selectGitLabOnPrem };
  },
};
</script>
