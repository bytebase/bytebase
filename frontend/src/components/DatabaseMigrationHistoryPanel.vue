<template>
  <div class="flex flex-col space-y-4">
    <div
      class="flex flex-row items-center text-lg leading-6 font-medium text-main space-x-2"
    >
      <span>{{ $t("migration-history.self") }}</span>
      <BBTooltipButton
        v-if="allowEdit"
        type="primary"
        :disabled="!allowMigrate"
        tooltip-mode="DISABLED-ONLY"
        @click="state.showBaselineModal = true"
      >
        {{ $t("migration-history.establish-baseline") }}

        <template v-if="isTenantProject" #tooltip>
          <div class="w-52 whitespace-pre-wrap">
            {{
              $t("issue.not-allowed-to-single-database-in-tenant-mode", {
                operation: $t(
                  "migration-history.establish-baseline"
                ).toLowerCase(),
              })
            }}
          </div>
        </template>
      </BBTooltipButton>
      <div>
        <BBSpin
          v-if="state.loading"
          :title="$t('migration-history.refreshing-history')"
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
        allowConfigInstance ? $t('migration-history.config-instance') : ''
      "
      @click-action="configInstance"
    />
  </div>

  <BBAlert
    v-if="state.showBaselineModal"
    :style="'INFO'"
    :ok-text="$t('migration-history.establish-baseline')"
    :cancel-text="$t('common.cancel')"
    :title="
      $t('migration-history.establish-database-baseline', {
        name: database.name,
      })
    "
    :description="$t('migration-history.establish-baseline-description')"
    @ok="
      () => {
        doCreateBaseline();
      }
    "
    @cancel="state.showBaselineModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  PropType,
  reactive,
  watchEffect,
} from "vue";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import {
  Database,
  InstanceMigration,
  MigrationHistory,
  MigrationSchemaStatus,
} from "../types";
import { useRouter } from "vue-router";
import { BBTableSectionDataSource } from "../bbkit/types";
import { instanceSlug, isDBAOrOwner } from "../utils";
import { useI18n } from "vue-i18n";
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

    watchEffect(prepareMigrationHistoryList);

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowConfigInstance = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const isTenantProject = computed(() => {
      return props.database.project.tenantMode === "TENANT";
    });

    const allowMigrate = computed(() => {
      if (!props.allowEdit) return false;

      if (state.migrationSetupStatus !== "OK") return false;

      // Migrating single database in tenant mode is not allowed
      // Since this will probably cause different migration version across a group of tenant databases
      return !isTenantProject.value;
    });

    const attentionTitle = computed((): string => {
      if (state.migrationSetupStatus == "NOT_EXIST") {
        return (
          t("migration-history.instance-missing-migration-schema", {
            name: props.database.instance.name,
          }) +
          (isDBAOrOwner(currentUser.value.role)
            ? ""
            : " " + t("migration-history.contact-dba"))
        );
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return (
          t("migration-history.instance-bad-connection", {
            name: props.database.instance.name,
          }) +
          (isDBAOrOwner(currentUser.value.role)
            ? ""
            : " " + t("migration-history.contact-dba"))
        );
      }
      return "";
    });

    const migrationHistorySectionList = computed(
      (): BBTableSectionDataSource<MigrationHistory>[] => {
        return [
          {
            title: "",
            list: instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
              props.database.instance.id,
              props.database.name
            ),
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
          name: t("migration-history.establish-database-baseline", {
            name: props.database.name,
          }),
          project: props.database.project.id,
          databaseList: `${props.database.id}`,
        },
      });
    };

    return {
      state,
      allowConfigInstance,
      isTenantProject,
      allowMigrate,
      attentionTitle,
      migrationHistorySectionList,
      configInstance,
      doCreateBaseline,
    };
  },
});
</script>
