<template>
  <BBTable
    :column-list="columnList"
    :data-source="mixedDatabaseList"
    :show-header="true"
    :left-bordered="bordered"
    :right-bordered="bordered"
    :top-bordered="bordered"
    :bottom-bordered="bordered"
    data-label="bb-database-table"
    @click-row="clickDatabase"
  >
    <template #body="{ rowData: database }: { rowData: Database }">
      <BBTableCell v-if="showSelectionColumn" class="w-[1%]">
        <!-- width: 1% means as narrow as possible -->
        <slot name="selection" :database="database" />
      </BBTableCell>
      <BBTableCell :left-padding="showSelectionColumn ? 2 : 4" class="w-[25%]">
        <div class="flex items-center space-x-2 tooltip-wrapper">
          <span>{{ database.name }}</span>
          <BBBadge
            v-if="isPITRDatabase(database)"
            text="PITR"
            :can-remove="false"
            class="text-xs"
          />
          <div v-if="!showMiscColumn && database.syncStatus != 'OK'">
            <span class="tooltip">
              {{
                $t("database.last-sync-status-long", [
                  database.syncStatus,
                  humanizeTs(database.lastSuccessfulSyncTs),
                ])
              }}
            </span>
            <heroicons-outline:exclamation-circle class="w-5 h-5 text-error" />
          </div>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showProjectColumn" class="w-[20%]">
        <div class="flex flex-row space-x-2 items-center">
          <div>{{ projectName(database.project) }}</div>
          <div class="tooltip-wrapper">
            <svg
              v-if="database.project.workflowType == 'UI'"
              class="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            ></svg>
            <template v-else-if="database.project.workflowType == 'VCS'">
              <span class="tooltip whitespace-nowrap">
                {{ $t("database.version-control-enabled") }}
              </span>
              <heroicons-outline:collection
                class="w-4 h-4 text-control hover:text-control-hover"
              />
            </template>
          </div>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showEnvironmentColumn" class="w-[10%]">
        {{ environmentName(database.instance.environment) }}
      </BBTableCell>
      <BBTableCell v-if="showInstanceColumn" class="w-[25%]">
        <div class="flex flex-row items-center space-x-1">
          <InstanceEngineIcon :instance="database.instance" />
          <span class="flex-1 whitespace-pre-wrap">
            {{ instanceName(database.instance) }}
          </span>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-[8%]">
        <span
          :class="{
            'text-error': database.syncStatus === 'NOT_FOUND',
          }"
        >
          {{ database.syncStatus }}
        </span>
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-[14%]">
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </BBTableCell>
      <BBTableCell v-if="showSQLEditorLink" class="w-[8%]">
        <button class="btn-icon" @click.stop="gotoSQLEditor(database)">
          <heroicons-outline:terminal class="w-4 h-4" />
        </button>
      </BBTableCell>
    </template>

    <template
      v-if="hasReservedDatabases && !state.showReservedDatabaseList"
      #footer
    >
      <tfoot>
        <tr>
          <td :colspan="columnList.length" class="p-0">
            <div
              class="flex items-center justify-center cursor-pointer hover:bg-gray-200 py-2 text-gray-400 text-sm"
              @click="state.showReservedDatabaseList = true"
            >
              {{ $t("database.show-reserved-databases") }}
            </div>
          </td>
        </tr>
      </tfoot>
    </template>
  </BBTable>

  <BBModal
    v-if="state.showIncorrectProjectModal"
    :title="$t('common.warning')"
    @close="handleIncorrectProjectModalCancel"
  >
    <div class="col-span-1 w-96">
      {{ $t("database.incorrect-project-warning") }}
    </div>
    <div class="pt-6 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="handleIncorrectProjectModalCancel"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="handleIncorrectProjectModalConfirm"
      >
        {{ $t("database.go-to-transfer") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useRouter } from "vue-router";
import { connectionSlug, databaseSlug, isPITRDatabase } from "../utils";
import type { Database } from "../types";
import { DEFAULT_PROJECT_ID, UNKNOWN_ID } from "../types";
import { BBTableColumn } from "../bbkit/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

type Mode = "ALL" | "ALL_SHORT" | "INSTANCE" | "PROJECT" | "PROJECT_SHORT";

interface State {
  showIncorrectProjectModal: boolean;
  warningDatabase?: Database;
  showReservedDatabaseList: boolean;
}

const props = defineProps({
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
  showSelectionColumn: {
    type: Boolean,
    default: false,
  },
  rowClickable: {
    default: true,
    type: Boolean,
  },
  customClick: {
    default: false,
    type: Boolean,
  },
  databaseList: {
    required: true,
    type: Object as PropType<Database[]>,
  },
});

const emit = defineEmits(["select-database"]);

const router = useRouter();
const { t } = useI18n();
const state = reactive<State>({
  showIncorrectProjectModal: false,
  showReservedDatabaseList: false,
});

const sortedDatabaseList = computed(() => {
  const list = [...props.databaseList];
  list.sort((a, b) => {
    if (a.syncStatus === "NOT_FOUND" && b.syncStatus === "OK") {
      return -1;
    }
    if (a.syncStatus === "OK" && b.syncStatus === "NOT_FOUND") {
      return 1;
    }
    return a.id - b.id;
  });
  return list;
});

const regularDatabaseList = computed(() =>
  sortedDatabaseList.value.filter((db) => !isPITRDatabase(db))
);
const reservedDatabaseList = computed(() =>
  sortedDatabaseList.value.filter((db) => isPITRDatabase(db))
);
const hasReservedDatabases = computed(
  () => reservedDatabaseList.value.length > 0
);

const mixedDatabaseList = computed(() => {
  const tableList = [...regularDatabaseList.value];
  if (state.showReservedDatabaseList) {
    tableList.push(...reservedDatabaseList.value);
  }

  return tableList;
});

const columnListMap = computed(() => {
  return new Map([
    [
      "ALL",
      [
        {
          title: t("common.name"),
        },
        {
          title: t("common.project"),
        },
        {
          title: t("common.environment"),
        },
        {
          title: t("common.instance"),
        },
        {
          title: t("db.sync-status"),
        },
        {
          title: t("db.last-successful-sync"),
        },
      ],
    ],
    [
      "ALL_SHORT",
      [
        {
          title: t("common.name"),
        },
        {
          title: t("common.project"),
        },
        {
          title: t("common.environment"),
        },
        {
          title: t("common.instance"),
        },
      ],
    ],
    [
      "INSTANCE",
      [
        {
          title: t("common.name"),
        },
        {
          title: t("common.project"),
        },
        {
          title: t("db.sync-status"),
        },
        {
          title: t("db.last-successful-sync"),
        },
      ],
    ],
    [
      "PROJECT",
      [
        {
          title: t("common.name"),
        },
        {
          title: t("common.environment"),
        },
        {
          title: t("common.instance"),
        },
        {
          title: t("db.sync-status"),
        },
        {
          title: t("db.last-successful-sync"),
        },
      ],
    ],
    [
      "PROJECT_SHORT",
      [
        {
          title: t("common.name"),
        },
        {
          title: t("common.environment"),
        },
        {
          title: t("common.instance"),
        },
      ],
    ],
  ]);
});

const showInstanceColumn = computed(() => {
  return props.mode != "INSTANCE";
});

const showProjectColumn = computed(() => {
  return props.mode != "PROJECT" && props.mode != "PROJECT_SHORT";
});

const showEnvironmentColumn = computed(() => {
  return props.mode != "INSTANCE";
});

const showMiscColumn = computed(() => {
  return props.mode != "ALL_SHORT" && props.mode != "PROJECT_SHORT";
});

const columnList = computed(() => {
  var list: BBTableColumn[] = columnListMap.value.get(props.mode)!;
  if (showSQLEditorLink.value) {
    // Use cloneDeep, otherwise it will alter the one in columnListMap
    list = cloneDeep(list);
    list.push({ title: t("sql-editor.self") });
  }

  if (props.showSelectionColumn) {
    list.unshift({ title: "" });
  }
  return list;
});

const showSQLEditorLink = computed(() => {
  if (props.mode == "ALL_SHORT" || props.mode == "PROJECT_SHORT") {
    return false;
  }
  return true;
});

const gotoSQLEditor = (database: Database) => {
  // SQL editors can only query databases in the projects available to the user.
  if (
    database.projectId === UNKNOWN_ID ||
    database.projectId === DEFAULT_PROJECT_ID
  ) {
    state.warningDatabase = database;
    state.showIncorrectProjectModal = true;
  } else {
    router.push({
      name: "sql-editor.detail",
      params: {
        connectionSlug: connectionSlug(database),
      },
    });
  }
};

const handleIncorrectProjectModalConfirm = () => {
  if (state.warningDatabase) {
    router.push(`/db/${databaseSlug(state.warningDatabase)}`);
  }
};

const handleIncorrectProjectModalCancel = () => {
  state.showIncorrectProjectModal = false;
  state.warningDatabase = undefined;
};

const clickDatabase = (section: number, row: number, e: MouseEvent) => {
  if (!props.rowClickable) return;

  const database = mixedDatabaseList.value[row];
  if (props.customClick) {
    emit("select-database", database);
  } else {
    const url = `/db/${databaseSlug(database)}`;
    if (e.ctrlKey || e.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }
};
</script>
