<template>
  <div class="flex flex-col space-y-4">
    <div
      class="flex flex-row items-center text-lg leading-6 font-medium text-main space-x-2"
    >
      <span>{{ $t("change-history.self") }}</span>
      <BBTooltipButton
        v-if="showEstablishBaselineButton"
        type="primary"
        :disabled="!allowMigrate"
        tooltip-mode="DISABLED-ONLY"
        data-label="bb-establish-baseline-button"
        @click="state.showBaselineModal = true"
      >
        {{ $t("change-history.establish-baseline") }}
        <template v-if="database.project === DEFAULT_PROJECT_V1_NAME" #tooltip>
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
      <div>
        <BBSpin
          v-if="state.loading"
          :title="$t('change-history.refreshing-history')"
        />
      </div>
    </div>
    <ChangeHistoryTable
      :database-section-list="[database]"
      :history-section-list="changeHistorySectionList"
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
import { computed, onBeforeMount, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";

import { ComposedDatabase, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { BBTableSectionDataSource } from "@/bbkit/types";
import { instanceV1HasAlterSchema } from "@/utils";
import { useChangeHistoryStore } from "@/store";
import { TenantMode } from "@/types/proto/v1/project_service";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { ChangeHistoryTable } from "@/components/ChangeHistory";

interface LocalState {
  showBaselineModal: boolean;
  loading: boolean;
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
});

const prepareChangeHistoryList = () => {
  state.loading = true;
  changeHistoryStore
    .fetchChangeHistoryList({
      parent: props.database.name,
    })
    .finally(() => {
      state.loading = false;
    });
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

const changeHistorySectionList = computed(
  (): BBTableSectionDataSource<ChangeHistory>[] => {
    return [
      {
        title: "",
        list: changeHistoryList.value,
      },
    ];
  }
);

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
