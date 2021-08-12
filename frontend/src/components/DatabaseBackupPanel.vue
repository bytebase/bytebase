<template>
  <div class="pt-6 space-y-4">
    <div class="flex justify-between border-b border-block-border pb-4">
      <div v-if="state.autoBackupEnabled" class="flex flex-col">
        <div class="flex items-center text-lg leading-6 font-medium text-main">
          Automatic weekly backup
          <span class="ml-1 text-success">enabled</span>
        </div>
        <div class="mt-2 text-control">
          Backup will be taken on every
          <span class="text-accent">{{ autoBackupWeekdayText }}</span> at
          <span class="text-accent"> {{ autoBackupHourText }}</span>
        </div>
      </div>
      <div
        v-else
        class="flex items-center text-lg leading-6 font-medium text-control"
      >
        Weekly backup disabled
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
    <div
      v-if="state.autoBackupEnabled"
      class="space-y-4 border-b border-block-border pb-4"
    >
      <div>
        <label for="auto-backup-path-template" class="textlabel">
          Backup file path template
        </label>
        <div class="mt-1 textinfolabel">
          Backup file path template. TODO: pointing to doc
        </div>
        <input
          type="text"
          id="auto-backup-path-template"
          name="autoBackupPathTemplate"
          placeholder="e.g. 0"
          class="textfield mt-2 w-full"
          v-model="state.autoBackupPathTemplate"
        />
      </div>
      <div class="flex justify-between">
        <BBButtonConfirm
          :style="'DISABLE'"
          :buttonText="'Disable weekly backup'"
          :okText="'Disable'"
          :confirmTitle="`Disable weekly backup for '${database.name}'?`"
          :confirmDescription="'Existing automatic backups will be kept as is.'"
          :requireConfirm="true"
          @confirm="toggleAutoBackup(false)"
        />
        <button
          type="button"
          class="btn-primary"
          :disabled="!allowUpdateAutoBackupSetting"
          @click.prevent="updateAutoBackupSetting"
        >
          Update
        </button>
      </div>
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
    <BackupTable :backupList="backupList" />
  </div>
  <BBModal
    v-if="state.showCreateBackupModal"
    :title="'Create a manual backup'"
    @close="state.showCreateBackupModal = false"
  >
    <DatabaseBackupCreateForm
      :database="database"
      @create="
        (backupName, backupPath, comment) => {
          createBackup(backupName, backupPath, comment);
          state.showCreateBackupModal = false;
        }
      "
      @cancel="state.showCreateBackupModal = false"
    />
  </BBModal>
</template>

<script lang="ts">
import { computed, watchEffect, reactive, onUnmounted, PropType } from "vue";
import { useStore } from "vuex";
import {
  Backup,
  BackupCreate,
  BackupSetting,
  BackupSettingSet,
  Database,
} from "../types";
import BackupTable from "../components/BackupTable.vue";
import DatabaseBackupCreateForm from "../components/DatabaseBackupCreateForm.vue";
import { isEmpty } from "lodash";

interface LocalState {
  showCreateBackupModal: boolean;
  autoBackupEnabled: boolean;
  autoBackupHour: number;
  autoBackupDayOfWeek: number;
  autoBackupPathTemplate: string;
  originalAutoBackupPathTemplate: string;
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
    const NORMAL_BACKUPS_POLL_INTERVAL = 10000;
    // Add jitter to avoid timer from different clients converging to the same polling frequency.
    const POLL_JITTER = 500;

    const state = reactive<LocalState>({
      showCreateBackupModal: false,
      autoBackupEnabled: false,
      autoBackupHour: 0,
      autoBackupDayOfWeek: 0,
      autoBackupPathTemplate: "",
      originalAutoBackupPathTemplate: "",
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
      var { hour, dayOfWeek } = alignUTC(
        backupSetting.hour,
        backupSetting.dayOfWeek,
        props.database.timezoneOffset
      );
      state.autoBackupEnabled = backupSetting.enabled;
      state.autoBackupHour = hour;
      state.autoBackupDayOfWeek = dayOfWeek;
      state.autoBackupPathTemplate = backupSetting.pathTemplate;
      state.originalAutoBackupPathTemplate = backupSetting.pathTemplate;
    };

    const backupList = computed(() => {
      return store.getters["backup/backupListByDatabaseId"](props.database.id);
    });

    const autoBackupWeekdayText = computed(() => {
      var { dayOfWeek } = alignUTC(
        state.autoBackupHour,
        state.autoBackupDayOfWeek,
        -props.database.timezoneOffset
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
      var { hour } = alignUTC(
        state.autoBackupHour,
        state.autoBackupDayOfWeek,
        -props.database.timezoneOffset
      );

      return `${String(hour).padStart(2, "0")}:00 (${
        props.database.timezoneName
      })`;
    });

    const allowUpdateAutoBackupSetting = computed(() => {
      return (
        !isEmpty(state.autoBackupPathTemplate) &&
        state.autoBackupPathTemplate != state.originalAutoBackupPathTemplate
      );
    });

    const createBackup = (
      backupName: string,
      backupPath: string,
      comment: string
    ) => {
      // Create backup
      const newBackup: BackupCreate = {
        databaseId: props.database.id!,
        name: backupName,
        status: "PENDING_CREATE",
        type: "MANUAL",
        storageBackend: "LOCAL",
        path: backupPath,
        comment,
      };
      store.dispatch("backup/createBackup", {
        databaseId: props.database.id,
        newBackup: newBackup,
      });
      pollBackups(NORMAL_BACKUPS_POLL_INTERVAL);
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
              pollBackups(Math.min(interval * 2, NORMAL_BACKUPS_POLL_INTERVAL));
            }
          });
      }, Math.max(1000, Math.min(interval, NORMAL_BACKUPS_POLL_INTERVAL) + (Math.random() * 2 - 1) * POLL_JITTER));
    };

    const prepareBackupSetting = () => {
      store
        .dispatch("backup/fetchBackupSettingByDatabaseId", props.database.id)
        .then((backupSetting: BackupSetting) => {
          assignBackupSetting(backupSetting);
        });
    };

    watchEffect(prepareBackupSetting);

    const toggleAutoBackup = (on: boolean) => {
      var { hour, dayOfWeek } = alignUTC(
        state.autoBackupHour,
        state.autoBackupDayOfWeek,
        -props.database.timezoneOffset
      );
      const newBackupSetting: BackupSettingSet = {
        databaseId: props.database.id,
        enabled: on,
        hour: hour,
        dayOfWeek: dayOfWeek,
        pathTemplate: state.autoBackupPathTemplate,
      };
      store
        .dispatch("backup/setBackupSetting", {
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

    const updateAutoBackupSetting = () => {
      var { hour, dayOfWeek } = alignUTC(
        state.autoBackupHour,
        state.autoBackupDayOfWeek,
        -props.database.timezoneOffset
      );
      const newBackupSetting: BackupSettingSet = {
        databaseId: props.database.id,
        enabled: state.autoBackupEnabled,
        hour: hour,
        dayOfWeek: dayOfWeek,
        pathTemplate: state.autoBackupPathTemplate,
      };
      store
        .dispatch("backup/setBackupSetting", {
          newBackupSetting: newBackupSetting,
        })
        .then((backupSetting: BackupSetting) => {
          assignBackupSetting(backupSetting);
        });
    };

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
      allowUpdateAutoBackupSetting,
      createBackup,
      toggleAutoBackup,
      updateAutoBackupSetting,
    };
  },
};
</script>
