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
      <BBTableHeaderCell
        v-if="allowEdit"
        class="w-4"
        :title="columnList[5].title"
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
      <BBTableCell class="tooltip-wrapper">
        <span class="tooltip whitespace-nowrap">
          {{ dayjs(backup.createdTs * 1000).format("YYYY-MM-DD HH:mm") }}
        </span>
        {{ humanizeTs(backup.createdTs) }}
      </BBTableCell>
      <BBTableCell v-if="allowEdit">
        <button
          :disabled="!allowRestore(backup)"
          class="normal-link disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:no-underline"
          @click.stop="showRestoreDialog(backup)"
        >
          {{ $t("database.restore") }}
        </button>
      </BBTableCell>
    </template>
  </BBTable>
  <BBModal
    v-if="state.restoreBackupContext"
    :title="$t('database.restore-database')"
    @close="state.restoreBackupContext = undefined"
  >
    <div class="space-y-4 w-[35rem]">
      <RestoreTargetForm
        v-if="allowRestoreInPlace"
        :target="state.restoreBackupContext.target"
        @change="state.restoreBackupContext!.target = $event"
      />

      <CreateDatabasePrepForm
        v-if="state.restoreBackupContext.target === 'NEW'"
        :project-id="database.projectEntity.uid"
        :environment-id="database.instanceEntity.environmentEntity.uid"
        :instance-id="database.instanceEntity.uid"
        :backup="state.restoreBackupContext.backup"
        @dismiss="state.restoreBackupContext = undefined"
      />
    </div>

    <div
      v-if="state.restoreBackupContext.target === 'IN-PLACE'"
      class="w-full pt-6 mt-4 flex justify-end gap-x-3 border-t border-block-border"
    >
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click="state.restoreBackupContext = undefined"
      >
        {{ $t("common.cancel") }}
      </button>

      <button
        type="button"
        class="btn-primary py-2 px-4"
        @click="doRestoreInPlace"
      >
        {{ $t("common.confirm") }}
      </button>
    </div>

    <div
      v-if="state.creatingRestoreIssue"
      class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
    >
      <BBSpin />
    </div>
  </BBModal>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.pitr"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { BBTableColumn, BBTableSectionDataSource } from "../bbkit/types";
import {
  Backup,
  ComposedDatabase,
  IssueCreate,
  MigrationHistory,
  PITRContext,
  SYSTEM_BOT_ID,
} from "../types";
import {
  changeHistorySlug,
  extractDatabaseResourceName,
  extractInstanceResourceName,
  issueSlug,
} from "../utils";
import {
  featureToRef,
  useLegacyInstanceStore,
  useIssueStore,
  pushNotification,
} from "@/store";
import CreateDatabasePrepForm from "../components/CreateDatabasePrepForm.vue";
import {
  default as RestoreTargetForm,
  RestoreTarget,
} from "../components/DatabaseBackup/RestoreTargetForm.vue";
import { Engine } from "@/types/proto/v1/common";

type RestoreBackupContext = {
  target: RestoreTarget;
  backup: Backup;
};

interface LocalState {
  restoreBackupContext?: RestoreBackupContext;
  loadingMigrationHistory: boolean;
  creatingRestoreIssue: boolean;
  showFeatureModal: boolean;
}

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  backupList: {
    required: true,
    type: Object as PropType<Backup[]>,
  },
  allowEdit: {
    required: true,
    type: Boolean,
  },
});

const router = useRouter();
const { t } = useI18n();

const state = reactive<LocalState>({
  restoreBackupContext: undefined,
  loadingMigrationHistory: false,
  creatingRestoreIssue: false,
  showFeatureModal: false,
});

const allowRestoreInPlace = computed((): boolean => {
  return props.database.instanceEntity.engine === Engine.POSTGRES;
});

const hasPITRFeature = featureToRef("bb.feature.pitr");

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
];

const backupSectionList = computed(() => {
  const manualList: Backup[] = [];
  const automaticList: Backup[] = [];
  const pitrList: Backup[] = [];
  const sectionList: BBTableSectionDataSource<Backup>[] = [
    {
      title: t("common.manual"),
      list: manualList,
    },
    {
      title: t("common.automatic"),
      list: automaticList,
    },
    {
      title: t("common.pitr"),
      list: pitrList,
    },
  ];

  for (const backup of props.backupList) {
    if (backup.type == "MANUAL") {
      manualList.push(backup);
    } else if (backup.type == "AUTOMATIC") {
      automaticList.push(backup);
    } else if (backup.type === "PITR") {
      pitrList.push(backup);
    }
  }

  return sectionList;
});

const statusIconClass = (backup: Backup) => {
  const iconClass = "w-5 h-5";
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
        iconClass + " bg-error text-white hover:text-white hover:bg-error-hover"
      );
  }
};

const gotoMigrationHistory = (backup: Backup) => {
  state.loadingMigrationHistory = true;
  useLegacyInstanceStore()
    .fetchMigrationHistoryByVersion({
      instanceId: Number(props.database.instanceEntity.uid),
      databaseName: props.database.databaseName,
      version: backup.migrationHistoryVersion,
    })
    .then((history: MigrationHistory) => {
      router
        .push({
          name: "workspace.database.history.detail",
          params: {
            instance: extractInstanceResourceName(props.database.instance),
            database: extractDatabaseResourceName(props.database.name).database,
            changeHistorySlug: changeHistorySlug(history.id, history.version),
          },
          hash: "#schema",
        })
        .then(() => {
          state.loadingMigrationHistory = false;
        });
    })
    .catch((error) => {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: error.message,
      });
      state.loadingMigrationHistory = false;
    });
};

const columnList = computed(() => {
  return props.allowEdit ? EDIT_COLUMN_LIST : NON_EDIT_COLUMN_LIST;
});

const allowRestore = (backup: Backup) => {
  return backup.status === "DONE";
};

const showRestoreDialog = (backup: Backup) => {
  state.restoreBackupContext = {
    target: "NEW",
    backup,
  };
};

const doRestoreInPlace = async () => {
  const { restoreBackupContext } = state;
  if (!restoreBackupContext) {
    return;
  }

  if (!hasPITRFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  state.creatingRestoreIssue = true;

  try {
    const { backup } = restoreBackupContext;
    const { database } = props;
    const issueNameParts: string[] = [
      `Restore database [${database.name}]`,
      `to backup snapshot [${restoreBackupContext.backup.name}]`,
    ];

    const issueStore = useIssueStore();
    const createContext: PITRContext = {
      databaseId: Number(database.uid),
      backupId: backup.id,
    };
    const issueCreate: IssueCreate = {
      name: issueNameParts.join(" "),
      type: "bb.issue.database.restore.pitr",
      description: "",
      assigneeId: SYSTEM_BOT_ID,
      projectId: Number(database.projectEntity.uid),
      payload: {},
      createContext,
    };

    await issueStore.validateIssue(issueCreate);

    const issue = await issueStore.createIssue(issueCreate);

    const slug = issueSlug(issue.name, issue.id);
    router.push(`/issue/${slug}`);
  } catch {
    state.creatingRestoreIssue = false;
  }
};
</script>
