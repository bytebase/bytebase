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
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import {
  useVCSConnectorStore,
  useProjectV1Store,
  useVCSV1Store,
  useCurrentUserV1,
  pushNotification,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Workflow } from "@/types/proto/v1/project_service";
import { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
  vcsConnectorId: string;
}>();

const vcsConnectorStore = useVCSConnectorStore();
const vcsProviderStore = useVCSV1Store();
const projectV1Store = useProjectV1Store();
const currentUser = useCurrentUserV1();
const router = useRouter();
const { t } = useI18n();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const vcsConnector = computed(
  () =>
    vcsConnectorStore.getConnector(project.value.name, props.vcsConnectorId) ??
    VCSConnector.fromPartial({})
);

const vcsConnectorList = computed(() =>
  vcsConnectorStore.getConnectorList(project.value.name)
);

const allowEdit = computed(() => {
  return hasProjectPermissionV2(
    project.value,
    currentUser.value,
    "bb.vcsConnectors.update"
  );
});

const allowDelete = computed(() => {
  return hasProjectPermissionV2(
    project.value,
    currentUser.value,
    "bb.vcsConnectors.delete"
  );
});

watchEffect(async () => {
  const connector = await vcsConnectorStore.getOrFetchConnector(
    project.value.name,
    props.vcsConnectorId
  );
  if (connector) {
    await vcsProviderStore.getOrFetchVCSByName(connector.vcsProvider);
  }
});

const onCancel = () => {
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
  });
};

const onDelete = () => {
  if (vcsConnectorList.value.length === 0) {
    // Update workflow type in local cache.
    projectV1Store.updateProjectCache({
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
