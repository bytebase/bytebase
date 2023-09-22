<template>
  <div class="m-4 flex justify-center">
    <dl
      class="divide-y divide-block-border border border-block-border shadow rounded-lg"
    >
      <div class="px-4 py-4 sm:px-6">
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
            v-if="vcsWithUIType"
            class="flex flex-row items-center space-x-2"
          >
            <VCSIcon custom-class="h-6" :type="vcsWithUIType.type" />
            <div class="whitespace-nowrap">
              {{ vcsWithUIType.title }}
            </div>
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
        <dd class="col-start-2 col-span-3 text-sm text-main break-all">
          {{ config.secret }}
        </dd>
      </div>
    </dl>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { VCSConfig } from "@/types";
import { vcsListByUIType } from "./utils";

const props = defineProps<{
  config: VCSConfig;
}>();

const vcsWithUIType = computed(() => {
  return vcsListByUIType.value.find(
    (data) => data.uiType === props.config.uiType
  );
});
</script>
