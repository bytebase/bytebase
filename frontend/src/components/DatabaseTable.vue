<template>
  <BBTable
    :columnList="columnList"
    :dataSource="databaseList"
    :showHeader="true"
    :leftBordered="bordered"
    :rightBordered="bordered"
    @click-row="clickDatabase"
  >
    <template v-slot:body="{ rowData: database }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ database.name }}
      </BBTableCell>
      <BBTableCell v-if="mode != 'PROJECT'" class="w-16">
        {{ projectName(database.project) }}
      </BBTableCell>
      <BBTableCell v-if="mode != 'INSTANCE'" class="w-12">
        {{ environmentName(database.instance.environment) }}
      </BBTableCell>
      <BBTableCell v-if="mode == 'ALL'" class="w-24">
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
import { BBTableColumn } from "../bbkit/types";

type Mode = "ALL" | "INSTANCE" | "PROJECT";

const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
  [
    "ALL",
    [
      {
        title: "Name",
      },
      {
        title: "Project",
      },
      {
        title: "Environment",
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
    ],
  ],
  [
    "INSTANCE",
    [
      {
        title: "Name",
      },
      {
        title: "Project",
      },
      {
        title: "Sync status",
      },
      {
        title: "Last successful sync",
      },
    ],
  ],
  [
    "PROJECT",
    [
      {
        title: "Name",
      },
      {
        title: "Environment",
      },
      {
        title: "Sync status",
      },
      {
        title: "Last successful sync",
      },
    ],
  ],
]);

export default {
  name: "DatabaseTable",
  components: {},
  props: {
    bordered: {
      default: true,
      type: Boolean,
    },
    mode: {
      default: "ALL",
      type: String as PropType<Mode>,
    },
    singleInstance: {
      default: true,
      type: Boolean,
    },
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
      columnList: columnListMap.get(props.mode),
      clickDatabase,
    };
  },
};
</script>
