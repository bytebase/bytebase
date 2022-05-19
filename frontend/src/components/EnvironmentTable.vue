<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="environmentList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="clickEnvironment"
  >
    <template #body="{ rowData: environment }">
      <BBTableCell :left-padding="4" class="w-4 table-cell text-gray-500">
        <span class="">#{{ environment.id }}</span>
      </BBTableCell>
      <BBTableCell class="w-48 table-cell">
        {{ environmentName(environment) }}
      </BBTableCell>
      <BBTableCell class="w-24 table-cell">
        {{ humanizeTs(environment.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-24 table-cell">
        {{ humanizeTs(environment.updatedTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { useRouter } from "vue-router";
import { Environment } from "../types";
import { environmentSlug } from "../utils";
import { useI18n } from "vue-i18n";

const props = defineProps({
  environmentList: {
    required: true,
    type: Object as PropType<Environment[]>,
  },
});

const router = useRouter();

const { t } = useI18n();

const COLUMN_LIST = [
  {
    title: t("common.id"),
  },
  {
    title: t("common.name"),
  },
  {
    title: t("common.created-at"),
  },
  {
    title: t("common.updated-at"),
  },
];

const clickEnvironment = function (section: number, row: number) {
  const environment = props.environmentList[row];
  router.push(`/environment/${environmentSlug(environment)}`);
};
</script>
