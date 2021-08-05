<template>
  <div class="pt-6">
    <div class="text-lg leading-6 font-medium text-main mb-4">Automatic backup settings</div>
    <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-10">
      <div class="sm:col-span-3 sm:col-start-1">
        <label for="autoBackupHour" class="textlabel block"> Hour </label>
        <input
        required
        type="text"
        id="autoBackupHour"
        name="autoBackupHour"
        placeholder="auto-backup-hour"
        class="textfield mt-1 w-full"
        :value="state.autoBackupHour"
        @input="state.autoBackupHour=Number($event.target.value)"
        />
      </div>
      <div class="sm:col-span-5">
        <label for="autoBackupDayOfWeek" class="textlabel block"> Day of Week </label>
        <input
          type="text"
          id="autoBackupDayOfWeek"
          name="autoBackupDayOfWeek"
          placeholder="e.g. 0"
          class="textfield mt-1 w-full"
          :value="state.autoBackupDayOfWeek"
          @input="state.autoBackupDayOfWeek=Number($event.target.value)"
        />
      </div>
      <div class="sm:col-span-1">
        <label for="autoBackupEnabled" class="textlabel block"> Enabled </label>
        <input
          type="checkbox"
          id="autoBackupEnabled"
          :checked="state.autoBackupEnabled"
          @change="state.autoBackupEnabled=$event.target.checked"
        />
      </div>
      <div class="sm:col-span-1">
        <label> Update </label>
        <button
          @click.prevent="setAutoBackupSetting"
          type="button"
          class="btn-normal whitespace-nowrap items-center"
        >
          Set
        </button>
      </div>
    </div>
  </div>
  <div class="pt-6">
    <div class="text-lg leading-6 font-medium text-main mb-4">Take a backup</div>
    <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-10">
      <div class="sm:col-span-3 sm:col-start-1">
        <label for="backupName" class="textlabel block"> Name </label>
        <input
        required
        type="text"
        id="backupName"
        name="backupName"
        placeholder="backup-unique-name"
        class="textfield mt-1 w-full"
        :value="state.backupName"
        @input="updateInstance('backupName', $event.target.value)"
        />
      </div>
      <div class="sm:col-span-6">
        <label for="backupPath" class="textlabel block"> Path </label>
        <input
          type="text"
          id="backupPath"
          name="backupPath"
          placeholder="e.g. backup-1.sql | /tmp/backup-1.sql"
          class="textfield mt-1 w-full"
          :value="state.backupPath"
          @input="updateInstance('backupPath', $event.target.value)"
        />
      </div>
      <div class="sm:col-span-1">
        <label> Backup </label>
        <button
          @click.prevent="createBackup"
          type="button"
          class="btn-normal whitespace-nowrap items-center"
        >
          Backup now
        </button>
      </div>
    </div>
  </div>
  <div class="pt-6">
    <div class="text-lg leading-6 font-medium text-main mb-4">Backups</div>
    <BackupTable
      :backupList="backupList"
    />
  </div>
</template>

<script lang="ts">
import { computed, watchEffect, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { v1 as uuidv1 } from "uuid";
import { BackupCreate, BackupSettingSet, Database } from "../types";
import BackupTable from "../components/BackupTable.vue";

interface LocalState {
  backupName: string;
  backupPath: string;
  autoBackupEnabled: boolean;
  autoBackupHour: number;
  autoBackupDayOfWeek: number;
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
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      backupName: uuidv1(),
      backupPath: "backup.sql",
      autoBackupEnabled: false,
      autoBackupHour: 0,
      autoBackupDayOfWeek: 0,
    });

    const prepareBackupList = () => {
      store.dispatch("backup/fetchBackupListByDatabaseId", props.database.id);
    };

    watchEffect(prepareBackupList);

    const backupList = computed(() => {
      return store.getters["backup/backupListByDatabaseId"](props.database.id);
    });

    const updateInstance = (field: string, value: string) => {
      (state as any)[field] = value;
    };

    const createBackup = () => {
      // Create backup
      const newBackup: BackupCreate = {
        databaseId: props.database.id!,
        name: state.backupName!,
        status: "PENDING_CREATE",
        type: "MANUAL",
        storageBackend: "LOCAL",
        path: state.backupPath!,
        comment: "",
      };
      store.dispatch("backup/createBackup", {
        databaseId: props.database.id,
        newBackup: newBackup
      });
    };

    const prepareBackupSetting = () => {
      store.dispatch("backup/fetchBackupSettingByDatabaseId", props.database.id)
        .then(setting => {
          state.autoBackupEnabled = setting.enabled;
          state.autoBackupHour = setting.hour;
          state.autoBackupDayOfWeek = setting.dayOfWeek;
        });
    };

    watchEffect(prepareBackupSetting);

    const setAutoBackupSetting = () => {
      const newBackupSetting: BackupSettingSet = {
        databaseId: props.database.id!,
        enabled: state.autoBackupEnabled! ? 1 : 0,
        hour: state.autoBackupHour!,
        dayOfWeek: state.autoBackupDayOfWeek!,
      };
      store.dispatch("backup/setBackupSetting", {
        newBackupSetting: newBackupSetting,
      });
    };

    return {
      state,
      backupList,
      updateInstance,
      createBackup,
      setAutoBackupSetting,
    };
},
}
</script>