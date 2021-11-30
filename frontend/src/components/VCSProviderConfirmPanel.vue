<template>
  <div class="m-4 flex justify-center">
    <dl
      class="
        divide-y divide-block-border
        border border-block-border
        shadow
        rounded-lg
      "
    >
      <div class="px-4 py-5 sm:px-6">
        <h3 class="text-lg leading-6 font-medium text-main">
          Confirm the info
        </h3>
        <p class="mt-1 textinfolabel">
          After creation, this Git provider can be chosen under the project
          dashboard "Version Control" tab by the project owner.
        </p>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">Type</dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          <div
            v-if="config.type == 'GITLAB_SELF_HOST'"
            class="flex flex-row items-center space-x-2"
          >
            <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
            <div class="whitespace-nowrap">
              Self-host GitLab Enterprise Edition (EE) or Community Edition (CE)
            </div>
          </div>
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">Name</dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          {{ config.name }}
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          Instance URL
        </dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          {{ config.instanceURL }}
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          Application ID
        </dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          {{ config.applicationId }}
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          Secret
        </dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          {{ config.secret }}
        </dd>
      </div>
    </dl>
  </div>
</template>

<script lang="ts">
import { PropType, reactive } from "@vue/runtime-core";
import { VCSConfig, redirectURL } from "../types";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
  name: "VCSProviderConfirmPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup() {
    const state = reactive<LocalState>({});

    return {
      redirectURL,
      state,
    };
  },
};
</script>
