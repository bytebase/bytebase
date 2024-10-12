<template>
  <div class="flex flex-col space-y-4" v-bind="$attrs">
    <div
      class="w-full flex flex-row justify-between items-center text-lg leading-6 font-medium text-main space-x-2"
    >
      <div class="flex flex-row justify-start items-center space-x-4">
        <div class="w-56">
          <AffectedTablesSelect
            v-model:tables="state.selectedAffectedTables"
            :database="database"
          />
        </div>
        <div class="w-44">
          <ChangeTypeSelect
            v-model:change-type="state.selectedChangeType"
            :change-history-list="changeHistoryList"
          />
        </div>
      </div>
      <div class="flex flex-row justify-end items-center grow space-x-2">
        <BBSpin
          v-if="state.loading"
          :size="20"
          :title="$t('change-history.refreshing-history')"
        />
        <TooltipButton
          tooltip-mode="DISABLED-ONLY"
          :disabled="!allowExportChangeHistory"
          :loading="state.isExporting"
          @click="handleExportChangeHistory"
        >
          <template #default>
            {{ $t("change-history.export") }}
          </template>
          <template #tooltip>
            <div class="whitespace-pre-line">
              {{ $t("change-history.need-to-select-first") }}
            </div>
          </template>
        </TooltipButton>
        <TooltipButton
          v-if="allowAlterSchema"
          tooltip-mode="DISABLED-ONLY"
          :disabled="state.selectedChangeHistoryNameList.length !== 1"
          @click="rollback"
        >
          <template #default>
            {{ $t("common.rollback") }}
          </template>
          <template #tooltip>
            <div class="whitespace-pre-line">
              {{ $t("change-history.rollback-tip") }}
            </div>
          </template>
        </TooltipButton>
        <TooltipButton
          v-if="allowEstablishBaseline"
          tooltip-mode="DISABLED-ONLY"
          :disabled="false"
          type="primary"
          @click="state.showBaselineModal = true"
        >
          <template #default>
            {{ $t("change-history.establish-baseline") }}
          </template>
          <template v-if="database.project === DEFAULT_PROJECT_NAME" #tooltip>
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
        </TooltipButton>
      </div>
    </div>

    <PagedChangeHistoryTable
      :database="database"
      :search-change-histories="{
        tables: state.selectedAffectedTables,
        types: state.selectedChangeType
          ? [state.selectedChangeType]
          : undefined,
      }"
      session-key="bb.paged-change-history-table"
    >
      <template #table="{ list }">
        <ChangeHistoryDataTable
          :key="`change-history-table.${database.name}`"
          v-model:selected-change-history-names="
            state.selectedChangeHistoryNameList
          "
          :change-histories="list"
          :custom-click="true"
          :show-selection="true"
          @row-click="(id: string) => (state.selectedChangeHistoryId = id)"
        />
      </template>
    </PagedChangeHistoryTable>
  </div>

  <Drawer
    :show="!!state.selectedChangeHistoryId"
    @close="state.selectedChangeHistoryId = ''"
  >
    <DrawerContent
      class="w-[80vw] max-w-[100vw] relative"
      :title="$t('change-history.self')"
    >
      <ChangeHistoryDetail
        :instance="database.instance"
        :database="database.name"
        :change-history-id="state.selectedChangeHistoryId"
      />
    </DrawerContent>
  </Drawer>

  <BBAlert
    v-model:show="state.showBaselineModal"
    data-label="bb-change-history-establish-baseline-alert"
    type="info"
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
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { computed, onBeforeMount, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAlert, BBSpin } from "@/bbkit";
import {
  ChangeHistoryDataTable,
  ChangeHistoryDetail,
  PagedChangeHistoryTable,
  ChangeTypeSelect,
  AffectedTablesSelect,
} from "@/components/ChangeHistory";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { TooltipButton } from "@/components/v2";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
} from "@/router/dashboard/projectV1";
import { useChangeHistoryStore, useDBSchemaV1Store } from "@/store";
import { DEFAULT_PAGE_SIZE } from "@/store/modules/common";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Table } from "@/types/changeHistory";
import {
  ChangeHistory_Status,
  ChangeHistory_Type,
  ChangeHistoryView,
} from "@/types/proto/v1/database_service";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  showBaselineModal: boolean;
  loading: boolean;
  selectedChangeHistoryNameList: string[];
  isExporting: boolean;
  selectedAffectedTables: Table[];
  selectedChangeType?: string;
  selectedChangeHistoryId: string;
}

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { t } = useI18n();

const changeHistoryStore = useChangeHistoryStore();
const router = useRouter();

const state = reactive<LocalState>({
  showBaselineModal: false,
  loading: false,
  selectedChangeHistoryNameList: [],
  isExporting: false,
  selectedAffectedTables: [],
  selectedChangeHistoryId: "",
});

const { allowAlterSchema } = useDatabaseDetailContext();

const prepareChangeHistoryList = async () => {
  state.loading = true;
  await changeHistoryStore.fetchChangeHistoryList({
    parent: props.database.name,
    pageSize: DEFAULT_PAGE_SIZE,
  });
  // prepare database metadata for getting affected tables.
  await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
    database: props.database.name,
    skipCache: true, // Skip cache to get the latest metadata.
  });
  state.loading = false;
};

onBeforeMount(prepareChangeHistoryList);

const allowExportChangeHistory = computed(() => {
  return state.selectedChangeHistoryNameList.length > 0;
});

const allowEstablishBaseline = computed(() => {
  return allowAlterSchema.value;
});

const changeHistoryList = computed(() => {
  return changeHistoryStore.changeHistoryListByDatabase(props.database.name);
});

const rollback = () => {
  if (state.selectedChangeHistoryNameList.length !== 1) {
    return;
  }
  const changeHistory = state.selectedChangeHistoryNameList[0];

  router.push({
    name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
    params: {
      projectId: extractProjectResourceName(props.database.project),
    },
    query: {
      version: changeHistory,
      target: props.database.name,
    },
  });
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
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(props.database.project),
      issueSlug: "create",
    },
    query: {
      template: "bb.issue.database.schema.baseline",
      name: t("change-history.establish-database-baseline", {
        name: props.database.databaseName,
      }),
      databaseList: props.database.name,
    },
  });
};
</script>
