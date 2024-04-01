<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t("workflow.gitops-workflow-description") }}
    </div>
    <div class="flex flex-row items-center justify-end gap-x-2">
      <NButton v-if="allowCreate" type="primary" @click="createConnector">
        {{ $t("project.gitops-connector.add") }}
      </NButton>
    </div>
    <NDataTable
      :data="connectorList"
      :columns="columnList"
      :striped="true"
      :bordered="true"
    />
  </div>
</template>

<script setup lang="ts">
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, h, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import VCSIcon from "@/components/VCS/VCSIcon.vue";
import {
  PROJECT_V1_ROUTE_GITOPS_DETAIL,
  PROJECT_V1_ROUTE_GITOPS_CREATE,
} from "@/router/dashboard/projectV1";
import {
  useVCSConnectorStore,
  useProjectV1Store,
  useCurrentUserV1,
  useVCSProviderStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getVCSConnectorId } from "@/store/modules/v1/common";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const projectV1Store = useProjectV1Store();
const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSProviderStore();
const vcsConnectorStore = useVCSConnectorStore();
const router = useRouter();
const { t } = useI18n();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const allowCreate = computed(() => {
  return hasProjectPermissionV2(
    project.value,
    currentUser.value,
    "bb.vcsConnectors.create"
  );
});

const allowView = computed(() => {
  return hasProjectPermissionV2(
    project.value,
    currentUser.value,
    "bb.vcsConnectors.get"
  );
});

const createConnector = () => {
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS_CREATE,
  });
};

const columnList = computed((): DataTableColumn<VCSConnector>[] => {
  const list: DataTableColumn<VCSConnector>[] = [
    {
      key: "provider",
      title: t("repository.git-provider"),
      resizable: true,
      render: (connector) => {
        const vcs = vcsV1Store.getVCSByName(connector.vcsProvider);
        return h("div", { class: "flex items-center gap-x-2" }, [
          h(VCSIcon, {
            type: vcs?.type ?? VCSType.VCS_TYPE_UNSPECIFIED,
            customClass: "h-4",
          }),
          vcs?.title ?? "",
        ]);
      },
    },
    {
      key: "id",
      title: t("common.id"),
      resizable: true,
      render: (connector) => getVCSConnectorId(connector.name).vcsConnectorId,
    },
    {
      key: "repository",
      title: t("common.repository"),
      resizable: true,
      render: (connector) => connector.fullPath,
    },
    {
      key: "branch",
      title: t("common.branch"),
      render: (connector) => connector.branch,
    },
  ];

  if (allowView.value) {
    list.push({
      key: "view",
      title: "",
      render: (connector) =>
        h(
          "div",
          { class: "flex justify-end" },
          h(
            NButton,
            {
              size: "small",
              onClick: () => {
                router.push({
                  name: PROJECT_V1_ROUTE_GITOPS_DETAIL,
                  params: {
                    vcsConnectorId: getVCSConnectorId(connector.name)
                      .vcsConnectorId,
                  },
                });
              },
            },
            t("common.view")
          )
        ),
    });
  }

  return list;
});

const connectorList = computed(() => {
  const list = vcsConnectorStore.getConnectorList(project.value.name);
  return list;
});

watchEffect(async () => {
  await Promise.all([
    vcsV1Store.getOrFetchVCSList(),
    vcsConnectorStore.getOrFetchConnectors(project.value.name),
  ]);
});
</script>
