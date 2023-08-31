<template>
  <div class="flex flex-col space-y-4">
    <div
      class="w-full flex flex-row justify-between items-center text-lg leading-6 font-medium text-main space-x-2"
    >
      <div class="flex flex-row justify-start items-center space-x-2">
        <span>{{ $t("change-history.self") }}</span>
        <BBSpin
          v-if="state.loading"
          :title="$t('change-history.refreshing-history')"
        />
      </div>
      <div class="flex flex-row justify-end items-center grow space-x-2">
        <div class="w-44">
          <BBSelect
            :selected-item="state.selectedAffectedTable"
            :item-list="affectedTables"
            @select-item="(item: AffectedTable) => state.selectedAffectedTable = item"
          >
            <template #menuItem="{ item }">
              <span
                class="block w-full truncate"
                :class="item.dropped && 'text-gray-400'"
              >
                {{ getAffectedTableDisplayName(item) }}
              </span>
            </template>
          </BBSelect>
        </div>
        <BBTooltipButton
          type="normal"
          :disabled="!allowExportChangeHistory || state.isExporting"
          tooltip-mode="DISABLED-ONLY"
          @click="handleExportChangeHistory"
        >
          {{ $t("change-history.export") }}
          <template #tooltip>
            <div class="whitespace-pre-line">
              {{ $t("change-history.need-to-select-first") }}
            </div>
          </template>
        </BBTooltipButton>
        <BBTooltipButton
          v-if="showEstablishBaselineButton"
          type="primary"
          :disabled="!allowMigrate"
          tooltip-mode="DISABLED-ONLY"
          data-label="bb-establish-baseline-button"
          @click="state.showBaselineModal = true"
        >
          {{ $t("change-history.establish-baseline") }}
          <template
            v-if="database.project === DEFAULT_PROJECT_V1_NAME"
            #tooltip
          >
            <div class="whitespace-pre-line">
              {{
                $t("issue.not-allowed-to-operate-unassigned-database", {
                  operation: $t(
                    "change-history.establish-baseline"
                  ).toLowerCase(),
                })
              }}
            </div>
          </template>
        </BBTooltipButton>
      </div>
    </div>
    <ChangeHistoryTable
      :mode="'DATABASE'"
      :database-section-list="[database]"
      :history-section-list="changeHistorySectionList"
      :selected-change-history-name-list="state.selectedChangeHistoryNameList"
      @update:selected="state.selectedChangeHistoryNameList = $event"
    />
  </div>

  <BBAlert
    v-if="state.showBaselineModal"
    data-label="bb-change-history-establish-baseline-alert"
    :style="'INFO'"
    :ok-text="$t('change-history.establish-baseline')"
    :cancel-text="$t('common.cancel')"
    :title="
      $t('change-history.establish-database-baseline', {
        name: database.databaseName,
      })
    "
    :description="$t('change-history.establish-baseline-description')"
    @ok="doCreateBaseline"
    @cancel="state.showBaselineModal = false"
  >
  </BBAlert>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { isEqual, orderBy, uniqBy } from "lodash-es";
import { computed, onBeforeMount, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBTooltipButton } from "@/bbkit";
import { BBTableSectionDataSource } from "@/bbkit/types";
import { ChangeHistoryTable } from "@/components/ChangeHistory";
import { useChangeHistoryStore, useDBSchemaV1Store } from "@/store";
import { ComposedDatabase, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { AffectedTable } from "@/types/changeHistory";
import {
  ChangeHistory,
  ChangeHistory_Status,
  ChangeHistory_Type,
  ChangeHistoryView,
} from "@/types/proto/v1/database_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  getAffectedTablesOfChangeHistory,
  instanceV1HasAlterSchema,
} from "@/utils";

const EmptyAffectedTable: AffectedTable = {
  schema: "",
  table: "",
  dropped: false,
};

interface LocalState {
  showBaselineModal: boolean;
  loading: boolean;
  selectedChangeHistoryNameList: string[];
  isExporting: boolean;
  selectedAffectedTable: AffectedTable;
}

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  allowEdit: {
    required: true,
    type: Boolean,
  },
});

const { t } = useI18n();

const changeHistoryStore = useChangeHistoryStore();
const router = useRouter();

const state = reactive<LocalState>({
  showBaselineModal: false,
  loading: false,
  selectedChangeHistoryNameList: [],
  isExporting: false,
  selectedAffectedTable: EmptyAffectedTable,
});

const prepareChangeHistoryList = async () => {
  state.loading = true;
  await changeHistoryStore.fetchChangeHistoryList({
    parent: props.database.name,
    pageSize: 1000,
  });
  // prepare database metadata for getting affected tables.
  await useDBSchemaV1Store().getOrFetchDatabaseMetadata(props.database.name);
  state.loading = false;
};

onBeforeMount(prepareChangeHistoryList);

const isTenantProject = computed(() => {
  return (
    props.database.projectEntity.tenantMode === TenantMode.TENANT_MODE_ENABLED
  );
});

const showEstablishBaselineButton = computed(() => {
  if (!instanceV1HasAlterSchema(props.database.instanceEntity)) {
    return false;
  }
  if (isTenantProject.value) return false;
  if (!props.allowEdit) return false;
  return true;
});

const allowExportChangeHistory = computed(() => {
  return state.selectedChangeHistoryNameList.length > 0;
});

const allowMigrate = computed(() => {
  if (!props.allowEdit) return false;

  if (props.database.projectEntity.name === DEFAULT_PROJECT_V1_NAME) {
    return false;
  }

  // Migrating single database in tenant mode is not allowed
  // Since this will probably cause different migration version across a group of tenant databases
  return !isTenantProject.value;
});

const changeHistoryList = computed(() => {
  return changeHistoryStore.changeHistoryListByDatabase(props.database.name);
});

const shownChangeHistoryList = computed(() => {
  return changeHistoryList.value.filter((changeHistory) => {
    if (
      state.selectedAffectedTable &&
      !isEqual(state.selectedAffectedTable, EmptyAffectedTable)
    ) {
      const affectedTables = getAffectedTablesOfChangeHistory(changeHistory);
      return affectedTables.find((item) =>
        isEqual(item, state.selectedAffectedTable)
      );
    }
    return true;
  });
});

const changeHistorySectionList = computed(
  (): BBTableSectionDataSource<ChangeHistory>[] => {
    return [
      {
        title: "",
        list: shownChangeHistoryList.value,
      },
    ];
  }
);

const affectedTables = computed(() => {
  return [
    EmptyAffectedTable,
    ...orderBy(
      uniqBy(
        changeHistoryList.value
          .map((changeHistory) =>
            getAffectedTablesOfChangeHistory(changeHistory)
          )
          .flat(),
        (affectedTable) => `${affectedTable.schema}.${affectedTable.table}`
      ),
      ["dropped", "table", "schema"]
    ),
  ];
});

const getAffectedTableDisplayName = (affectedTable: AffectedTable) => {
  if (isEqual(affectedTable, EmptyAffectedTable)) {
    return t("change-history.all-tables");
  }

  const { schema, table, dropped } = affectedTable;
  let name = table;
  if (schema !== "") {
    name = `${schema}.${table}`;
  }
  if (dropped) {
    name = `${name} (deleted)`;
  }
  return name;
};

const handleExportChangeHistory = async () => {
  if (state.isExporting) {
    return;
  }

  state.isExporting = true;
  const zip = new JSZip();
  for (const name of state.selectedChangeHistoryNameList) {
    const changeHistory = await changeHistoryStore.fetchChangeHistory({
      name,
      view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
    });

    if (changeHistory) {
      if (changeHistory.status !== ChangeHistory_Status.DONE) {
        continue;
      }

      if (
        changeHistory.type === ChangeHistory_Type.MIGRATE ||
        changeHistory.type === ChangeHistory_Type.MIGRATE_SDL ||
        changeHistory.type === ChangeHistory_Type.DATA
      ) {
        zip.file(`${changeHistory.version}.sql`, changeHistory.statement);
      } else if (changeHistory.type === ChangeHistory_Type.BASELINE) {
        zip.file(`${changeHistory.version}_baseline.sql`, changeHistory.schema);
      } else {
        // NOT SUPPORTED.
      }
    }
  }

  try {
    const content = await zip.generateAsync({ type: "blob" });
    const fileName = `${props.database.databaseName}_${dayjs().format(
      "YYYYMMDD"
    )}.zip`;
    saveAs(content, fileName);
  } catch (error) {
    console.error(error);
  }

  state.selectedChangeHistoryNameList = [];
  state.isExporting = false;
};

const doCreateBaseline = () => {
  state.showBaselineModal = false;

  router.push({
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.database.schema.baseline",
      name: t("change-history.establish-database-baseline", {
        name: props.database.databaseName,
      }),
      project: props.database.projectEntity.uid,
      databaseList: `${props.database.uid}`,
    },
  });
};
</script>
