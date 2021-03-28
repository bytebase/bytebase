<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="databaseList"
    :showHeader="true"
    :leftBordered="false"
    :rightBordered="false"
    @click-row="clickDatabase"
  >
    <template v-slot:body="{ rowData: database }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ database.instance.environment.name }}
      </BBTableCell>
      <BBTableCell class="w-36">
        {{ database.name }}
      </BBTableCell>
      <BBTableCell class="w-48">
        {{ database.instance.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ database.syncStatus }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(database.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { BBTableColumn } from "../bbkit/types";
import { databaseSlug, instanceSlug } from "../utils";
import { EnvironmentId, Database } from "../types";

const COLUMN_LIST = [
  {
    title: "Environment",
  },
  {
    title: "Name",
  },
  {
    title: "Instance",
  },
  {
    title: "Sync status",
  },
  {
    title: "Last successful sync",
  },
  {
    title: "Created",
  },
];

export default {
  name: "DatabaseTable",
  components: {},
  props: {
    databaseList: {
      required: true,
      type: Object as PropType<Database[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const clickDatabase = function (section: number, row: number) {
      const database = props.databaseList[row];
      router.push(`/db/${databaseSlug(database)}`);
    };

    return {
      COLUMN_LIST,
      clickDatabase,
    };
  },
};
</script>
