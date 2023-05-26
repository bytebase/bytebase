<template>
  <div ref="wrapper" rule="database-table" v-bind="$attrs">
    <BBGrid
      :column-list="columnList"
      :data-source="pagedDataSource"
      :custom-header="true"
      :class="tableClass"
      :row-clickable="rowClickable"
      :show-placeholder="showPlaceholder"
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

      <template #item="{ item: database }: { item: ComposedDatabase }">
        <div v-if="showSelectionColumn" class="bb-grid-cell !px-2">
          <slot name="selection" :database="database" />
        </div>
        <div class="bb-grid-cell">
          <div class="flex items-center space-x-2">
            <SQLEditorButtonV1
              :database="database"
              :disabled="!allowQuery(database)"
              :tooltip="true"
              @failed="handleGotoSQLEditorFailed"
            />
            <DatabaseV1Name :database="database" :link="false" tag="span" />
            <BBBadge
              v-if="isPITRDatabaseV1(database)"
              text="PITR"
              :can-remove="false"
              class="text-xs"
            />
            <NTooltip
              v-if="!showMiscColumn && database.syncState !== State.ACTIVE"
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
                    "NOT_FOUND",
                    humanizeDate(database.successfulSyncTime),
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
            <ProjectV1Name
              :project="database.projectEntity"
              :link="false"
              tag="div"
            />
            <div
              v-if="
                showTenantIcon &&
                database.projectEntity.tenantMode ===
                  TenantMode.TENANT_MODE_ENABLED
              "
              class="tooltip-wrapper"
            >
              <span class="tooltip whitespace-nowrap">
                {{ $t("project.mode.tenant") }}
              </span>
              <TenantIcon class="w-4 h-4 text-control" />
            </div>
            <div class="tooltip-wrapper">
              <svg
                v-if="database.projectEntity.workflow === Workflow.UI"
                class="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              ></svg>
              <template
                v-else-if="database.projectEntity.workflow === Workflow.VCS"
              >
                <span v-if="mode === 'ALL_SHORT'" class="tooltip w-40">
                  {{ $t("alter-schema.vcs-info") }}
                </span>
                <span v-else class="tooltip whitespace-nowrap">
                  {{ $t("database.gitops-enabled") }}
                </span>

                <heroicons-outline:collection
                  class="w-4 h-4 text-control hover:text-control-hover"
                />
              </template>
            </div>
          </div>
        </div>
        <div v-if="showEnvironmentColumn" class="bb-grid-cell">
          <EnvironmentV1Name
            :environment="database.instanceEntity.environmentEntity"
            :link="false"
            tag="div"
          />
        </div>
        <div v-if="showInstanceColumn" class="bb-grid-cell">
          <InstanceV1Name
            :instance="database.instanceEntity"
            :link="false"
            tag="div"
          />
        </div>
        <div v-if="showMiscColumn" class="bb-grid-cell">
          <div class="w-full flex justify-center">
            <NTooltip placement="left">
              <template #trigger>
                <div
                  class="flex items-center justify-center rounded-full select-none w-5 h-5 overflow-hidden text-white font-medium text-base"
                  :class="
                    database.syncState === State.ACTIVE
                      ? 'bg-success'
                      : 'bg-error'
                  "
                >
                  <template v-if="database.syncState === State.ACTIVE">
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
                <template v-if="database.syncState === State.ACTIVE">
                  {{
                    $t("database.synced-at", {
                      time: humanizeDate(database.successfulSyncTime),
                    })
                  }}
                </template>
                <template v-else>
                  {{
                    $t("database.not-found-last-successful-sync-was", {
                      time: humanizeDate(database.successfulSyncTime),
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
  computed,
  nextTick,
  PropType,
  reactive,
  ref,
  watch,
  watchEffect,
} from "vue";
import { useRouter } from "vue-router";
import { NTooltip, NPagination } from "naive-ui";
import { useI18n } from "vue-i18n";
import cloneDeep from "lodash-es/cloneDeep";
import {
  ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useVueTable,
} from "@tanstack/vue-table";

import {
  databaseV1Slug,
  isDatabaseV1Accessible,
  isPITRDatabaseV1,
  VueClass,
} from "@/utils";
import { ComposedDatabase } from "@/types";
import { BBGridColumn } from "@/bbkit/types";
import TenantIcon from "@/components/TenantIcon.vue";
import { SQLEditorButtonV1 } from "@/components/DatabaseDetail";
import { DatabaseV1Name, InstanceV1Name, EnvironmentV1Name } from ".";
import { useCurrentUserV1 } from "@/store";
import { getScrollParent } from "@/plugins/demo/utils";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { State } from "@/types/proto/v1/common";
import { TenantMode, Workflow } from "@/types/proto/v1/project_service";

type Mode =
  | "ALL"
  | "ALL_SHORT"
  | "ALL_TINY"
  | "INSTANCE"
  | "PROJECT"
  | "PROJECT_SHORT";

interface LocalState {
  showIncorrectProjectModal: boolean;
  warningDatabase?: ComposedDatabase;
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
    type: Object as PropType<ComposedDatabase[]>,
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
  showPlaceholder: {
    type: Boolean,
    default: undefined,
  },
});

const emit = defineEmits(["select-database"]);

const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const { t } = useI18n();
const state = reactive<LocalState>({
  showIncorrectProjectModal: false,
  showReservedDatabaseList: false,
});
const wrapper = ref<HTMLElement>();

const regularDatabaseList = computed(() =>
  props.databaseList.filter((db) => !isPITRDatabaseV1(db))
);
const reservedDatabaseList = computed(() =>
  props.databaseList.filter((db) => isPITRDatabaseV1(db))
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

const policyList = ref<Policy[]>([]);

const preparePolicyList = () => {
  if (showSQLEditorLink.value) {
    usePolicyV1Store()
      .fetchPolicies({
        resourceType: PolicyResourceType.DATABASE,
        policyType: PolicyType.ACCESS_CONTROL,
      })
      .then((list) => (policyList.value = list));
  }
};

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

const table = useVueTable<ComposedDatabase>({
  get data() {
    return mixedDatabaseList.value;
  },
  get columns() {
    return columnList.value.map<ColumnDef<ComposedDatabase>>((col, index) => ({
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

const allowQuery = (database: ComposedDatabase) => {
  return isDatabaseV1Accessible(
    database,
    policyList.value,
    currentUserV1.value
  );
};

const showTenantIcon = computed(() => {
  return ["ALL", "ALL_SHORT", "INSTANCE"].includes(props.mode);
});

const handleGotoSQLEditorFailed = (database: ComposedDatabase) => {
  state.warningDatabase = database;
  state.showIncorrectProjectModal = true;
};

const handleIncorrectProjectModalConfirm = () => {
  if (state.warningDatabase) {
    router.push(`/db/${databaseV1Slug(state.warningDatabase)}`);
  }
};

const handleIncorrectProjectModalCancel = () => {
  state.showIncorrectProjectModal = false;
  state.warningDatabase = undefined;
};

const clickDatabase = (
  database: ComposedDatabase,
  section: number,
  row: number,
  e: MouseEvent
) => {
  if (!props.rowClickable) return;

  if (props.customClick) {
    emit("select-database", database);
  } else {
    const url = `/db/${databaseV1Slug(database)}`;
    if (e.ctrlKey || e.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }
};

watchEffect(preparePolicyList);
</script>
