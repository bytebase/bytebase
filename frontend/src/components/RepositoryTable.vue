<template>
  <BBTable
    :column-list="columnList"
    :data-source="repositoryList"
    :show-header="true"
    :left-bordered="true"
    :right-bordered="true"
    @click-row="clickRepository"
  >
    <template #body="{ rowData: repository }">
      <BBTableCell :left-padding="4" class="w-16">
        {{ projectV1Name(repository.project) }}
      </BBTableCell>
      <BBTableCell class="w-32">
        {{ repository.fullPath }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_GITOPS } from "@/router/dashboard/projectV1";
import { getProjectName } from "@/store/modules/v1/common";
import { ComposedRepository } from "@/types";
import { projectV1Name } from "../utils";

const props = defineProps({
  repositoryList: {
    required: true,
    type: Object as PropType<ComposedRepository[]>,
  },
});
const { t } = useI18n();

const router = useRouter();

const columnList = computed(() => [
  {
    title: t("common.project"),
  },
  {
    title: t("common.repository"),
  },
]);

const clickRepository = function (_: number, row: number) {
  const repository = props.repositoryList[row];
  router.push({
    name: PROJECT_V1_ROUTE_GITOPS,
    params: {
      projectId: getProjectName(repository.project.name),
    },
  });
};
</script>
