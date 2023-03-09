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
        <template v-if="database.project.id === DEFAULT_PROJECT_ID" #tooltip>
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
        name: database.name,
      })
    "
    :description="$t('change-history.establish-baseline-description')"
    @ok="doCreateBaseline"
    @cancel="state.showBaselineModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  onBeforeMount,
  PropType,
  reactive,
} from "vue";
import { useI18n } from "vue-i18n";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import {
  Database,
  DEFAULT_PROJECT_ID,
  InstanceMigration,
  MigrationHistory,
  MigrationSchemaStatus,
} from "../types";
import { useRouter } from "vue-router";
import { BBTableSectionDataSource } from "../bbkit/types";
import {
  hasWorkspacePermission,
  instanceHasAlterSchema,
  instanceSlug,
} from "../utils";
import { useCurrentUser, useInstanceStore } from "@/store";

interface LocalState {
  migrationSetupStatus: MigrationSchemaStatus;
  showBaselineModal: boolean;
  loading: boolean;
}

export default defineComponent({
  name: "DatabaseMigrationHistoryPanel",
  components: { MigrationHistoryTable },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const instanceStore = useInstanceStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      migrationSetupStatus: "OK",
      showBaselineModal: false,
      loading: false,
    });

    const currentUser = useCurrentUser();

    const prepareMigrationHistoryList = () => {
      state.loading = true;
      instanceStore
        .checkMigrationSetup(props.database.instance.id)
        .then((migration: InstanceMigration) => {
          state.migrationSetupStatus = migration.status;
          if (state.migrationSetupStatus == "OK") {
            instanceStore
              .fetchMigrationHistory({
                instanceId: props.database.instance.id,
                databaseName: props.database.name,
              })
              .then(() => {
                state.loading = false;
              })
              .catch(() => {
                state.loading = false;
              });
          }
        })
        .catch(() => {
          state.loading = false;
        });
    };

    onBeforeMount(prepareMigrationHistoryList);

    const allowConfigInstance = computed(() => {
      return hasWorkspacePermission(
        "bb.permission.workspace.manage-instance",
        currentUser.value.role
      );
    });
    const isTenantProject = computed(() => {
      return props.database.project.tenantMode === "TENANT";
    });

    const showEstablishBaselineButton = computed(() => {
      if (!instanceHasAlterSchema(props.database.instance)) return false;
      if (isTenantProject.value) return false;
      if (!props.allowEdit) return false;
      return true;
    });

    const allowMigrate = computed(() => {
      if (!props.allowEdit) return false;

      if (state.migrationSetupStatus !== "OK") return false;

      if (props.database.project.id === DEFAULT_PROJECT_ID) {
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
            name: props.database.instance.name,
          }) +
          (allowConfigInstance.value
            ? ""
            : " " + t("change-history.contact-dba"))
        );
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return (
          t("change-history.instance-bad-connection", {
            name: props.database.instance.name,
          }) +
          (allowConfigInstance.value
            ? ""
            : " " + t("change-history.contact-dba"))
        );
      }
      return "";
    });

    const migrationHistoryList = computed(() => {
      return instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
        props.database.instance.id,
        props.database.name
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
      router.push(`/instance/${instanceSlug(props.database.instance)}`);
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
            name: props.database.name,
          }),
          project: props.database.project.id,
          databaseList: `${props.database.id}`,
        },
      });
    };

    return {
      DEFAULT_PROJECT_ID,
      state,
      allowConfigInstance,
      isTenantProject,
      showEstablishBaselineButton,
      allowMigrate,
      attentionTitle,
      migrationHistorySectionList,
      configInstance,
      doCreateBaseline,
    };
  },
});
</script>
