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
      <BBTableCell :leftPadding="4" class="w-12">
        {{ environmentName(database.instance.environment) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ projectName(database.project) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ database.name }}
      </BBTableCell>
      <BBTableCell class="w-24">
        {{ instanceName(database.instance) }}
      </BBTableCell>
      <BBTableCell class="w-8" v-database-sync-status>
        {{ database.syncStatus }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { databaseSlug } from "../utils";
import { Database } from "../types";

const COLUMN_LIST = [
  {
    title: "Environment",
  },
  {
    title: "Project",
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
