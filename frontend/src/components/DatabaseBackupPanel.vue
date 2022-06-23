<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div class="space-y-4">
      <div v-if="state.autoBackupEnabled" class="flex justify-between flex-col">
        <div class="flex justify-between">
          <div
            class="flex items-center text-lg leading-6 font-medium text-main"
          >
            {{ $t("database.automatic-weekly-backup") }}
            <span class="ml-1 text-success">
              {{ $t("database.backup.enabled") }}
            </span>
          </div>
          <button
            v-if="allowDisableAutoBackup"
            type="button"
            class="ml-4 btn-normal"
            @click.prevent="toggleAutoBackup(false)"
          >
            {{ $t("database.disable-automatic-backup") }}
          </button>
          <router-link
            v-else
            class="normal-link text-sm"
            :to="`/environment/${database.instance.environment.id}`"
          >
            {{
              $t("database.backuppolicy-backup-enforced-and-cant-be-disabled", [
                $t(`database.backup-policy.${backupPolicy}`),
              ])
            }}
          </router-link>
        </div>
        <div class="mt-2 text-control">
          <i18n-t keypath="database.backup-info.template">
            <template #dayOrWeek>
              <span class="text-accent">{{ autoBackupWeekdayText }}</span>
            </template>
            <template #time>
              <span class="text-accent"> {{ autoBackupHourText }}</span>
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
              href="https://bytebase.com/docs/use-bytebase/webhook-integration/database-webhook?source=console"
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
            class="btn-primary mt-2"
            :disabled="!allowEdit || !UrlChanged"
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
        {{ $t("database.automatic-weekly-backup") }}
        <span class="ml-1 text-control-light">{{
          $t("database.backup.disabled")
        }}</span>
        <button
          v-if="allowAdmin && !state.autoBackupEnabled"
          type="button"
          class="ml-4 btn-primary"
          @click.prevent="toggleAutoBackup(true)"
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
            v-if="isDev && allowAdmin"
            :database="database"
            :allow-admin="allowAdmin"
          />

          <button
            v-if="allowEdit"
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
    <BBModal
      v-if="state.showCreateBackupModal"
      :title="$t('database.create-a-manual-backup')"
      @close="state.showCreateBackupModal = false"
    >
      <DatabaseBackupCreateForm
        :database="database"
        @create="
          (backupName) => {
            createBackup(backupName);
            state.showCreateBackupModal = false;
          }
        "
        @cancel="state.showCreateBackupModal = false"
      />
    </BBModal>
  </div>
</template>

<script lang="ts">
import {
  computed,
  watchEffect,
  reactive,
  onUnmounted,
  PropType,
  defineComponent,
} from "vue";
import {
  Backup,
  BackupCreate,
  BackupSetting,
  BackupSettingUpsert,
  Database,
  NORMAL_POLL_INTERVAL,
  BackupPlanPolicyPayload,
  POLL_JITTER,
  MINIMUM_POLL_INTERVAL,
  UNKNOWN_ID,
} from "../types";
import BackupTable from "../components/BackupTable.vue";
import DatabaseBackupCreateForm from "../components/DatabaseBackupCreateForm.vue";
import { cloneDeep, isEqual } from "lodash-es";
import { useI18n } from "vue-i18n";
import { pushNotification, useBackupStore, usePolicyStore } from "@/store";

interface LocalState {
  showCreateBackupModal: boolean;
  autoBackupEnabled: boolean;
  autoBackupHour: number;
  autoBackupDayOfWeek: number;
  autoBackupHookUrl: string;
  autoBackupUpdatedHookUrl: string;
  pollBackupsTimer?: ReturnType<typeof setTimeout>;
}

export default defineComponent({
  name: "DatabaseBackupPanel",
  components: {
    BackupTable,
    DatabaseBackupCreateForm,
  },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    allowAdmin: {
      required: true,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  setup(props) {
    const backupStore = useBackupStore();
    const policyStore = usePolicyStore();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      showCreateBackupModal: false,
      autoBackupEnabled: false,
      autoBackupHour: 0,
      autoBackupDayOfWeek: 0,
      autoBackupHookUrl: "",
      autoBackupUpdatedHookUrl: "",
    });

    onUnmounted(() => {
      if (state.pollBackupsTimer) {
        clearInterval(state.pollBackupsTimer);
      }
    });

    const prepareBackupList = () => {
      backupStore.fetchBackupListByDatabaseId(props.database.id);
    };

    watchEffect(prepareBackupList);

    const prepareBackupPolicy = () => {
      policyStore.fetchPolicyByEnvironmentAndType({
        environmentId: props.database.instance.environment.id,
        type: "bb.policy.backup-plan",
      });
    };

    watchEffect(prepareBackupPolicy);

    const assignBackupSetting = (backupSetting: BackupSetting) => {
      state.autoBackupEnabled = backupSetting.enabled;
      state.autoBackupHour = backupSetting.hour;
      state.autoBackupDayOfWeek = backupSetting.dayOfWeek;
      state.autoBackupHookUrl = backupSetting.hookUrl;
      state.autoBackupUpdatedHookUrl = backupSetting.hookUrl;
    };

    // List PENDING_CREATE backups first, followed by backups in createdTs descending order.
    const backupList = computed(() => {
      const list = cloneDeep(
        backupStore.backupListByDatabaseId(props.database.id)
      );
      return list.sort((a: Backup, b: Backup) => {
        if (a.status == "PENDING_CREATE" && b.status != "PENDING_CREATE") {
          return -1;
        } else if (
          a.status != "PENDING_CREATE" &&
          b.status == "PENDING_CREATE"
        ) {
          return 1;
        }

        return b.createdTs - a.createdTs;
      });
    });

    const autoBackupWeekdayText = computed(() => {
      var { dayOfWeek } = localFromUTC(
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
      var { hour } = localFromUTC(
        state.autoBackupHour,
        state.autoBackupDayOfWeek
      );

      return `${String(hour).padStart(2, "0")}:00 (${
        Intl.DateTimeFormat().resolvedOptions().timeZone
      })`;
    });

    const backupPolicy = computed(() => {
      const policy = policyStore.getPolicyByEnvironmentIdAndType(
        props.database.instance.environment.id,
        "bb.policy.backup-plan"
      );
      const payload = policy?.payload;
      return (payload as BackupPlanPolicyPayload | undefined)?.schedule;
    });

    const allowDisableAutoBackup = computed(() => {
      return props.allowAdmin && backupPolicy.value == "UNSET";
    });

    const UrlChanged = computed(() => {
      return !isEqual(state.autoBackupHookUrl, state.autoBackupUpdatedHookUrl);
    });

    const createBackup = (backupName: string) => {
      // Create backup
      const newBackup: BackupCreate = {
        databaseId: props.database.id!,
        name: backupName,
        type: "MANUAL",
        storageBackend: "LOCAL",
      };
      backupStore.createBackup({
        databaseId: props.database.id,
        newBackup: newBackup,
      });
      pollBackups(MINIMUM_POLL_INTERVAL);
    };

    // pollBackups invalidates the current timer and schedule a new timer in <<interval>> microseconds
    const pollBackups = (interval: number) => {
      if (state.pollBackupsTimer) {
        clearInterval(state.pollBackupsTimer);
      }
      state.pollBackupsTimer = setTimeout(() => {
        backupStore
          .fetchBackupListByDatabaseId(props.database.id)
          .then((backups: Backup[]) => {
            var pending = false;
            for (let idx in backups) {
              if (backups[idx].status.includes("PENDING")) {
                pending = true;
                continue;
              }
            }
            if (pending) {
              pollBackups(Math.min(interval * 2, NORMAL_POLL_INTERVAL));
            }
          });
      }, Math.max(1000, Math.min(interval, NORMAL_POLL_INTERVAL) + (Math.random() * 2 - 1) * POLL_JITTER));
    };

    const prepareBackupSetting = () => {
      backupStore
        .fetchBackupSettingByDatabaseId(props.database.id)
        .then((backupSetting: BackupSetting) => {
          // UNKNOWN_ID means database does not have backup setting and we should NOT overwrite the default setting.
          if (backupSetting.id != UNKNOWN_ID) {
            assignBackupSetting(backupSetting);
          }
        });
    };

    watchEffect(prepareBackupSetting);

    const toggleAutoBackup = (on: boolean) => {
      // For now, we hard code the backup time to a time between 0:00 AM ~ 6:00 AM on Sunday local time.
      // Choose a new random time everytime we re-enabling the auto backup. This is a workaround for
      // user to choose a desired backup window.
      const DEFAULT_BACKUP_HOUR = () => Math.floor(Math.random() * 7);
      const DEFAULT_BACKUP_DAYOFWEEK = 0;
      const { hour, dayOfWeek } = localToUTC(
        DEFAULT_BACKUP_HOUR(),
        DEFAULT_BACKUP_DAYOFWEEK
      );
      const newBackupSetting: BackupSettingUpsert = {
        databaseId: props.database.id,
        enabled: on,
        hour: on ? hour : state.autoBackupHour,
        dayOfWeek: on
          ? backupPolicy.value == "DAILY"
            ? -1
            : dayOfWeek
          : state.autoBackupDayOfWeek,
        hookUrl: "",
      };
      backupStore
        .upsertBackupSetting({
          newBackupSetting: newBackupSetting,
        })
        .then((backupSetting: BackupSetting) => {
          assignBackupSetting(backupSetting);
          const action = on ? t("database.enabled") : t("database.disabled");
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t(
              "database.action-automatic-backup-for-database-props-database-name",
              [action, props.database.name]
            ),
          });
        });
    };

    const updateBackupHookUrl = () => {
      const newBackupSetting: BackupSettingUpsert = {
        databaseId: props.database.id,
        enabled: state.autoBackupEnabled,
        hour: state.autoBackupHour,
        dayOfWeek: state.autoBackupDayOfWeek,
        hookUrl: state.autoBackupUpdatedHookUrl,
      };
      backupStore
        .upsertBackupSetting({
          newBackupSetting: newBackupSetting,
        })
        .then((backupSetting: BackupSetting) => {
          assignBackupSetting(backupSetting);
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t(
              "database.updated-backup-webhook-url-for-database-props-database-name",
              [props.database.name]
            ),
          });
        });
    };

    function localToUTC(hour: number, dayOfWeek: number) {
      return alignUTC(hour, dayOfWeek, new Date().getTimezoneOffset() * 60);
    }

    function localFromUTC(hour: number, dayOfWeek: number) {
      return alignUTC(hour, dayOfWeek, -new Date().getTimezoneOffset() * 60);
    }

    function alignUTC(hour: number, dayOfWeek: number, offsetInSecond: number) {
      if (hour != -1) {
        hour = hour + offsetInSecond / 60 / 60;
        var dayOffset = 0;
        if (hour > 23) {
          hour = hour - 24;
          dayOffset = 1;
        }
        if (hour < 0) {
          hour = hour + 24;
          dayOffset = -1;
        }
        if (dayOfWeek != -1) {
          dayOfWeek = (7 + dayOfWeek + dayOffset) % 7;
        }
      }
      return { hour, dayOfWeek };
    }

    return {
      state,
      backupList,
      autoBackupWeekdayText,
      autoBackupHourText,
      allowDisableAutoBackup,
      backupPolicy,
      createBackup,
      toggleAutoBackup,
      UrlChanged,
      updateBackupHookUrl,
    };
  },
});
</script>
