<template>
  <div class="space-y-4 h-full flex flex-col">
    <BBAttention
      v-if="!externalUrl"
      class="mt-4 w-full border-none"
      type="error"
      :title="$t('banner.external-url')"
      :description="$t('settings.general.workspace.external-url.description')"
    >
      <template #action>
        <NButton type="primary" @click="configureSetting">
          {{ $t("common.configure-now") }}
        </NButton>
      </template>
    </BBAttention>
    <div class="flex-1">
      <RepositorySetupWizard
        :project="project"
        @cancel="onCancel"
        @finish="onFinish"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import RepositorySetupWizard from "@/components/RepositorySetupWizard.vue";
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { SQL_EDITOR_SETTING_GENERAL_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useVCSConnectorStore,
  useProjectV1Store,
  useProjectByName,
  useActuatorV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Workflow } from "@/types/proto/v1/project_service";

const props = defineProps<{
  projectId: string;
}>();

const { t } = useI18n();
const router = useRouter();
const vcsConnectorStore = useVCSConnectorStore();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const vcsConnectorList = computed(() =>
  vcsConnectorStore.getConnectorList(project.value.name)
);

const onCancel = () => {
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
  });
};

const configureSetting = () => {
  router.push({
    name: router.currentRoute.value.name?.toString().startsWith("sql-editor")
      ? SQL_EDITOR_SETTING_GENERAL_MODULE
      : SETTING_ROUTE_WORKSPACE_GENERAL,
  });
};

const externalUrl = computed(
  () => useActuatorV1Store().serverInfo?.externalUrl ?? ""
);

const onFinish = () => {
  if (vcsConnectorList.value.length >= 1) {
    // Update workflow type in local cache.
    useProjectV1Store().updateProjectCache({
      ...project.value,
      workflow: Workflow.VCS,
    });
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("project.gitops-connector.create-success"),
  });
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
  });
};
</script>
