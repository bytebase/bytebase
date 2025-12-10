<template>
  <div class="flex flex-col gap-y-4" v-bind="$attrs">
      <div class="flex flex-row justify-end items-center grow gap-x-2">
        <BBSpin
          v-if="state.loading"
          :size="20"
          :title="$t('changelog.refreshing')"
        />
        <TooltipButton
          tooltip-mode="DISABLED-ONLY"
          :disabled="!allowExportChangelog"
          :loading="state.isExporting"
          @click="handleExportChangelogs"
        >
          <template #default>
            {{ $t("changelog.export") }}
          </template>
          <template #tooltip>
            <div class="whitespace-pre-line">
              {{ $t("changelog.need-to-select-first") }}
            </div>
          </template>
        </TooltipButton>
        <TooltipButton
          v-if="allowAlterSchema"
          tooltip-mode="DISABLED-ONLY"
          :disabled="
            !selectedChangelogForRollback ||
            selectedChangelogForRollback.type !== Changelog_Type.MIGRATE
          "
          @click="rollback"
        >
          <template #default>
            {{ $t("common.rollback") }}
          </template>
          <template #tooltip>
            <div class="whitespace-pre-line">
              {{ $t("changelog.rollback-tip") }}
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
            {{ $t("changelog.establish-baseline") }}
          </template>
          <template v-if="database.project === DEFAULT_PROJECT_NAME" #tooltip>
            <div class="whitespace-pre-line">
              {{
                $t("issue.not-allowed-to-operate-unassigned-database", {
                  operation: $t("changelog.establish-baseline").toLowerCase(),
                })
              }}
            </div>
          </template>
        </TooltipButton>
      </div>


    <PagedTable
      ref="changedlogPagedTable"
      :session-key="`bb.paged-changelog-table.${database.name}`"
      :fetch-list="fetchChangelogList"
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
    </PagedTable>
  </div>

  <BBAlert
    v-model:show="state.showBaselineModal"
    data-label="bb-changelog-establish-baseline-alert"
    type="info"
    :ok-text="$t('changelog.establish-baseline')"
    :cancel-text="$t('common.cancel')"
    :title="
      $t('changelog.establish-database-baseline', {
        name: database.databaseName,
      })
    "
    :description="$t('changelog.establish-baseline-description')"
    @ok="updateDatabaseDrift"
    @cancel="state.showBaselineModal = false"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { computed, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBAlert, BBSpin } from "@/bbkit";
import {
  ChangelogDataTable,
} from "@/components/Changelog";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { TooltipButton } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { PROJECT_V1_ROUTE_SYNC_SCHEMA } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useChangelogStore,
  useDatabaseV1Store,
} from "@/store";
import type { ComposedDatabase, SearchChangeLogParams, Table } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  Changelog_Type,
  ChangelogView,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  showBaselineModal: boolean;
  loading: boolean;
  selectedChangelogNames: string[];
  isExporting: boolean;

}

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { t } = useI18n();
const router = useRouter();
const changelogStore = useChangelogStore();
const databaseStore = useDatabaseV1Store();
const changedlogPagedTable =
  ref<ComponentExposed<typeof PagedTable<Changelog>>>();

const state = reactive<LocalState>({
  showBaselineModal: false,
  loading: false,
  selectedChangelogNames: [],
  isExporting: false,

});



const searchChangelogFilter = computed(() => {
  return "";
});

const fetchChangelogList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, changelogs } = await changelogStore.fetchChangelogList(
    {
      parent: props.database.name,
      filter: searchChangelogFilter.value,
      pageSize,
      pageToken,
    }
  );
  return {
    nextPageToken,
    list: changelogs,
  };
};



const { allowAlterSchema } = useDatabaseDetailContext();

const allowExportChangelog = computed(() => {
  return state.selectedChangelogNames.length > 0;
});

const allowEstablishBaseline = computed(() => {
  return allowAlterSchema.value;
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
      changelog: selectedChangelogForRollback.value.name,
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
      view: ChangelogView.FULL,
    });

    if (changelog) {
      if (changelog.status !== Changelog_Status.DONE) {
        continue;
      }

      const filePathPrefix = dayjs(
        changelog.createTime
          ? new Date(Number(changelog.createTime.seconds) * 1000)
          : new Date()
      ).format("YYYY-MM-DDTHH-mm-ss");
      if (
        changelog.type === Changelog_Type.MIGRATE ||
        changelog.type === Changelog_Type.SDL
      ) {
        zip.file(`${filePathPrefix}.sql`, changelog.statement);
      } else if (changelog.type === Changelog_Type.BASELINE) {
        zip.file(`${filePathPrefix}_baseline.sql`, changelog.schema);
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

const updateDatabaseDrift = async () => {
  const updatedDatabase = create(DatabaseSchema$, {
    ...props.database,
    drifted: false,
  });

  await databaseStore.updateDatabase(
    create(UpdateDatabaseRequestSchema, {
      database: updatedDatabase,
      updateMask: create(FieldMaskSchema, { paths: ["drifted"] }),
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("database.drifted.new-baseline.successfully-established"),
  });
  state.showBaselineModal = false;
  changedlogPagedTable.value?.refresh();
};
</script>
