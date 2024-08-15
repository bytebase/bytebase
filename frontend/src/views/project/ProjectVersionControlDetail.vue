<template>
  <RepositoryPanel
    :project="project"
    :vcs-connector="vcsConnector"
    :allow-edit="allowEdit"
    :allow-delete="allowDelete"
    @delete="onDelete"
    @update="onUpdate"
    @cancel="onCancel"
  />
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import RepositoryPanel from "@/components/RepositoryPanel.vue";
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import {
  useVCSConnectorStore,
  useProjectV1Store,
  useVCSProviderStore,
  pushNotification,
  useProjectByName,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Workflow } from "@/types/proto/v1/project_service";
import { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
  vcsConnectorId: string;
}>();

const { t } = useI18n();
const router = useRouter();
const vcsConnectorStore = useVCSConnectorStore();
const vcsProviderStore = useVCSProviderStore();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const vcsConnector = computed(
  () =>
    vcsConnectorStore.getConnector(project.value.name, props.vcsConnectorId) ??
    VCSConnector.fromPartial({})
);

const vcsConnectorList = computed(() =>
  vcsConnectorStore.getConnectorList(project.value.name)
);

const allowEdit = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.vcsConnectors.update");
});

const allowDelete = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.vcsConnectors.delete");
});

watchEffect(async () => {
  await Promise.all([
    vcsProviderStore.getOrFetchVCSList(),
    vcsConnectorStore.getOrFetchConnectors(project.value.name),
  ]);
});

const onCancel = () => {
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
  });
};

const onDelete = () => {
  if (vcsConnectorList.value.length === 0) {
    // Update workflow type in local cache.
    useProjectV1Store().updateProjectCache({
      ...project.value,
      workflow: Workflow.UI,
    });
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("project.gitops-connector.delete-success"),
  });

  onCancel();
};

const onUpdate = () => {
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("project.gitops-connector.update-success"),
  });
};
</script>
