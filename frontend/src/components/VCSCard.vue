<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <template v-if="vcs.type === ExternalVersionControl_Type.GITLAB">
          <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
        <template v-if="vcs.type === ExternalVersionControl_Type.GITHUB">
          <img class="h-6 w-auto" src="../assets/github-logo.svg" />
        </template>
        <template v-if="vcs.type === ExternalVersionControl_Type.BITBUCKET">
          <img class="h-6 w-auto" src="../assets/bitbucket-logo.svg" />
        </template>
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ vcs.title }}
        </h3>
      </div>
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="editVCS"
      >
        {{ $t("common.edit") }}
      </button>
    </div>
    <div class="border-t border-block-border">
      <dl class="divide-y divide-block-border">
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.instance") }} URL
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ vcs.url }}
          </dd>
        </div>
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.application") }} ID
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ vcs.applicationId }}
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { useRouter } from "vue-router";
import {
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";
import { vcsSlugV1 } from "../utils";

const props = defineProps({
  vcs: {
    required: true,
    type: Object as PropType<ExternalVersionControl>,
  },
});

const router = useRouter();

const editVCS = () => {
  router.push({
    name: "setting.workspace.gitops.detail",
    params: {
      vcsSlug: vcsSlugV1(props.vcs),
    },
  });
};
</script>
