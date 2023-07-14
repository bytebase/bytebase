<template>
  <div class="m-4 flex justify-center">
    <dl
      class="divide-y divide-block-border border border-block-border shadow rounded-lg"
    >
      <div class="px-4 py-5 sm:px-6">
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ $t("gitops.setting.add-git-provider.confirm.confirm-info") }}
        </h3>
        <p class="mt-1 textinfolabel">
          {{
            $t("gitops.setting.add-git-provider.confirm.confirm-description")
          }}
        </p>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          {{ $t("common.type") }}
        </dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          <div
            v-if="config.uiType == 'GITLAB_SELF_HOST'"
            class="flex flex-row items-center space-x-2"
          >
            <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
            <div class="whitespace-nowrap">
              {{ $t("gitops.setting.add-git-provider.gitlab-self-host-ce-ee") }}
            </div>
          </div>
          <div
            v-else-if="config.uiType == 'GITLAB_COM'"
            class="flex flex-row items-center space-x-2"
          >
            <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
            <div class="whitespace-nowrap">GitLab.com</div>
          </div>
          <div
            v-else-if="config.uiType == 'GITHUB_COM'"
            class="flex flex-row items-center space-x-2"
          >
            <img class="h-6 w-auto" src="../assets/github-logo.svg" />
            <div class="whitespace-nowrap">GitHub.com</div>
          </div>
          <div
            v-else-if="config.uiType == 'GITHUB_ENTERPRISE'"
            class="flex flex-row items-center space-x-2"
          >
            <img class="h-6 w-auto" src="../assets/github-logo.svg" />
            <div class="whitespace-nowrap">
              {{ $t("gitops.setting.add-git-provider.github-self-host-ee") }}
            </div>
          </div>
          <div
            v-else-if="config.uiType == 'BITBUCKET_ORG'"
            class="flex flex-row items-center space-x-2"
          >
            <img class="h-6 w-auto" src="../assets/bitbucket-logo.svg" />
            <div class="whitespace-nowrap">Bitbucket.org</div>
          </div>
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          {{ $t("common.name") }}
        </dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          {{ config.name }}
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          {{ $t("common.instance") }} URL
        </dt>
        <dd class="col-start-2 col-span-3 text-sm text-main">
          {{ config.instanceUrl }}
        </dd>
      </div>
      <div class="grid grid-cols-4 gap-4 px-4 py-2">
        <dt class="text-sm font-medium text-control-light text-right">
          {{ $t("common.application") }} ID
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
  <div>{{ config }}</div>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { VCSConfig, redirectUrl } from "../types";

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
      redirectUrl,
      state,
    };
  },
};
</script>
