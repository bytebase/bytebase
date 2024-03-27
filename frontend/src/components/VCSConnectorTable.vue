<template>
  <BBTable
    :column-list="columnList"
    :data-source="connectorList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    @click-row="onClick"
  >
    <template #body="{ rowData: connector }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ projectV1Name(getProject(connector)) }}
      </BBTableCell>
      <BBTableCell class="w-32">
        {{ connector.fullPath }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_GITOPS_DETAIL } from "@/router/dashboard/projectV1";
import { useProjectV1Store, useCurrentUserV1 } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getVCSConnectorId } from "@/store/modules/v1/common";
import type { VCSConnector } from "@/types/proto/v1/vcs_connector_service";
import { hasProjectPermissionV2 } from "@/utils";
import { projectV1Name } from "../utils";

const props = defineProps<{
  connectorList: VCSConnector[];
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();
const router = useRouter();
const currentUser = useCurrentUserV1();

const columnList = computed(() => [
  {
    title: t("common.project"),
  },
  {
    title: t("common.repository"),
  },
]);

const getProject = (connector: VCSConnector) => {
  const project = `${projectNamePrefix}${getVCSConnectorId(connector.name).projectId}`;
  return projectV1Store.getProjectByName(project);
};

const onClick = function (_: number, row: number) {
  const connector = props.connectorList[row];
  const project = getProject(connector);
  if (
    !hasProjectPermissionV2(project, currentUser.value, "bb.vcsConnectors.get")
  ) {
    return;
  }

  const { projectId, vcsConnectorId } = getVCSConnectorId(connector.name);
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS_DETAIL,
    params: {
      projectId,
      vcsConnectorId,
    },
  });
};
</script>
