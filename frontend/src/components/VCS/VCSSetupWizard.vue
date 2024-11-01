<template>
  <FormLayout>
    <template #body>
      <VCSProviderBasicInfoPanel :create="true" :config="state.config" />
    </template>
    <template #footer>
      <div class="flex justify-end items-center">
        <div class="space-x-3">
          <NButton v-if="showCancel" @click.prevent="cancelSetup">
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
  </FormLayout>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import FormLayout from "@/components/v2/Form/FormLayout.vue";
import { WORKSPACE_ROUTE_GITOPS } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useVCSProviderStore } from "@/store";
import type { VCSConfig } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import VCSProviderBasicInfoPanel from "./VCSProviderBasicInfoPanel.vue";

defineProps<{
  showCancel: boolean;
}>();

interface LocalState {
  config: VCSConfig;
}

const { t } = useI18n();
const router = useRouter();
const vcsV1Store = useVCSProviderStore();

const state = reactive<LocalState>({
  config: {
    type: VCSType.GITLAB,
    uiType: "GITLAB_SELF_HOST",
    resourceId: "",
    name: t("gitops.setting.add-git-provider.gitlab-self-host"),
    instanceUrl: "",
    accessToken: "",
  },
});

const allowCreate = computed(() => {
  return (
    hasWorkspacePermissionV2("bb.vcsProviders.create") &&
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
  router.back();
};
</script>
