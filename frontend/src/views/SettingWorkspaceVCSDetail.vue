<template>
  <div class="flex flex-col gap-y-4">
    <VCSProviderBasicInfoPanel :create="false" :config="state.config" />

    <div class="pt-4 mt-2 flex border-t justify-between">
      <template v-if="connectorList.length == 0">
        <BBButtonConfirm
          :style="'DELETE'"
          :button-text="$t('gitops.setting.git-provider.delete')"
          :ok-text="$t('common.delete')"
          :confirm-title="
            $t('gitops.setting.git-provider.delete-confirm', {
              name: vcs?.title,
            })
          "
          :disabled="!hasDeleteVCSPermission"
          :require-confirm="true"
          @confirm="deleteVCS"
        />
      </template>
      <template v-else>
        <div class="mt-1 textinfolabel">
          {{ $t("gitops.setting.git-provider.delete-forbidden") }}
        </div>
      </template>
      <div class="space-x-3">
        <NButton v-if="allowUpdate" @click.prevent="cancel">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowUpdate"
          @click.prevent="doUpdate"
        >
          {{ $t("common.update") }}
        </NButton>
      </div>
    </div>
  </div>

  <div class="py-6">
    <div class="text-lg leading-6 font-medium text-main">
      {{ $t("repository.linked") + ` (${connectorList.length})` }}
    </div>
    <div class="mt-4">
      <VCSConnectorTable :connector-list="connectorList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import { reactive, computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import VCSConnectorTable from "@/components/VCSConnectorTable.vue";
import { WORKSPACE_ROUTE_GITOPS } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useCurrentUserV1,
  useVCSConnectorStore,
  useVCSProviderStore,
} from "@/store";
import type { VCSConfig } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  config: VCSConfig;
}

const props = defineProps<{
  vcsResourceId: string;
}>();

const router = useRouter();
const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSProviderStore();
const vcsConnectorStore = useVCSConnectorStore();

const vcs = computed((): VCSProvider | undefined => {
  return vcsV1Store.getVCSById(props.vcsResourceId);
});

const initState = computed(
  (): VCSConfig => ({
    type: vcs.value?.type ?? VCSType.GITLAB,
    uiType: "GITLAB_SELF_HOST",
    resourceId: props.vcsResourceId,
    name: vcs.value?.title ?? "",
    instanceUrl: vcs.value?.url ?? "",
    accessToken: "",
  })
);

const resetState = () => {
  state.config = initState.value;
};

const state = reactive<LocalState>({
  config: initState.value,
});

const hasUpdateVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.update");
});

const hasDeleteVCSPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.delete");
});

watchEffect(async () => {
  await vcsV1Store.getOrFetchVCSList();
  resetState();
  if (vcs.value) {
    if (
      !hasWorkspacePermissionV2(
        currentUser.value,
        "bb.vcsProviders.listProjects"
      )
    ) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `You don't have permission to list projects in '${vcs.value.title}'`,
      });
      return;
    }
    await vcsConnectorStore.fetchConnectorsInProvider(vcs.value.name);
  }
});

const connectorList = computed(() => {
  return vcsConnectorStore.getConnectorsInProvider(vcs.value?.name ?? "");
});

const allowUpdate = computed(() => {
  return (
    (state.config.name != vcs.value?.title ||
      !isEmpty(state.config.accessToken)) &&
    hasUpdateVCSPermission.value
  );
});

const doUpdate = () => {
  if (!vcs.value) {
    return;
  }
  const vcsPatch: Partial<VCSProvider> = {
    name: vcs.value.name,
  };
  if (state.config.name != vcs.value.title) {
    vcsPatch.title = state.config.name;
  }
  if (!isEmpty(state.config.accessToken)) {
    vcsPatch.accessToken = state.config.accessToken;
  }

  vcsV1Store.updateVCS(vcsPatch).then((updatedVCS: VCSProvider | undefined) => {
    if (!updatedVCS) {
      return;
    }
    resetState();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully updated '${updatedVCS.title}'`,
    });
  });
};

const cancel = () => {
  resetState();
};

const deleteVCS = () => {
  if (!vcs.value) {
    return;
  }
  const title = vcs.value.title;
  vcsV1Store.deleteVCS(vcs.value.name).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully deleted '${title}'`,
    });
    router.push({
      name: WORKSPACE_ROUTE_GITOPS,
    });
  });
};
</script>
