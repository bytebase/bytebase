<template>
  <div class="flex flex-col space-y-4" v-bind="$attrs">
    <div class="w-full flex flex-row justify-between items-center space-x-2">
      <div class="flex flex-row justify-start items-center space-x-4">
        <div class="w-52">
          <AffectedTablesSelect
            v-model:tables="state.selectedAffectedTables"
            :database="database"
          />
        </div>
        <div class="w-40">
          <ChangeTypeSelect
            v-model:change-type="state.selectedChangeType"
            :changelogs="changelogs"
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
          :disabled="!allowExportChangelog"
          :loading="state.isExporting"
          @click="handleExportChangelogs"
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
          :disabled="
            !selectedChangelogForRollback ||
            getChangelogChangeType(selectedChangelogForRollback.type) !== 'DDL'
          "
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

    <PagedChangelogTable
      :database="database"
      :search-changelogs="{
        tables: state.selectedAffectedTables,
        types: state.selectedChangeType
          ? [state.selectedChangeType]
          : undefined,
      }"
      session-key="bb.paged-changelog-table"
    >
      <template #table="{ list, loading }">
        <ChangelogDataTable
          :key="`changelog-table.${database.name}`"
          v-model:selected-changelogs="state.selectedChangelogNames"
          :loading="loading"
          :changelogs="list"
          :show-selection="true"
        />
      </template>
    </PagedChangelogTable>
  </div>

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
  AffectedTablesSelect,
  ChangeTypeSelect,
  ChangelogDataTable,
  PagedChangelogTable,
} from "@/components/Changelog";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { TooltipButton } from "@/components/v2";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_SYNC_SCHEMA,
} from "@/router/dashboard/projectV1";
import { useChangelogStore, useDBSchemaV1Store } from "@/store";
import { DEFAULT_PAGE_SIZE } from "@/store/modules/common";
import type { ComposedDatabase, Table } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import {
  Changelog_Status,
  Changelog_Type,
  ChangelogView,
} from "@/types/proto/v1/database_service";
import { extractProjectResourceName } from "@/utils";
import { getChangelogChangeType } from "@/utils/v1/changelog";

interface LocalState {
  showBaselineModal: boolean;
  loading: boolean;
  selectedChangelogNames: string[];
  isExporting: boolean;
  selectedAffectedTables: Table[];
  selectedChangeType?: Changelog_Type;
}

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { t } = useI18n();
const router = useRouter();
const changelogStore = useChangelogStore();

const state = reactive<LocalState>({
  showBaselineModal: false,
  loading: false,
  selectedChangelogNames: [],
  isExporting: false,
  selectedAffectedTables: [],
});

const { allowAlterSchema } = useDatabaseDetailContext();

const prepareChangelogList = async () => {
  state.loading = true;
  await changelogStore.fetchChangelogList({
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

onBeforeMount(prepareChangelogList);

const allowExportChangelog = computed(() => {
  return state.selectedChangelogNames.length > 0;
});

const allowEstablishBaseline = computed(() => {
  return allowAlterSchema.value;
});

const changelogs = computed(() => {
  return changelogStore.changelogListByDatabase(props.database.name);
});

const selectedChangelogForRollback = computed(() => {
  if (state.selectedChangelogNames.length !== 1) {
    return;
  }
  return changelogStore.getChangelogByName(state.selectedChangelogNames[0]);
});

const rollback = () => {
  if (!selectedChangelogForRollback.value) {
    return;
  }

  router.push({
    name: PROJECT_V1_ROUTE_SYNC_SCHEMA,
    params: {
      projectId: extractProjectResourceName(props.database.project),
    },
    query: {
      version: selectedChangelogForRollback.value.name,
      target: props.database.name,
    },
  });
};

const handleExportChangelogs = async () => {
  if (state.isExporting) {
    return;
  }

  state.isExporting = true;
  const zip = new JSZip();
  for (const name of state.selectedChangelogNames) {
    const changelog = await changelogStore.fetchChangelog({
      name,
      view: ChangelogView.CHANGELOG_VIEW_FULL,
    });

    if (changelog) {
      if (changelog.status !== Changelog_Status.DONE) {
        continue;
      }

      if (
        changelog.type === Changelog_Type.MIGRATE ||
        changelog.type === Changelog_Type.MIGRATE_SDL ||
        changelog.type === Changelog_Type.DATA
      ) {
        zip.file(`${changelog.version}.sql`, changelog.statement);
      } else if (changelog.type === Changelog_Type.BASELINE) {
        zip.file(`${changelog.version}_baseline.sql`, changelog.schema);
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

  state.selectedChangelogNames = [];
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
