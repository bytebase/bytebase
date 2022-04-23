<template>
  <BBTable
    :column-list="columnList"
    :data-source="databaseList"
    :show-header="true"
    :left-bordered="bordered"
    :right-bordered="bordered"
    :top-bordered="bordered"
    :bottom-bordered="bordered"
    @click-row="clickDatabase"
  >
    <template
      #body="{ rowData: database }: { rowData: (typeof databaseList)[number] }"
    >
      <BBTableCell :left-padding="4" class="w-16">
        <div class="flex flex-row items-center space-x-1 tooltip-wrapper">
          <span>{{ database.name }}</span>
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
      <BBTableCell v-if="showProjectColumn" class="w-16">
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
      <BBTableCell v-if="showEnvironmentColumn" class="w-16">
        {{ environmentName(database.instance.environment) }}
      </BBTableCell>
      <BBTableCell v-if="showInstanceColumn" class="w-32">
        <div class="flex flex-row items-center space-x-1">
          <InstanceEngineIcon :instance="database.instance" />
          <span>{{ instanceName(database.instance) }}</span>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-4">
        {{ database.syncStatus }}
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-16">
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </BBTableCell>
      <BBTableCell v-if="showSQLEditorLink" class="w-16">
        <button class="btn-icon" @click.stop="gotoSQLEditor(database)">
          <heroicons-outline:terminal class="w-4 h-4" />
        </button>
      </BBTableCell>
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
import { connectionSlug, databaseSlug } from "../utils";
import { Database, DEFAULT_PROJECT_ID, UNKNOWN_ID } from "../types";
import { BBTableColumn } from "../bbkit/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

type Mode = "ALL" | "ALL_SHORT" | "INSTANCE" | "PROJECT" | "PROJECT_SHORT";

interface State {
  showIncorrectProjectModal: boolean;
  warningDatabase?: Database;
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

const clickDatabase = (section: number, row: number) => {
  if (!props.rowClickable) return;

  const database = props.databaseList[row];
  if (props.customClick) {
    emit("select-database", database);
  } else {
    router.push(`/db/${databaseSlug(database)}`);
  }
};
</script>
