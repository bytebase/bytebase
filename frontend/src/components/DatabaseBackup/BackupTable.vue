<template>
  <div>
    <div
      v-for="section in backupSectionList"
      :key="section.title"
      class="border-x border-b first:border-t"
    >
      <div class="py-2 px-2">{{ section.title }}</div>

      <BBGrid
        :column-list="columnList"
        :data-source="section.list"
        class="border-t"
        :row-clickable="false"
        :show-placeholder="true"
      >
        <template #item="{ item: backup }: BackupRow">
          <div class="bb-grid-cell">
            <span
              class="flex items-center justify-center rounded-full select-none"
              :class="statusIconClass(backup)"
            >
              <template
                v-if="backup.state === Backup_BackupState.PENDING_CREATE"
              >
                <span
                  class="h-2 w-2 bg-info hover:bg-info-hover rounded-full"
                  style="
                    animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
                  "
                >
                </span>
              </template>
              <template v-else-if="backup.state === Backup_BackupState.DONE">
                <heroicons-outline:check class="w-4 h-4" />
              </template>
              <template v-else-if="backup.state === Backup_BackupState.FAILED">
                <span
                  class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
                  aria-hidden="true"
                  >!</span
                >
              </template>
            </span>
          </div>
          <div class="bb-grid-cell">
            {{ extractBackupResourceName(backup.name) }}
          </div>
          <div class="bb-grid-cell">
            <EllipsisText>
              {{ backup.comment }}
            </EllipsisText>
          </div>
          <div class="bb-grid-cell">
            <HumanizeDate :date="backup.createTime" />
          </div>
          <div v-if="allowEdit" class="bb-grid-cell">
            <NButton
              :disabled="!allowRestore(backup)"
              @click.stop="showRestoreDialog(backup)"
            >
              {{ $t("database.restore") }}
            </NButton>
          </div>
        </template>
      </BBGrid>
    </div>

    <Drawer
      :show="state.restoreBackupContext !== undefined"
      @close="state.restoreBackupContext = undefined"
    >
      <DrawerContent
        v-if="state.restoreBackupContext"
        :title="$t('database.restore-database')"
      >
        <div class="w-72">
          <div v-if="allowRestoreInPlace" class="space-y-4">
            <RestoreTargetForm
              :target="state.restoreBackupContext.target"
              first="NEW"
              @change="state.restoreBackupContext!.target = $event"
            />
          </div>

          <div class="mt-2">
            <CreateDatabasePrepForm
              v-if="state.restoreBackupContext.target === 'NEW'"
              ref="createDatabasePrepForm"
              :project-id="database.projectEntity.uid"
              :environment-id="database.instanceEntity.environmentEntity.uid"
              :instance-id="database.instanceEntity.uid"
              :backup="state.restoreBackupContext.backup"
              @dismiss="state.restoreBackupContext = undefined"
            />
          </div>
          <div
            v-if="state.creatingRestoreIssue"
            class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center"
          >
            <BBSpin />
          </div>
        </div>

        <template #footer>
          <div v-if="state.restoreBackupContext.target === 'NEW'">
            <CreateDatabasePrepButtonGroup :form="createDatabasePrepForm" />
          </div>

          <div
            v-if="state.restoreBackupContext.target === 'IN-PLACE'"
            class="w-full flex justify-end gap-x-3"
          >
            <NButton @click="state.restoreBackupContext = undefined">
              {{ $t("common.cancel") }}
            </NButton>

            <NButton type="primary" @click="doRestoreInPlace">
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </template>
      </DrawerContent>
    </Drawer>
    <BBModal
      v-if="false && state.restoreBackupContext"
      :title="$t('database.restore-database')"
      @close="state.restoreBackupContext = undefined"
    >
    </BBModal>

    <FeatureModal
      feature="bb.feature.pitr"
      :open="state.showFeatureModal"
      :instance="database.instanceEntity"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, PropType, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGrid, BBGridColumn, BBGridRow } from "@/bbkit";
import {
  CreateDatabasePrepForm,
  CreateDatabasePrepButtonGroup,
} from "@/components/CreateDatabasePrepForm";
import {
  default as RestoreTargetForm,
  RestoreTarget,
} from "@/components/DatabaseBackup/RestoreTargetForm.vue";
import EllipsisText from "@/components/EllipsisText.vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { experimentalCreateIssueByPlan, useSubscriptionV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  Backup,
  Backup_BackupState,
  Backup_BackupType,
} from "@/types/proto/v1/database_service";
import { DeploymentType } from "@/types/proto/v1/deployment";
import { Issue, Issue_Type } from "@/types/proto/v1/issue_service";
import { Plan, Plan_Spec } from "@/types/proto/v1/rollout_service";
import { extractBackupResourceName } from "@/utils";
import { trySetDefaultAssigneeByEnvironmentAndDeploymentType } from "../IssueV1/logic/initialize/assignee";

export type BackupRow = BBGridRow<Backup>;

type RestoreBackupContext = {
  target: RestoreTarget;
  backup: Backup;
};

type Section = {
  title: string;
  list: Backup[];
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

const hasPITRFeature = computed(() => {
  return useSubscriptionV1Store().hasInstanceFeature(
    "bb.feature.pitr",
    props.database.instanceEntity
  );
});

const createDatabasePrepForm =
  ref<InstanceType<typeof CreateDatabasePrepForm>>();

const columnList = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("common.status"),
      width: "auto",
    },
    {
      title: t("common.name"),
      width: "1fr",
    },
    {
      title: t("common.comment"),
      width: "2fr",
    },
    {
      title: t("common.time"),
      width: "auto",
    },
  ];
  if (props.allowEdit) {
    columns.push({
      title: "",
      width: "auto",
    });
  }
  return columns;
});

const backupSectionList = computed(() => {
  const manualList: Backup[] = [];
  const automaticList: Backup[] = [];
  const pitrList: Backup[] = [];
  const sectionList: Section[] = [
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
    if (backup.backupType === Backup_BackupType.MANUAL) {
      manualList.push(backup);
    } else if (backup.backupType === Backup_BackupType.AUTOMATIC) {
      automaticList.push(backup);
    } else if (backup.backupType === Backup_BackupType.PITR) {
      pitrList.push(backup);
    }
  }

  return sectionList;
});

const statusIconClass = (backup: Backup) => {
  const iconClass = "w-5 h-5";
  switch (backup.state) {
    case Backup_BackupState.PENDING_CREATE:
      return (
        iconClass +
        " bg-white border-2 border-info text-info hover:text-info-hover hover:border-info-hover"
      );
    case Backup_BackupState.DONE:
      return iconClass + " bg-success hover:bg-success-hover text-white";
    case Backup_BackupState.FAILED:
      return (
        iconClass + " bg-error text-white hover:text-white hover:bg-error-hover"
      );
  }
};

const allowRestore = (backup: Backup) => {
  return backup.state === Backup_BackupState.DONE;
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
      `Restore database [${database.databaseName}]`,
      `to backup snapshot [${extractBackupResourceName(
        restoreBackupContext.backup.name
      )}]`,
    ];

    const restoreDatabaseSpec: Plan_Spec = {
      id: uuidv4(),
      restoreDatabaseConfig: {
        backup: backup.name,
        target: database.name, // in-place
      },
    };
    const planCreate = Plan.fromJSON({
      steps: [{ specs: [restoreDatabaseSpec] }],
    });
    const issueCreate = Issue.fromJSON({
      title: issueNameParts.join(" "),
      type: Issue_Type.DATABASE_CHANGE,
    });
    await trySetDefaultAssigneeByEnvironmentAndDeploymentType(
      issueCreate,
      database.projectEntity,
      database.instanceEntity.environment,
      DeploymentType.DATABASE_RESTORE_PITR
    );
    const { createdIssue } = await experimentalCreateIssueByPlan(
      database.projectEntity,
      issueCreate,
      planCreate
    );

    router.push(`/issue/${createdIssue.uid}`);
  } catch {
    state.creatingRestoreIssue = false;
  }
};
</script>
