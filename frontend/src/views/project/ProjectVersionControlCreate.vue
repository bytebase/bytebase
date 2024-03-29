<template>
  <RepositorySetupWizard
    :project="project"
    @cancel="onCancel"
    @finish="onFinish"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useVCSConnectorStore,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { Workflow } from "@/types/proto/v1/project_service";

const props = defineProps<{
  projectId: string;
}>();

const router = useRouter();
const projectV1Store = useProjectV1Store();
const vcsConnectorStore = useVCSConnectorStore();
const { t } = useI18n();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const vcsConnectorList = computed(() =>
  vcsConnectorStore.getConnectorList(project.value.name)
);

const onCancel = () => {
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
  });
};

const onFinish = () => {
  if (vcsConnectorList.value.length >= 1) {
    // Update workflow type in local cache.
    projectV1Store.updateProjectCache({
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
