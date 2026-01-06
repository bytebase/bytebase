<template>
  <div class="flex flex-col gap-y-4" v-bind="$attrs">
      <div class="flex flex-row justify-end items-center grow gap-x-2">
        <BBSpin
          v-if="state.loading"
          :size="20"
          :title="$t('changelog.refreshing')"
        />
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
          :loading="loading"
          :changelogs="list"
          :show-selection="false"
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
import { computed, reactive, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import { BBAlert, BBSpin } from "@/bbkit";
import { ChangelogDataTable } from "@/components/Changelog";
import { useDatabaseDetailContext } from "@/components/Database/context";
import { TooltipButton } from "@/components/v2";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  pushNotification,
  useChangelogStore,
  useDatabaseV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";

interface LocalState {
  showBaselineModal: boolean;
  loading: boolean;
}

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { t } = useI18n();
const changelogStore = useChangelogStore();
const databaseStore = useDatabaseV1Store();
const changedlogPagedTable =
  ref<ComponentExposed<typeof PagedTable<Changelog>>>();

const state = reactive<LocalState>({
  showBaselineModal: false,
  loading: false,
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

const allowEstablishBaseline = computed(() => {
  return allowAlterSchema.value;
});

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
