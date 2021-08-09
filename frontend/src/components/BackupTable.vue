<template>
  <BBTable
    :columnList="columnList"
    :dataSource="backupList"
    :showHeader="true"
    :leftBordered="true"
    :rightBordered="true"
    @click-row="clickBackup"
  >
    <template v-slot:body="{ rowData: backup }">
      <BBTableCell :leftPadding="4" class="w-8">
        {{ backup.name }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ backup.path }}
      </BBTableCell>
      <BBTableCell class="w-4">
        {{ backup.status }}
      </BBTableCell>
      <BBTableCell class="w-4">
        {{ humanizeTs(backup.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-4">
        {{ backup.creator }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Backup } from "../types";
import { bytesToString, databaseSlug } from "../utils";
import { useRouter } from "vue-router";
import { useStore } from "vuex";

const columnList: BBTableColumn[] = 
    [
      {
        title: "Name",
      },
      {
        title: "Path",
      },
      {
        title: "Status",
      },
      {
        title: "Time",
      },
      {
        title: "Creator",
      },
    ];

export default {
  name: "BackupTable",
  components: {},
  props: {
    backupList: {
      required: true,
      type: Object as PropType<Backup[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const clickBackup = (section: number, row: number) => {
      const backup = props.backupList[row];
      store.dispatch("backup/restoreFromBackup", {
        databaseId: backup.database.id,
        backupName: backup.name,
      });
    };

    return {
      columnList,
      bytesToString,
      clickBackup,
    };
  },
};
</script>
