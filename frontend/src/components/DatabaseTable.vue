<template>
  <BBTable
    :column-list="columnList"
    :data-source="mixedDatabaseList"
    :show-header="true"
    :left-bordered="bordered"
    :right-bordered="bordered"
    :top-bordered="bordered"
    :bottom-bordered="bordered"
    :custom-header="true"
    header-class="capitalize"
    data-label="bb-database-table"
    @click-row="clickDatabase"
  >
    <template #header>
      <tr>
        <th
          v-for="(column, index) in columnList"
          :key="index"
          scope="col"
          class="py-2 text-left text-xs font-medium text-gray-500 tracking-wider capitalize"
          :class="[
            !showSelectionColumn && index === 0 ? 'pl-4' : 'pl-2',
            column.center && 'text-center pr-2',
          ]"
        >
          <template v-if="showSelectionColumn && index === 0">
            <slot name="selection-all" :database-list="mixedDatabaseList" />
          </template>
          <template v-else>{{ column.title }}</template>
        </th>
      </tr>
    </template>

    <template #body="{ rowData: database }: { rowData: Database }">
      <BBTableCell v-if="showSelectionColumn" class="w-[1%]">
        <!-- width: 1% means as narrow as possible -->
        <slot name="selection" :database="database" />
      </BBTableCell>
      <BBTableCell :left-padding="showSelectionColumn ? 2 : 4" class="w-[25%]">
        <div class="flex items-center space-x-2">
          <button
            v-if="showSQLEditorLink"
            class="btn-icon tooltip-wrapper disabled:hover:text-control"
            :disabled="isSQLEditorLinkDisabled(database)"
            @click.stop="gotoSQLEditor(database)"
          >
            <heroicons-solid:terminal class="w-5 h-5" />
            <div
              v-if="!isSQLEditorLinkDisabled(database)"
              class="tooltip whitespace-nowrap"
            >
              {{ $t("sql-editor.self") }}
            </div>
          </button>
          <span>{{ database.name }}</span>
          <BBBadge
            v-if="isPITRDatabase(database)"
            text="PITR"
            :can-remove="false"
            class="text-xs"
          />
          <NTooltip
            v-if="!showMiscColumn && database.syncStatus != 'OK'"
            placement="right"
          >
            <template #trigger>
              <heroicons-outline:exclamation-circle
                class="w-5 h-5 text-error"
              />
            </template>

            <div class="whitespace-nowrap">
              {{
                $t("database.last-sync-status-long", [
                  database.syncStatus,
                  humanizeTs(database.lastSuccessfulSyncTs),
                ])
              }}
            </div>
          </NTooltip>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showSchemaVersionColumn" class="w-[10%]">
        {{ database.schemaVersion }}
      </BBTableCell>
      <BBTableCell v-if="showProjectColumn" class="w-[15%]">
        <div class="flex flex-row space-x-2 items-center">
          <div>{{ projectName(database.project) }}</div>
          <div
            v-if="showTenantIcon && database.project.tenantMode === 'TENANT'"
            class="tooltip-wrapper"
          >
            <span class="tooltip whitespace-nowrap">
              {{ $t("project.mode.tenant") }}
            </span>
            <TenantIcon class="w-4 h-4 text-control" />
          </div>
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
              <span v-if="mode === 'ALL_SHORT'" class="tooltip w-40">
                {{ $t("alter-schema.vcs-info") }}
              </span>
              <span v-else class="tooltip whitespace-nowrap">
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
        <div class="flex items-center">
          {{ environmentName(database.instance.environment) }}
          <ProtectedEnvironmentIcon
            class="ml-1"
            :environment="database.instance.environment"
          />
        </div>
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
        <div class="w-full flex justify-center">
          <NTooltip placement="left">
            <template #trigger>
              <div
                class="flex items-center justify-center rounded-full select-none w-5 h-5 overflow-hidden text-white font-medium text-base"
                :class="
                  database.syncStatus === 'OK' ? 'bg-success' : 'bg-error'
                "
              >
                <template v-if="database.syncStatus === 'OK'">
                  <heroicons-solid:check class="w-4 h-4" />
                </template>
                <template v-else>
                  <span
                    class="h-2 w-2 flex items-center justify-center"
                    aria-hidden="true"
                    >!</span
                  >
                </template>
              </div>
            </template>

            <span>
              <template v-if="database.syncStatus === 'OK'">
                {{
                  $t("database.synced-at", {
                    time: humanizeTs(database.lastSuccessfulSyncTs),
                  })
                }}
              </template>
              <template v-else>
                {{
                  $t("database.not-found-last-successful-sync-was", {
                    time: humanizeTs(database.lastSuccessfulSyncTs),
                  })
                }}
              </template>
            </span>
          </NTooltip>
        </div>
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
import { NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";
import cloneDeep from "lodash-es/cloneDeep";
import {
  connectionSlug,
  databaseSlug,
  isDatabaseAccessible,
  isPITRDatabase,
} from "../utils";
import type { Database, Policy } from "../types";
import { DEFAULT_PROJECT_ID, UNKNOWN_ID } from "../types";
import { BBTableColumn } from "../bbkit/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import TenantIcon from "./TenantIcon.vue";
import { useCurrentUser } from "@/store";

type Mode =
  | "ALL"
  | "ALL_SHORT"
  | "ALL_TINY"
  | "INSTANCE"
  | "PROJECT"
  | "PROJECT_SHORT";

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
  policyList: {
    type: Array as PropType<Policy[]>,
    default: () => [],
  },
});

const emit = defineEmits(["select-database"]);

const router = useRouter();
const currentUser = useCurrentUser();
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
    return b.createdTs - a.createdTs;
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
          title: t("common.schema-version"),
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
          title: t("database.sync-status"),
          center: true,
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
          title: t("common.schema-version"),
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
      "ALL_TINY",
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
          title: t("common.schema-version"),
        },
        {
          title: t("common.project"),
        },
        {
          title: t("database.sync-status"),
          center: true,
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
          title: t("common.schema-version"),
        },
        {
          title: t("common.environment"),
        },
        {
          title: t("common.instance"),
        },
        {
          title: t("database.sync-status"),
          center: true,
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
          title: t("common.schema-version"),
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

const showSchemaVersionColumn = computed(() => {
  return props.mode !== "ALL_TINY";
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
  if (
    props.mode === "ALL_SHORT" ||
    props.mode === "ALL_TINY" ||
    props.mode === "PROJECT_SHORT"
  ) {
    return false;
  }
  return true;
});

const columnList = computed(() => {
  const list: BBTableColumn[] = cloneDeep(columnListMap.value.get(props.mode)!);
  if (props.showSelectionColumn) {
    list.unshift({ title: "" });
  }
  return list;
});

const showSQLEditorLink = computed(() => {
  if (
    props.mode === "ALL_SHORT" ||
    props.mode === "ALL_TINY" ||
    props.mode === "PROJECT_SHORT"
  ) {
    return false;
  }
  return true;
});

const isSQLEditorLinkDisabled = (database: Database) => {
  return !isDatabaseAccessible(database, props.policyList, currentUser.value);
};

const showTenantIcon = computed(() => {
  return ["ALL", "ALL_SHORT", "INSTANCE"].includes(props.mode);
});

const gotoSQLEditor = (database: Database) => {
  if (isSQLEditorLinkDisabled(database)) {
    return;
  }
  // SQL editors can only query databases in the projects available to the user.
  if (
    database.projectId === UNKNOWN_ID ||
    database.projectId === DEFAULT_PROJECT_ID
  ) {
    state.warningDatabase = database;
    state.showIncorrectProjectModal = true;
  } else {
    const url = `/sql-editor/${connectionSlug(database.instance, database)}`;
    window.open(url);
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
