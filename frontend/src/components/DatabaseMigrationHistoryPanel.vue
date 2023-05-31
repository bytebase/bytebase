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
    <MigrationHistoryTable
      v-if="state.migrationSetupStatus == 'OK'"
      :database-section-list="[database]"
      :history-section-list="migrationHistorySectionList"
    />
    <BBAttention
      v-else
      :style="`WARN`"
      :title="attentionTitle"
      :action-text="
        allowConfigInstance ? $t('change-history.config-instance') : ''
      "
      @click-action="configInstance"
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

import MigrationHistoryTable from "@/components/MigrationHistoryTable.vue";
import {
  ComposedDatabase,
  DEFAULT_PROJECT_V1_NAME,
  MigrationHistory,
  MigrationSchemaStatus,
} from "@/types";
import { BBTableSectionDataSource } from "@/bbkit/types";
import {
  hasWorkspacePermissionV1,
  instanceV1HasAlterSchema,
  instanceV1Slug,
} from "@/utils";
import { useCurrentUserV1, useLegacyInstanceStore } from "@/store";
import { TenantMode } from "@/types/proto/v1/project_service";

interface LocalState {
  migrationSetupStatus: MigrationSchemaStatus;
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

const instanceStore = useLegacyInstanceStore();
const router = useRouter();

const state = reactive<LocalState>({
  migrationSetupStatus: "OK",
  showBaselineModal: false,
  loading: false,
});

const currentUserV1 = useCurrentUserV1();

const prepareMigrationHistoryList = () => {
  state.loading = true;
  instanceStore
    .fetchMigrationHistory({
      instanceId: Number(props.database.instanceEntity.uid),
      databaseName: props.database.databaseName,
    })
    .then(() => {
      state.loading = false;
    })
    .catch(() => {
      state.loading = false;
    });
};

onBeforeMount(prepareMigrationHistoryList);

const allowConfigInstance = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});
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

  if (state.migrationSetupStatus !== "OK") return false;

  if (props.database.projectEntity.name === DEFAULT_PROJECT_V1_NAME) {
    return false;
  }

  // Migrating single database in tenant mode is not allowed
  // Since this will probably cause different migration version across a group of tenant databases
  return !isTenantProject.value;
});

const attentionTitle = computed((): string => {
  if (state.migrationSetupStatus == "NOT_EXIST") {
    return (
      t("change-history.instance-missing-change-schema", {
        name: props.database.instance,
      }) +
      (allowConfigInstance.value ? "" : " " + t("change-history.contact-dba"))
    );
  } else if (state.migrationSetupStatus == "UNKNOWN") {
    return (
      t("change-history.instance-bad-connection", {
        name: props.database.instance,
      }) +
      (allowConfigInstance.value ? "" : " " + t("change-history.contact-dba"))
    );
  }
  return "";
});

const migrationHistoryList = computed(() => {
  return instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
    Number(props.database.instanceEntity.uid),
    props.database.databaseName
  );
});

const migrationHistorySectionList = computed(
  (): BBTableSectionDataSource<MigrationHistory>[] => {
    return [
      {
        title: "",
        list: migrationHistoryList.value,
      },
    ];
  }
);

const configInstance = () => {
  router.push(`/instance/${instanceV1Slug(props.database.instanceEntity)}`);
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
