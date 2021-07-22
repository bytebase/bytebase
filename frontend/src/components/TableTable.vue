<template>
  <BBTable
    :columnList="columnList"
    :dataSource="tableList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    :rowClickable="mode == 'TABLE'"
    @click-row="clickTable"
  >
    <template v-slot:body="{ rowData: table }">
      <BBTableCell :leftPadding="4" class="w-16">
        {{ table.name }}
      </BBTableCell>
      <BBTableCell v-if="mode == 'TABLE'" class="w-8">
        {{ table.engine }}
      </BBTableCell>
      <BBTableCell v-if="mode == 'TABLE'" class="w-8">
        {{ table.rowCount }}
      </BBTableCell>
      <BBTableCell v-if="mode == 'TABLE'" class="w-8">
        {{ bytesToString(table.dataSize) }}
      </BBTableCell>
      <BBTableCell v-if="mode == 'TABLE'" class="w-8">
        {{ bytesToString(table.indexSize) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ table.syncStatus }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(table.lastSuccessfulSyncTs) }}
      </BBTableCell>
      <BBTableCell v-if="showConsoleLink" class="w-4">
        <button
          class="btn-icon"
          @click.stop="
            window.open(consoleLink(table.database.name, table.name), '_blank')
          "
        >
          <svg
            class="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            ></path>
          </svg>
        </button>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Table } from "../types";
import {
  bytesToString,
  databaseSlug,
  databaseTableConsoleLink,
} from "../utils";
import { useRouter } from "vue-router";
import { isEmpty } from "lodash";
import { useStore } from "vuex";

type Mode = "TABLE" | "VIEW";

const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
  [
    "TABLE",
    [
      {
        title: "Name",
      },
      {
        title: "Engine",
      },
      {
        title: "Row count est.",
      },
      {
        title: "Data size",
      },
      {
        title: "Index size",
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
    "VIEW",
    [
      {
        title: "Name",
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
  name: "TableTable",
  components: {},
  props: {
    mode: {
      default: "TABLE",
      type: String as PropType<Mode>,
    },
    tableList: {
      required: true,
      type: Object as PropType<Table[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const columnList = computed(() => {
      const list = columnListMap.get(props.mode);
      if (showConsoleLink.value) {
        list?.push({ title: "SQL console" });
      }

      return list;
    });

    const showConsoleLink = computed(() => {
      if (props.mode != "TABLE") {
        return false;
      }

      const consoleURL =
        store.getters["setting/settingByName"]("bb.console.table").value;
      return !isEmpty(consoleURL);
    });

    const consoleLink = (databaseName: string, tableName: string) => {
      const consoleURL =
        store.getters["setting/settingByName"]("bb.console.table").value;
      if (!isEmpty(consoleURL)) {
        return databaseTableConsoleLink(consoleURL, databaseName, tableName);
      }
      return "";
    };

    const clickTable = (section: number, row: number) => {
      const table = props.tableList[row];
      router.push(`/db/${databaseSlug(table.database)}/table/${table.name}`);
    };

    return {
      columnList,
      showConsoleLink,
      consoleLink,
      bytesToString,
      clickTable,
    };
  },
};
</script>
