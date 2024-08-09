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
import RepositorySetupWizard from "@/components/RepositorySetupWizard.vue";
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useVCSConnectorStore,
  useProjectV1Store,
  useProjectByName,
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
