<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <VCSIcon custom-class="h-6" :type="vcs.type" />
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ vcs.title }}
        </h3>
      </div>
      <NButton :disabled="!hasUpdateVCSPermission" @click.prevent="editVCS">
        {{ $t("common.edit") }}
      </NButton>
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
      </dl>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_GITOPS_DETAIL } from "@/router/dashboard/workspaceRoutes";
import { useCurrentUserV1 } from "@/store";
import { getVCSId } from "@/store/modules/v1/common";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps({
  vcs: {
    required: true,
    type: Object as PropType<VCSProvider>,
  },
});

const router = useRouter();
const currentUser = useCurrentUserV1();

const hasUpdateVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.update");
});

const editVCS = () => {
  router.push({
    name: WORKSPACE_ROUTE_GITOPS_DETAIL,
    params: {
      vcsResourceId: getVCSId(props.vcs.name),
    },
  });
};
</script>
