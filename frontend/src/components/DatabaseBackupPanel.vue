<template>
  <div class="max-w-6xl mx-auto px-6 space-y-6 divide-y divide-block-border">
    <div class="space-y-4">
      <div v-if="state.autoBackupEnabled" class="flex justify-between flex-col">
        <div class="flex justify-between">
          <div
            class="flex items-center text-lg leading-6 font-medium text-main"
          >
            Automatic weekly backup
            <span class="ml-1 text-success">enabled</span>
          </div>
          <button
            type="button"
            class="ml-4 btn-normal"
            @click.prevent="toggleAutoBackup(false)"
          >
            Disable automatic backup
          </button>
        </div>
        <div class="mt-2 text-control">
          Backup will be taken on every
          <span class="text-accent">{{ autoBackupWeekdayText }}</span> at
          <span class="text-accent"> {{ autoBackupHourText }}</span>
        </div>
      </div>
      <div
        v-else
        class="flex items-center text-lg leading-6 font-medium text-main"
      >
        Automatic weekly backup
        <span class="ml-1 text-control-light">disabled</span>
        <button
          v-if="!state.autoBackupEnabled"
          type="button"
          class="ml-4 btn-primary"
          @click.prevent="toggleAutoBackup(true)"
        >
          Enable backup
        </button>
      </div>
    </div>
    <div class="pt-6 space-y-4">
      <div class="flex justify-between items-center">
        <div class="text-lg leading-6 font-medium text-main">Backups</div>
        <button
          @click.prevent="state.showCreateBackupModal = true"
          type="button"
          class="btn-normal whitespace-nowrap items-center"
        >
          Backup now
        </button>
      </div>
      <BackupTable :database="database" :backupList="backupList" />
    </div>
    <BBModal
      v-if="state.showCreateBackupModal"
      :title="'Create a manual backup'"
      @close="state.showCreateBackupModal = false"
    >
      <DatabaseBackupCreateForm
        :database="database"
        @create="
          (backupName, comment) => {
            createBackup(backupName, comment);
            state.showCreateBackupModal = false;
          }
        "
        @cancel="state.showCreateBackupModal = false"
      />
    </BBModal>
  </div>
</template>

<script lang="ts">
import { computed, watchEffect, reactive, onUnmounted, PropType } from "vue";
import { useStore } from "vuex";
import {
  Backup,
  BackupCreate,
  BackupSetting,
  BackupSettingUpsert,
  Database,
  NORMAL_POLL_INTERVAL,
  POLL_JITTER,
  POST_CHANGE_POLL_INTERVAL,
  UNKNOWN_ID,
} from "../types";
import BackupTable from "../components/BackupTable.vue";
import DatabaseBackupCreateForm from "../components/DatabaseBackupCreateForm.vue";
import { cloneDeep, isEmpty } from "lodash";

interface LocalState {
  showCreateBackupModal: boolean;
  autoBackupEnabled: boolean;
  autoBackupHour: number;
  autoBackupDayOfWeek: number;
  pollBackupsTimer?: ReturnType<typeof setTimeout>;
}

export default {
  name: "DatabaseBackupPanel",
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
  },
  components: {
    BackupTable,
    DatabaseBackupCreateForm,
  },
  setup(props, ctx) {
    const store = useStore();

    // For now, we hard code the backup time to a time between 0:00 AM ~ 6:00 AM on Sunday local time.
    const DEFAULT_BACKUP_HOUR = Math.floor(Math.random() * 7);
    const DEFAULT_BACKUP_DAYOFWEEK = 0;
    const { hour, dayOfWeek } = localToUTC(
      DEFAULT_BACKUP_HOUR,
      DEFAULT_BACKUP_DAYOFWEEK
    );

    const state = reactive<LocalState>({
      showCreateBackupModal: false,
      autoBackupEnabled: false,
      autoBackupHour: hour,
      autoBackupDayOfWeek: dayOfWeek,
    });

    onUnmounted(() => {
      if (state.pollBackupsTimer) {
        clearInterval(state.pollBackupsTimer);
      }
    });

    const prepareBackupList = () => {
      store.dispatch("backup/fetchBackupListByDatabaseId", props.database.id);
    };

    watchEffect(prepareBackupList);

    const assignBackupSetting = (backupSetting: BackupSetting) => {
      state.autoBackupEnabled = backupSetting.enabled;
      state.autoBackupHour = backupSetting.hour;
      state.autoBackupDayOfWeek = backupSetting.dayOfWeek;
    };

    // List PENDING_CREATE backups first, followed by backups in createdTs descending order.
    const backupList = computed(() => {
      const list = cloneDeep(
        store.getters["backup/backupListByDatabaseId"](props.database.id)
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
      if (dayOfWeek == 0) {
        return "Sunday";
      }
      if (dayOfWeek == 1) {
        return "Monday";
      }
      if (dayOfWeek == 2) {
        return "Tuesday";
      }
      if (dayOfWeek == 3) {
        return "Wednesday";
      }
      if (dayOfWeek == 4) {
        return "Thursday";
      }
      if (dayOfWeek == 5) {
        return "Friday";
      }
      if (dayOfWeek == 6) {
        return "Saturday";
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

    const createBackup = (backupName: string, comment: string) => {
      // Create backup
      const newBackup: BackupCreate = {
        databaseId: props.database.id!,
        name: backupName,
        status: "PENDING_CREATE",
        type: "MANUAL",
        storageBackend: "LOCAL",
        comment,
      };
      store.dispatch("backup/createBackup", {
        databaseId: props.database.id,
        newBackup: newBackup,
      });
      pollBackups(POST_CHANGE_POLL_INTERVAL);
    };

    // pollBackups invalidates the current timer and schedule a new timer in <<interval>> microseconds
    const pollBackups = (interval: number) => {
      if (state.pollBackupsTimer) {
        clearInterval(state.pollBackupsTimer);
      }
      state.pollBackupsTimer = setTimeout(() => {
        store
          .dispatch("backup/fetchBackupListByDatabaseId", props.database.id)
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
      store
        .dispatch("backup/fetchBackupSettingByDatabaseId", props.database.id)
        .then((backupSetting: BackupSetting) => {
          // UNKNOWN_ID means database does not have backup setting and we should NOT overwrite the default setting.
          if (backupSetting.id != UNKNOWN_ID) {
            assignBackupSetting(backupSetting);
          }
        });
    };

    watchEffect(prepareBackupSetting);

    const toggleAutoBackup = (on: boolean) => {
      const newBackupSetting: BackupSettingUpsert = {
        databaseId: props.database.id,
        enabled: on,
        hour: state.autoBackupHour,
        dayOfWeek: state.autoBackupDayOfWeek,
      };
      store
        .dispatch("backup/upsertBackupSetting", {
          newBackupSetting: newBackupSetting,
        })
        .then((backupSetting: BackupSetting) => {
          assignBackupSetting(backupSetting);
          const action = on ? "Enabled" : "Disabled";
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `${action} automatic backup for database '${props.database.name}'.`,
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
      createBackup,
      toggleAutoBackup,
    };
  },
};
</script>
