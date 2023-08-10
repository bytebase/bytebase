<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div class="space-y-4">
      <div v-if="state.autoBackupEnabled" class="flex justify-between flex-col">
        <div class="flex justify-between">
          <div
            class="flex items-center text-lg leading-6 font-medium text-main"
          >
            {{ $t("database.automatic-backup") }}
            <span class="ml-1 text-success">
              {{ $t("database.backup.enabled") }}
            </span>
          </div>
          <div class="flex items-center">
            <router-link
              v-if="hasBackupPolicyViolation"
              class="flex items-center normal-link text-sm"
              :to="`/environment/${database.instanceEntity.environmentEntity.uid}`"
            >
              <heroicons-outline:exclamation-circle class="w-4 h-4 mr-1" />
              <span>{{ $t("database.backup-policy-violation") }}</span>
            </router-link>
            <button
              v-if="allowAdmin"
              type="button"
              class="ml-4 btn-normal"
              @click.prevent="state.showBackupSettingModal = true"
            >
              {{
                hasBackupPolicyViolation ? $t("common.fix") : $t("common.edit")
              }}
            </button>
          </div>
        </div>
        <div class="mt-2 text-control">
          <i18n-t keypath="database.backup-info.template">
            <template #dayOrWeek>
              <span class="text-accent">{{ autoBackupWeekdayText }}</span>
            </template>
            <template #time>
              <span class="text-accent"> {{ autoBackupHourText }}</span>
            </template>
            <template #retentionDays>
              <span class="text-accent"> {{ autoBackupRetentionDays }}</span>
            </template>
          </i18n-t>
        </div>
        <div class="mt-2">
          <label for="hookUrl" class="textlabel"> Webhook URL </label>
          <div class="mt-1 textinfolabel">
            {{
              $t(
                "database.an-http-post-request-will-be-sent-to-it-after-a-successful-backup"
              )
            }}
            <a
              href="https://bytebase.com/docs/backup/#post-backup-webhook?source=console"
              class="normal-link inline-flex flex-row items-center"
            >
              {{ $t("common.learn-more") }}
              <heroicons-outline:external-link class="w-4 h-4" />
            </a>
          </div>
          <input
            id="hookUrl"
            v-model="state.autoBackupUpdatedHookUrl"
            name="hookUrl"
            type="text"
            class="textfield mt-1 w-full"
            placeholder="https://betteruptime.com/api/v1/heartbeat/..."
            :disabled="!allowEdit"
          />
          <button
            v-if="allowEdit"
            class="btn-primary mt-2"
            :disabled="!allowEdit || !urlChanged"
            @click.prevent="updateBackupHookUrl()"
          >
            {{ $t("common.update") }}
          </button>
        </div>
      </div>
      <div
        v-else
        class="flex items-center text-lg leading-6 font-medium text-main"
      >
        {{ $t("database.automatic-backup") }}
        <span class="ml-1 text-control-light">{{
          $t("database.backup.disabled")
        }}</span>
        <button
          v-if="allowAdmin && !state.autoBackupEnabled"
          type="button"
          class="ml-4 btn-primary"
          @click.prevent="state.showBackupSettingModal = true"
        >
          {{ $t("database.enable-backup") }}
        </button>
      </div>
    </div>
    <div class="pt-6 space-y-4">
      <div class="flex justify-between items-center">
        <div class="text-lg leading-6 font-medium text-main">
          {{ $t("common.backups") }}
        </div>

        <div class="flex-1 flex items-center justify-end">
          <PITRRestoreButton
            v-if="allowAdmin"
            :database="database"
            :allow-admin="allowAdmin"
          />

          <button
            v-if="allowEdit && !disableBackupButton"
            type="button"
            class="btn-normal whitespace-nowrap items-center"
            @click.prevent="state.showCreateBackupModal = true"
          >
            {{ $t("database.backup-now") }}
          </button>
        </div>
      </div>
      <BackupTable
        :database="database"
        :backup-list="backupList"
        :allow-edit="allowEdit"
      />
    </div>

    <Drawer
      :show="state.showBackupSettingModal"
      @close="state.showBackupSettingModal = false"
    >
      <DatabaseBackupSettingForm
        :database="database"
        :allow-admin="allowAdmin"
        :backup-policy="backupPolicy"
        :backup-setting="state.backupSetting"
        @cancel="state.showBackupSettingModal = false"
        @update="updateBackupSetting"
      />
    </Drawer>

    <BBModal
      v-if="state.showCreateBackupModal"
      :title="$t('database.create-a-manual-backup')"
      @close="state.showCreateBackupModal = false"
    >
      <DatabaseBackupCreateForm
        :database="database"
        @create="
          (backupName: string) => {
            createBackup(backupName);
            state.showCreateBackupModal = false;
          }
        "
        @cancel="state.showCreateBackupModal = false"
      />
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import {
  computed,
  watchEffect,
  reactive,
  onUnmounted,
  PropType,
  onBeforeMount,
} from "vue";
import { useI18n } from "vue-i18n";
import {
  DatabaseBackupSettingForm,
  BackupTable,
  levelOfSchedule,
  localFromUTC,
  parseScheduleFromBackupSetting,
} from "@/components/DatabaseBackup/";
import DatabaseBackupCreateForm from "@/components/DatabaseBackupCreateForm.vue";
import PITRRestoreButton from "@/components/DatabaseDetail/PITRRestoreButton.vue";
import { Drawer } from "@/components/v2";
import {
  pushNotification,
  useBackupV1Store,
  useGracefulRequest,
} from "@/store";
import {
  usePolicyV1Store,
  defaultBackupSchedule,
} from "@/store/modules/v1/policy";
import {
  ComposedDatabase,
  NORMAL_POLL_INTERVAL,
  POLL_JITTER,
  MINIMUM_POLL_INTERVAL,
} from "@/types";
import {
  Backup,
  BackupSetting,
  Backup_BackupState,
  Backup_BackupType,
} from "@/types/proto/v1/database_service";
import {
  PolicyType,
  BackupPlanSchedule,
} from "@/types/proto/v1/org_policy_service";
import { instanceV1HasBackupRestore } from "@/utils";

interface LocalState {
  showCreateBackupModal: boolean;
  autoBackupEnabled: boolean;
  autoBackupHour: number;
  autoBackupDayOfWeek: number;
  autoBackupRetentionPeriodTs: number;
  autoBackupHookUrl: string;
  autoBackupUpdatedHookUrl: string;
  pollBackupsTimer?: ReturnType<typeof setTimeout>;
  showBackupSettingModal: boolean;
  backupSetting: BackupSetting | undefined;
}

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  allowAdmin: {
    required: true,
    type: Boolean,
  },
  allowEdit: {
    required: true,
    type: Boolean,
  },
});

const backupStore = useBackupV1Store();
const policyV1Store = usePolicyV1Store();
const { t } = useI18n();

const state = reactive<LocalState>({
  showCreateBackupModal: false,
  autoBackupEnabled: false,
  autoBackupHour: 0,
  autoBackupDayOfWeek: 0,
  autoBackupRetentionPeriodTs: 0,
  autoBackupHookUrl: "",
  autoBackupUpdatedHookUrl: "",
  showBackupSettingModal: false,
  backupSetting: undefined,
});

onUnmounted(() => {
  if (state.pollBackupsTimer) {
    clearInterval(state.pollBackupsTimer);
  }
});

const prepareBackupList = () => {
  backupStore.fetchBackupList({
    parent: props.database.name,
  });
};

watchEffect(prepareBackupList);

const prepareBackupPolicy = () => {
  policyV1Store.getOrFetchPolicyByParentAndType({
    parentPath: props.database.instanceEntity.environment,
    policyType: PolicyType.BACKUP_PLAN,
  });
};

watchEffect(prepareBackupPolicy);

const assignBackupSetting = (backupSetting: BackupSetting) => {
  const schedule = backupStore.parseBackupSchedule(backupSetting.cronSchedule);
  state.autoBackupEnabled = backupSetting.cronSchedule !== "";
  state.autoBackupHour = schedule.hourOfDay;
  state.autoBackupDayOfWeek = schedule.dayOfWeek;
  state.autoBackupRetentionPeriodTs =
    backupSetting.backupRetainDuration?.seconds ?? 0;
  state.autoBackupHookUrl = backupSetting.hookUrl;
  state.autoBackupUpdatedHookUrl = backupSetting.hookUrl;

  state.backupSetting = backupSetting;
};

const disableBackupButton = computed(() => {
  return !instanceV1HasBackupRestore(props.database.instanceEntity);
});

// List PENDING_CREATE backups first, followed by backups in createdTs descending order.
const backupList = computed(() => {
  const list = [...backupStore.backupListByDatabase(props.database.name)];
  return list.sort((a, b) => {
    if (
      a.state === Backup_BackupState.PENDING_CREATE &&
      b.state !== Backup_BackupState.PENDING_CREATE
    ) {
      return -1;
    } else if (
      a.state !== Backup_BackupState.PENDING_CREATE &&
      b.state === Backup_BackupState.PENDING_CREATE
    ) {
      return 1;
    }

    return (b.createTime?.getTime() ?? 0) - (a.createTime?.getTime() ?? 0);
  });
});

const autoBackupWeekdayText = computed(() => {
  const { dayOfWeek } = localFromUTC(
    state.autoBackupHour,
    state.autoBackupDayOfWeek
  );
  if (dayOfWeek == -1) {
    return t("database.week.day");
  }
  if (dayOfWeek == 0) {
    return t("database.week.Sunday");
  }
  if (dayOfWeek == 1) {
    return t("database.week.Monday");
  }
  if (dayOfWeek == 2) {
    return t("database.week.Tuesday");
  }
  if (dayOfWeek == 3) {
    return t("database.week.Wednesday");
  }
  if (dayOfWeek == 4) {
    return t("database.week.Thursday");
  }
  if (dayOfWeek == 5) {
    return t("database.week.Friday");
  }
  if (dayOfWeek == 6) {
    return t("database.week.Saturday");
  }
  return `Invalid day of week: ${dayOfWeek}`;
});

const autoBackupHourText = computed(() => {
  const { hour } = localFromUTC(
    state.autoBackupHour,
    state.autoBackupDayOfWeek
  );

  return `${String(hour).padStart(2, "0")}:00 (${
    Intl.DateTimeFormat().resolvedOptions().timeZone
  })`;
});

const autoBackupRetentionDays = computed(() => {
  return state.autoBackupRetentionPeriodTs / 3600 / 24;
});

const backupPolicy = computed((): BackupPlanSchedule => {
  const policy = policyV1Store.getPolicyByParentAndType({
    parentPath: props.database.instanceEntity.environment,
    policyType: PolicyType.BACKUP_PLAN,
  });
  return policy?.backupPlanPolicy?.schedule ?? defaultBackupSchedule;
});

const hasBackupPolicyViolation = computed((): boolean => {
  if (!state.backupSetting) return false;
  if (!backupPolicy.value) return false;
  const schedule = parseScheduleFromBackupSetting(
    state.backupSetting.cronSchedule
  );
  return levelOfSchedule(schedule) < levelOfSchedule(backupPolicy.value);
});

const updateBackupSetting = (setting: BackupSetting) => {
  state.showBackupSettingModal = false;
  assignBackupSetting(setting);
};

const urlChanged = computed(() => {
  return !isEqual(state.autoBackupHookUrl, state.autoBackupUpdatedHookUrl);
});

const createBackup = async (backupName: string) => {
  // Create backup

  useGracefulRequest(async () => {
    const parent = props.database.name;
    const backupCreate = Backup.fromJSON({
      name: `${parent}/backups/${backupName}`,
      backupType: Backup_BackupType.MANUAL,
    });
    await backupStore.createBackup(
      backupCreate,
      parent,
      true /* refreshList */
    );
    pollBackups(MINIMUM_POLL_INTERVAL);
  });
};

// pollBackups invalidates the current timer and schedule a new timer in <<interval>> microseconds
const pollBackups = (interval: number) => {
  if (state.pollBackupsTimer) {
    clearInterval(state.pollBackupsTimer);
  }
  state.pollBackupsTimer = setTimeout(() => {
    backupStore
      .fetchBackupList({
        parent: props.database.name,
      })
      .then((backups) => {
        const pending = backups.some(
          (backup) => backup.state === Backup_BackupState.PENDING_CREATE
        );
        if (pending) {
          pollBackups(Math.min(interval * 2, NORMAL_POLL_INTERVAL));
        }
      });
  }, Math.max(1000, Math.min(interval, NORMAL_POLL_INTERVAL) + (Math.random() * 2 - 1) * POLL_JITTER));
};

const prepareBackupSetting = () => {
  backupStore
    .fetchBackupSetting(props.database.name)
    .then((backupSetting: BackupSetting | undefined) => {
      if (backupSetting) {
        assignBackupSetting(backupSetting);
      }
    });
};

onBeforeMount(prepareBackupSetting);

const updateBackupHookUrl = () => {
  backupStore
    .upsertBackupSetting({
      name: `${props.database.name}/backupSetting`,
      cronSchedule: backupStore.buildSimpleSchedule({
        enabled: state.autoBackupEnabled,
        hourOfDay: state.autoBackupHour,
        dayOfWeek: state.autoBackupDayOfWeek,
      }),
      hookUrl: state.autoBackupUpdatedHookUrl,
      backupRetainDuration: {
        seconds: state.autoBackupRetentionPeriodTs,
        nanos: 0,
      },
    })
    .then((backupSetting: BackupSetting) => {
      assignBackupSetting(backupSetting);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "database.updated-backup-webhook-url-for-database-props-database-name",
          [props.database.databaseName]
        ),
      });
    });
};
</script>
