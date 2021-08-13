<template>
  <BBTable
    :columnList="columnList"
    :dataSource="backupList"
    :showHeader="true"
    :rowClickable="false"
    :leftBordered="true"
    :rightBordered="true"
  >
    <template v-slot:body="{ rowData: backup }">
      <BBTableCell :leftPadding="4" class="w-4">
        <span
          class="flex items-center justify-center rounded-full select-none"
          :class="statusIconClass(backup)"
        >
          <template v-if="backup.status == 'PENDING_CREATE'">
            <span
              class="h-2 w-2 bg-blue-600 hover:bg-blue-700 rounded-full"
              style="
                animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
              "
            >
            </span>
          </template>
          <template v-else-if="backup.status == 'DONE'">
            <svg
              class="w-4 h-4"
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              aria-hidden="true"
            >
              <path
                fill-rule="evenodd"
                d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                clip-rule="evenodd"
              />
            </svg>
          </template>
          <template v-else-if="backup.status == 'FAILED'">
            <span
              class="
                h-2
                w-2
                rounded-full
                text-center
                pb-6
                font-normal
                text-base
              "
              aria-hidden="true"
              >!</span
            >
          </template>
        </span>
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ backup.name }}
      </BBTableCell>
      <BBTableCell class="w-48">
        {{ backup.path }}
      </BBTableCell>
      <BBTableCell class="w-8 whitespace-nowrap tooltip-wrapper">
        <span v-if="backup.comment.length > 30" class="tooltip">{{
          backup.comment
        }}</span>
        {{
          backup.comment.length > 30
            ? backup.comment.substring(0, 30) + "..."
            : backup.comment
        }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(backup.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ backup.creator.name }}
      </BBTableCell>
      <BBTableCell class="w-4">
        <BBButtonConfirm
          v-if="backup.status == 'DONE'"
          :style="'RESTORE'"
          :requireConfirm="true"
          :okText="'Restore'"
          :confirmTitle="`Are you sure to restore '${backup.name}' to database '${database.name}'?`"
          :confirmDescription="''"
          @confirm="restoreBackup(backup)"
        />
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { Backup, Database } from "../types";
import { bytesToString } from "../utils";
import { useStore } from "vuex";

const columnList: BBTableColumn[] = [
  {
    title: "Status",
  },
  {
    title: "Name",
  },
  {
    title: "Path",
  },
  {
    title: "Comment",
  },
  {
    title: "Time",
  },
  {
    title: "Creator",
  },
  {
    title: "Restore",
  },
];

export default {
  name: "BackupTable",
  components: {},
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    backupList: {
      required: true,
      type: Object as PropType<Backup[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const statusIconClass = (backup: Backup) => {
      let iconClass = "w-5 h-5";
      switch (backup.status) {
        case "PENDING_CREATE":
          return (
            iconClass +
            " bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700"
          );
        case "DONE":
          return iconClass + " bg-success hover:bg-success-hover text-white";
        case "FAILED":
          return (
            iconClass +
            " bg-error text-white hover:text-white hover:bg-error-hover"
          );
      }
    };

    const restoreBackup = (backup: Backup) => {
      store.dispatch("backup/restoreFromBackup", {
        databaseId: props.database.id,
        backupId: backup.id,
      });
    };

    return {
      columnList,
      bytesToString,
      statusIconClass,
      restoreBackup,
    };
  },
};
</script>
