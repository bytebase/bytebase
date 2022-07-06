<template>
  <BBTable
    :column-list="columnList"
    :section-data-source="backupSectionList"
    :show-header="true"
    :row-clickable="false"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #header>
      <BBTableHeaderCell
        :left-padding="4"
        class="w-4"
        :title="columnList[0].title"
      />
      <BBTableHeaderCell class="w-16" :title="columnList[1].title" />
      <BBTableHeaderCell class="w-48" :title="columnList[2].title" />
      <BBTableHeaderCell class="w-16" :title="columnList[3].title" />
      <BBTableHeaderCell class="w-16" :title="columnList[4].title" />
      <BBTableHeaderCell class="w-16" :title="columnList[5].title" />
      <BBTableHeaderCell
        v-if="allowEdit"
        class="w-4"
        :title="columnList[6].title"
      />
    </template>
    <template #body="{ rowData: backup }">
      <BBTableCell :left-padding="4">
        <span
          class="flex items-center justify-center rounded-full select-none"
          :class="statusIconClass(backup)"
        >
          <template v-if="backup.status == 'PENDING_CREATE'">
            <span
              class="h-2 w-2 bg-info hover:bg-info-hover rounded-full"
              style="
                animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
              "
            >
            </span>
          </template>
          <template v-else-if="backup.status == 'DONE'">
            <heroicons-outline:check class="w-4 h-4" />
          </template>
          <template v-else-if="backup.status == 'FAILED'">
            <span
              class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
              aria-hidden="true"
              >!</span
            >
          </template>
        </span>
      </BBTableCell>
      <BBTableCell>
        {{ backup.name }}
      </BBTableCell>
      <BBTableCell class="tooltip-wrapper">
        <span v-if="backup.comment.length > 100" class="tooltip">{{
          backup.comment
        }}</span>
        {{
          backup.comment.length > 100
            ? backup.comment.substring(0, 100) + "..."
            : backup.comment
        }}
      </BBTableCell>
      <BBTableCell>
        <div class="flex flex-row space-x-2">
          <div
            class="normal-link"
            @click.prevent="gotoMigrationHistory(backup)"
          >
            {{ backup.migrationHistoryVersion }}
          </div>
          <BBSpin v-if="state.loadingMigrationHistory" />
        </div>
      </BBTableCell>
      <BBTableCell>
        {{ humanizeTs(backup.createdTs) }}
      </BBTableCell>
      <BBTableCell>
        {{ backup.creator.name }}
      </BBTableCell>
      <BBTableCell v-if="allowEdit">
        <button
          :disabled="!allowRestore(backup)"
          class="normal-link disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:no-underline"
          @click.stop="
            () => {
              state.restoredBackup = backup;
              state.showRestoreBackupModal = true;
            }
          "
        >
          {{ $t("database.restore") }}
        </button>
      </BBTableCell>
    </template>
  </BBTable>
  <BBModal
    v-if="state.showRestoreBackupModal"
    :title="$t('database.restore-backup', [state.restoredBackup.name])"
    @close="
      () => {
        state.showRestoreBackupModal = false;
        state.restoredBackup = undefined;
      }
    "
  >
    <!-- eslint-disable vue/attribute-hyphenation -->
    <CreateDatabasePrepForm
      :projectId="database.project.id"
      :environmentId="database.instance.environment.id"
      :instanceId="database.instance.id"
      :backup="state.restoredBackup"
      @dismiss="
        () => {
          state.showRestoreBackupModal = false;
          state.restoredBackup = undefined;
        }
      "
    />
  </BBModal>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import { Backup, Database, MigrationHistory } from "../types";
import { bytesToString, databaseSlug, migrationHistorySlug } from "../utils";
import CreateDatabasePrepForm from "../components/CreateDatabasePrepForm.vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useInstanceStore } from "@/store";

interface LocalState {
  showRestoreBackupModal: boolean;
  restoredBackup?: Backup;
  loadingMigrationHistory: boolean;
}

export default defineComponent({
  name: "BackupTable",
  components: { CreateDatabasePrepForm },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    backupList: {
      required: true,
      type: Object as PropType<Backup[]>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  setup(props) {
    const router = useRouter();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      showRestoreBackupModal: false,
      loadingMigrationHistory: false,
    });

    const EDIT_COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.status"),
      },
      {
        title: t("common.name"),
      },
      {
        title: t("common.comment"),
      },
      {
        title: t("common.schema-version"),
      },
      {
        title: t("common.time"),
      },
      {
        title: t("common.creator"),
      },
      {
        title: "",
      },
    ];

    const NON_EDIT_COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.status"),
      },
      {
        title: t("common.name"),
      },
      {
        title: t("common.comment"),
      },
      {
        title: t("common.schema-version"),
      },
      {
        title: t("common.time"),
      },
      {
        title: t("common.creator"),
      },
    ];

    const backupSectionList = computed(() => {
      const manualList: Backup[] = [];
      const automaticList: Backup[] = [];
      const sectionList: BBTableSectionDataSource<Backup>[] = [
        {
          title: t("common.manual"),
          list: manualList,
        },
        {
          title: t("common.automatic"),
          list: automaticList,
        },
      ];

      for (const backup of props.backupList) {
        if (backup.type == "MANUAL") {
          manualList.push(backup);
        } else if (backup.type == "AUTOMATIC") {
          automaticList.push(backup);
        }
      }

      return sectionList;
    });

    const statusIconClass = (backup: Backup) => {
      let iconClass = "w-5 h-5";
      switch (backup.status) {
        case "PENDING_CREATE":
          return (
            iconClass +
            " bg-white border-2 border-info text-info hover:text-info-hover hover:border-info-hover"
          );
        case "DONE":
          return iconClass + " bg-success hover:bg-success-hover text-white";
        case "FAILED":
          return (
            iconClass +
            " bg-error text-white hover:text-white hover:bg-error-hover"
          );
      }
    };

    const gotoMigrationHistory = (backup: Backup) => {
      state.loadingMigrationHistory = true;
      useInstanceStore()
        .fetchMigrationHistoryByVersion({
          instanceId: props.database.instance.id,
          databaseName: props.database.name,
          version: backup.migrationHistoryVersion,
        })
        .then((history: MigrationHistory) => {
          router
            .push({
              name: "workspace.database.history.detail",
              params: {
                databaseSlug: databaseSlug(props.database),
                migrationHistorySlug: migrationHistorySlug(
                  history.id,
                  history.version
                ),
              },
              hash: "#schema",
            })
            .then(() => {
              state.loadingMigrationHistory = false;
            });
        })
        .catch(() => {
          state.loadingMigrationHistory = false;
        });
    };

    const columnList = computed(() => {
      return props.allowEdit ? EDIT_COLUMN_LIST : NON_EDIT_COLUMN_LIST;
    });

    const allowRestore = (backup: Backup) => {
      return backup.status === "DONE";
    };

    return {
      state,
      gotoMigrationHistory,
      columnList,
      bytesToString,
      backupSectionList,
      statusIconClass,
      allowRestore,
    };
  },
});
</script>
