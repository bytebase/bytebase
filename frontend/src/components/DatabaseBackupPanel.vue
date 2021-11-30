<template>
  <div class="space-y-6 divide-y divide-block-border">
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
            v-if="allowDisableAutoBackup"
            type="button"
            class="ml-4 btn-normal"
            @click.prevent="toggleAutoBackup(false)"
          >
            Disable automatic backup
          </button>
          <router-link
            v-else
            class="normal-link text-sm"
            :to="`/environment/${database.instance.environment.id}`"
          >
            {{ `${backupPolicy} backup enforced and can't be disabled` }}
          </router-link>
        </div>
        <div class="mt-2 text-control">
          Backup will be taken on every
          <span class="text-accent">{{ autoBackupWeekdayText }}</span> at
          <span class="text-accent"> {{ autoBackupHourText }}</span>
        </div>
        <div class="mt-2">
          <label for="hookURL" class="textlabel">
            Webhook URL
          </label>
          <div class="mt-1 textinfolabel">
            An HTTP POST request will be sent to it after a successful backup. 
            <a
             href="https://docs.bytebase.com/use-bytebase/webhook-integration/database-webhook" 
             class="normal-link inline-flex flex-row items-center"
            >
            Learn more.
              <svg
                class="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                ></path>
              </svg>
            </a>
          </div>
          <input
            id="hookURL"
            name="hookURL"
            type="text"
            class="textfield mt-1 w-full"
            placeholder="https://betteruptime.com/api/v1/heartbeat/..."
            :disabled="!allowEdit"
            v-model="state.autoBackupUpdatedHookURL"
          />
          <button class="btn-primary mt-2" :disabled="!allowEdit || !URLChanged" @click.prevent="updateBackupHookURL()">Update</button>
        </div>
      </div>
      <div
        v-else
        class="flex items-center text-lg leading-6 font-medium text-main"
      >
        Automatic weekly backup
        <span class="ml-1 text-control-light">disabled</span>
        <button
          v-if="allowAdmin && !state.autoBackupEnabled"
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
          v-if="allowEdit"
          type="button"
          class="btn-normal whitespace-nowrap items-center"
          @click.prevent="state.showCreateBackupModal = true"
        >
          Backup now
        </button>
      </div>
      <BackupTable
        :database="database"
        :backup-list="backupList"
        :allow-edit="allowEdit"
      />
    </div>
    <BBModal
      v-if="state.showCreateBackupModal"
      :title="'Create a manual backup'"
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
import { computed, watchEffect, reactive, onUnmounted, PropType } from "vue";
import { useStore } from "vuex";
import {
  Backup,
  BackupCreate,
  BackupSetting,
  BackupSettingUpsert,
  Database,
  NORMAL_POLL_INTERVAL,
  PolicyBackupPlanPolicyPayload,
  POLL_JITTER,
  POST_CHANGE_POLL_INTERVAL,
  UNKNOWN_ID,
} from "../types";
import BackupTable from "../components/BackupTable.vue";
import DatabaseBackupCreateForm from "../components/DatabaseBackupCreateForm.vue";
import { cloneDeep, isEmpty, isEqual } from "lodash";

interface LocalState {
  showCreateBackupModal: boolean;
  autoBackupEnabled: boolean;
  autoBackupHour: number;
  autoBackupDayOfWeek: number;
  autoBackupHookURL: string;
  autoBackupUpdatedHookURL: string;
  pollBackupsTimer?: ReturnType<typeof setTimeout>;
}

export default {
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
    const store = useStore();

    const state = reactive<LocalState>({
      showCreateBackupModal: false,
      autoBackupEnabled: false,
      autoBackupHour: 0,
      autoBackupDayOfWeek: 0,
      autoBackupHookURL: '',
      autoBackupUpdatedHookURL: '',
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

    const prepareBackupPolicy = () => {
      store.dispatch("policy/fetchPolicyByEnvironmentAndType", {
        environmentId: props.database.instance.environment.id,
        type: "bb.policy.backup-plan",
      });
    };

    watchEffect(prepareBackupPolicy);

    const assignBackupSetting = (backupSetting: BackupSetting) => {
      state.autoBackupEnabled = backupSetting.enabled;
      state.autoBackupHour = backupSetting.hour;
      state.autoBackupDayOfWeek = backupSetting.dayOfWeek;
      state.autoBackupHookURL = backupSetting.hookURL;
      state.autoBackupUpdatedHookURL = backupSetting.hookURL;
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
      if (dayOfWeek == -1) {
        return "day";
      }
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

    const backupPolicy = computed(() => {
      const policy = store.getters["policy/policyByEnvironmentIdAndType"](
        props.database.instance.environment.id,
        "bb.policy.backup-plan"
      );
      return (policy.payload as PolicyBackupPlanPolicyPayload).schedule;
    });

    const allowDisableAutoBackup = computed(() => {
      return props.allowAdmin && backupPolicy.value == "UNSET";
    });

    const URLChanged = computed(() => {
      return !isEqual(state.autoBackupHookURL, state.autoBackupUpdatedHookURL);
    })

    const createBackup = (backupName: string) => {
      // Create backup
      const newBackup: BackupCreate = {
        databaseId: props.database.id!,
        name: backupName,
        status: "PENDING_CREATE",
        type: "MANUAL",
        storageBackend: "LOCAL",
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
        hookURL: "",
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

    const updateBackupHookURL = () => {
      const newBackupSetting: BackupSettingUpsert = {
        databaseId: props.database.id,
        enabled: state.autoBackupEnabled,
        hour: state.autoBackupHour,
        dayOfWeek: state.autoBackupDayOfWeek,
        hookURL: state.autoBackupUpdatedHookURL,
      };
      store
        .dispatch("backup/upsertBackupSetting", {
          newBackupSetting: newBackupSetting,
        })
        .then((backupSetting: BackupSetting) => {
          assignBackupSetting(backupSetting);
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Updated backup hook URL for database '${props.database.name}'.`,
          });
        });
    }

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
      URLChanged,
      updateBackupHookURL,
    };
  },
};
</script>
