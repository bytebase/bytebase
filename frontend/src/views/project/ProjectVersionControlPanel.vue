<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t("workflow.gitops-workflow-description") }}
      <LearnMoreLink
        url="https://www.bytebase.com/docs/vcs-integration/add-gitops-connector/?source=console"
      />
    </div>
    <div class="flex flex-row items-center justify-end gap-x-2">
      <NButton v-if="allowCreate" type="primary" @click="createConnector">
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
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
import { PlusIcon } from "lucide-vue-next";
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, h, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import VCSIcon from "@/components/VCS/VCSIcon.vue";
import {
  PROJECT_V1_ROUTE_GITOPS_DETAIL,
  PROJECT_V1_ROUTE_GITOPS_CREATE,
} from "@/router/dashboard/projectV1";
import {
  useVCSConnectorStore,
  useVCSProviderStore,
  useProjectByName,
} from "@/store";
import { getProjectNameAndDatabaseGroupName, projectNamePrefix } from "@/store/modules/v1/common";
import { getVCSConnectorId } from "@/store/modules/v1/common";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const { t } = useI18n();
const router = useRouter();
const vcsV1Store = useVCSProviderStore();
const vcsConnectorStore = useVCSConnectorStore();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const allowCreate = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.vcsConnectors.create");
});

const allowView = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.vcsConnectors.get");
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
            customClass: "h-5",
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
      key: "databaseGroup",
      title: t("database-group.self"),
      resizable: true,
      render: (connector) => getProjectNameAndDatabaseGroupName(connector.databaseGroup)[1] || '-',
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
    {
      key: "baseDirectory",
      title: t("repository.base-directory"),
      render: (connector) => connector.baseDirectory,
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
