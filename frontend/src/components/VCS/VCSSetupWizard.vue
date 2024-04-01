<template>
  <VCSProviderBasicInfoPanel :config="state.config" />
  <div class="pt-4 mt-6 flex border-t justify-end">
    <div class="space-x-3">
      <NButton @click.prevent="cancelSetup">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        type="primary"
        :disabled="!allowCreate"
        @click.prevent="tryFinishSetup"
      >
        {{ $t("common.confirm-and-add") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { WORKSPACE_ROUTE_GITOPS } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useVCSProviderStore,
  useCurrentUserV1,
} from "@/store";
import type { VCSConfig } from "@/types";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import VCSProviderBasicInfoPanel from "./VCSProviderBasicInfoPanel.vue";

withDefaults(
  defineProps<{
    showCancel?: boolean;
  }>(),
  {
    showCancel: true,
  }
);

interface LocalState {
  config: VCSConfig;
}

const { t } = useI18n();
const router = useRouter();
const vcsV1Store = useVCSProviderStore();
const currentUser = useCurrentUserV1();

const state = reactive<LocalState>({
  config: {
    type: VCSProvider_Type.GITLAB,
    uiType: "GITLAB_SELF_HOST",
    resourceId: "",
    name: t("gitops.setting.add-git-provider.gitlab-self-host"),
    instanceUrl: "",
    accessToken: "",
  },
});

const allowCreate = computed(() => {
  return (
    hasWorkspacePermissionV2(currentUser.value, "bb.vcsProviders.create") &&
    state.config.instanceUrl &&
    state.config.accessToken &&
    state.config.name
  );
});

const tryFinishSetup = () => {
  vcsV1Store
    .createVCS(state.config.resourceId, {
      name: "",
      title: state.config.name,
      type: state.config.type,
      url: state.config.instanceUrl,
      accessToken: state.config.accessToken,
    })
    .then((vcs: VCSProvider) => {
      router.push({
        name: WORKSPACE_ROUTE_GITOPS,
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("gitops.setting.add-git-provider.add-success", {
          vcs: vcs.title,
        }),
      });
    });
};

const cancelSetup = () => {
  router.push({
    name: WORKSPACE_ROUTE_GITOPS,
  });
};
</script>
