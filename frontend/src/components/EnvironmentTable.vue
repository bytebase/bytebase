<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="environmentList"
    :showHeader="true"
    :leftBordered="false"
    :rightBordered="false"
    @click-row="clickEnvironment"
  >
    <template v-slot:body="{ rowData: environment }">
      <BBTableCell :leftPadding="4" class="w-4 table-cell text-gray-500">
        <span class="">#{{ environment.id }}</span>
      </BBTableCell>
      <BBTableCell class="w-48 table-cell">
        {{ environment.name }}
      </BBTableCell>
      <BBTableCell class="w-24 table-cell">
        {{ humanizeTs(environment.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-24 table-cell">
        {{ humanizeTs(environment.lastUpdatedTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { useRouter } from "vue-router";
import { Environment } from "../types";
import { environmentSlug } from "../utils";

const COLUMN_LIST = [
  {
    title: "ID",
  },
  {
    title: "Name",
  },
  {
    title: "Created",
  },
  {
    title: "Last updated",
  },
];

export default {
  name: "EnvironmentTable",
  components: {},
  props: {
    environmentList: {
      required: true,
      type: Object as PropType<Environment[]>,
    },
  },
  setup(props, ctx) {
    const router = useRouter();

    const clickEnvironment = function (section: number, row: number) {
      const environment = props.environmentList[row];
      router.push(`/environment/${environmentSlug(environment)}`);
    };

    return {
      COLUMN_LIST,
      clickEnvironment,
    };
  },
};
</script>
