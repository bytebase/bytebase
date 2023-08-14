<template>
  <div ref="wrapper" rule="database-table" v-bind="$attrs">
    <BBGrid
      :column-list="columnList"
      :data-source="pagedDataSource"
      :custom-header="true"
      :class="tableClass"
      :row-clickable="rowClickable"
      @click-row="clickDatabase"
    >
      <template #header>
        <div role="table-row" class="bb-grid-row bb-grid-header-row group">
          <div
            v-for="(column, index) in columnList"
            :key="index"
            role="table-cell"
            class="bb-grid-header-cell capitalize"
            :class="[column.class]"
          >
            <template v-if="showSelectionColumn && index === 0">
              <slot name="selection-all" :database-list="mixedDatabaseList" />
            </template>
            <template v-else>{{ column.title }}</template>
          </div>
        </div>
      </template>

      <template #item="{ item: database }: { item: Database }">
        <div v-if="showSelectionColumn" class="bb-grid-cell !px-2">
          <slot name="selection" :database="database" />
        </div>
        <div class="bb-grid-cell">
          <div class="flex items-center space-x-2">
            <SQLEditorButton
              :database="database"
              :disabled="!allowQuery(database)"
              :tooltip="true"
              @failed="handleGotoSQLEditorFailed"
            />
            <DatabaseName :database="database" tag="span" />
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
        </div>
        <div v-if="showSchemaVersionColumn" class="hidden lg:bb-grid-cell">
          {{ database.schemaVersion }}
        </div>
        <div v-if="showProjectColumn" class="bb-grid-cell">
          <div class="flex flex-row space-x-2 items-center">
            <div>{{ projectName(database.project) }}</div>
            <div
              v-if="showTenantIcon && database.project.tenantMode === 'TENANT'"
              class="tooltip-wrapper"
            >
              <span class="tooltip whitespace-nowrap">
                {{ $t("project.mode.batch") }}
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
                  {{ $t("database.gitops-enabled") }}
                </span>

                <GitIcon
                  class="w-4 h-4 text-control hover:text-control-hover"
                />
              </template>
            </div>
          </div>
        </div>
        <div v-if="showEnvironmentColumn" class="bb-grid-cell">
          <div class="flex items-center">
            {{ environmentName(database.instance.environment) }}
            <ProductionEnvironmentIcon
              class="ml-1"
              :environment="database.instance.environment"
            />
          </div>
        </div>
        <div v-if="showInstanceColumn" class="bb-grid-cell">
          <div class="flex flex-row items-center space-x-1">
            <InstanceEngineIcon :instance="database.instance" />
            <span class="flex-1 whitespace-pre-wrap">
              {{ instanceName(database.instance) }}
            </span>
          </div>
        </div>
        <div v-if="showMiscColumn" class="bb-grid-cell">
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
        </div>
      </template>

      <template #footer>
        <div
          v-if="hasReservedDatabases && !state.showReservedDatabaseList"
          class="flex items-center justify-center cursor-pointer hover:bg-gray-200 py-2 text-gray-400 text-sm"
          @click="showReservedDatabaseList()"
        >
          {{ $t("database.show-reserved-databases") }}
        </div>
      </template>
    </BBGrid>
  </div>

  <div
    v-if="showPagination"
    class="flex justify-end !mt-2"
    :class="paginationClass"
  >
    <NPagination
      :item-count="table.getCoreRowModel().rows.length"
      :page="table.getState().pagination.pageIndex + 1"
      :page-size="table.getState().pagination.pageSize"
      :show-quick-jumper="true"
      @update-page="handleChangePage"
      @update-page-size="(ps) => table.setPageSize(ps)"
    />
  </div>

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
import {
  ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useVueTable,
} from "@tanstack/vue-table";
import cloneDeep from "lodash-es/cloneDeep";
import { NTooltip, NPagination } from "naive-ui";
import { computed, nextTick, PropType, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { SQLEditorButton } from "@/components/DatabaseDetail";
import DatabaseName from "@/components/DatabaseName.vue";
import { getScrollParent } from "@/plugins/demo/utils";
import { useCurrentUserV1, useDatabaseV1Store } from "@/store";
import { BBGridColumn } from "../bbkit/types";
import { Database } from "../types";
import {
  databaseSlug,
  isDatabaseAccessible,
  isPITRDatabase,
  VueClass,
} from "../utils";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import TenantIcon from "./TenantIcon.vue";

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
  tableClass: {
    type: [String, Array, Object] as PropType<VueClass>,
    default: undefined,
  },
  paginationClass: {
    type: [String, Array, Object] as PropType<VueClass>,
    default: undefined,
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
  pageSize: {
    type: Number,
    default: 20,
  },
  scrollOnPageChange: {
    type: Boolean,
    default: true,
  },
  schemaless: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["select-database"]);

const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const databaseV1Store = useDatabaseV1Store();
const { t } = useI18n();
const state = reactive<State>({
  showIncorrectProjectModal: false,
  showReservedDatabaseList: false,
});
const wrapper = ref<HTMLElement>();

const sortedDatabaseList = computed(() => {
  const list = [...props.databaseList];
  list.sort((a, b) => {
    // Fallback to `id` DESC
    return -(+a.id - +b.id);
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
  const databaseList = [...regularDatabaseList.value];
  if (state.showReservedDatabaseList) {
    databaseList.push(...reservedDatabaseList.value);
  }
  return databaseList;
});

const columnListMap = computed(() => {
  const NAME = {
    title: t("common.name"),
    width: "minmax(auto, 1.5fr)",
  };
  const SCHEMA_VERSION = props.schemaless
    ? undefined
    : {
        title: t("common.schema-version"),
        width: { lg: "minmax(auto, 1fr)" },
        class: "hidden lg:flex",
      };
  const PROJECT = {
    title: t("common.project"),
    width: "minmax(auto, 1fr)",
  };
  const ENVIRONMENT = {
    title: t("common.environment"),
    width: "minmax(auto, 1fr)",
  };
  const INSTANCE = {
    title: t("common.instance"),
    width: "minmax(auto, 1fr)",
  };
  const SYNC_STATUS = {
    title: t("database.sync-status"),
    width: "auto",
    class: "items-center",
  };
  return new Map<Mode, (BBGridColumn | undefined)[]>([
    [
      "ALL",
      [NAME, SCHEMA_VERSION, PROJECT, ENVIRONMENT, INSTANCE, SYNC_STATUS],
    ],
    ["ALL_SHORT", [NAME, SCHEMA_VERSION, PROJECT, ENVIRONMENT, INSTANCE]],
    ["ALL_TINY", [NAME, PROJECT, ENVIRONMENT, INSTANCE]],
    ["INSTANCE", [NAME, SCHEMA_VERSION, PROJECT, SYNC_STATUS]],
    ["PROJECT", [NAME, SCHEMA_VERSION, ENVIRONMENT, INSTANCE, SYNC_STATUS]],
    ["PROJECT_SHORT", [NAME, SCHEMA_VERSION, ENVIRONMENT, INSTANCE]],
  ]);
});

const showSchemaVersionColumn = computed(() => {
  return props.mode !== "ALL_TINY" && !props.schemaless;
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
  const list = cloneDeep(columnListMap.value.get(props.mode)!).filter(Boolean);
  if (props.showSelectionColumn) {
    list.unshift({
      title: "",
      width: "minmax(auto, 2rem)",
      class: "items-center !px-2",
    });
  }
  return list as BBGridColumn[];
});

const table = useVueTable<Database>({
  get data() {
    return mixedDatabaseList.value;
  },
  get columns() {
    return columnList.value.map<ColumnDef<Database>>((col, index) => ({
      header: col.title!,
    }));
  },
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
});

const pagedDataSource = computed(() => {
  return table.getRowModel().rows.map((row) => row.original);
});

const showPagination = computed(() => {
  return mixedDatabaseList.value.length > props.pageSize;
});

const handleChangePage = (page: number) => {
  table.setPageIndex(page - 1);
  if (props.scrollOnPageChange && wrapper.value) {
    const parent = getScrollParent(wrapper.value);
    parent.scrollTo(0, 0);
  }
};
watch(
  () => props.pageSize,
  (ps) => {
    table.setPageSize(ps);
  },
  { immediate: true }
);

const showReservedDatabaseList = () => {
  const count = regularDatabaseList.value.length;
  const pageCount = table.getPageCount();
  const targetPage =
    count === pageCount * props.pageSize ? pageCount + 1 : pageCount;
  state.showReservedDatabaseList = true;
  nextTick(() => {
    handleChangePage(targetPage);
  });
};

const allowQuery = (database: Database) => {
  const composedDatabase = databaseV1Store.getDatabaseByUID(
    String(database.id)
  );
  return isDatabaseAccessible(composedDatabase, currentUserV1.value);
};

const showTenantIcon = computed(() => {
  return ["ALL", "ALL_SHORT", "INSTANCE"].includes(props.mode);
});

const handleGotoSQLEditorFailed = (database: Database) => {
  state.warningDatabase = database;
  state.showIncorrectProjectModal = true;
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

const clickDatabase = (
  database: Database,
  section: number,
  row: number,
  e: MouseEvent
) => {
  if (!props.rowClickable) return;

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
