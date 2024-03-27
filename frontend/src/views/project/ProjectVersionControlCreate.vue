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
  if (vcsConnectorList.value.length === 1) {
    // refresh project
    // TODO(ed): we can only update the frontend data, don't need to call API.
    projectV1Store.fetchProjectByName(project.value.name);
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
