<template>
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
        <label> Click </label>
        <button
          @click.prevent="testConnection"
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
import { computed, watchEffect, reactive, PropType, ComputedRef } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { v1 as uuidv1 } from "uuid";
import { Backup, Database } from "../types";
import BackupTable from "../components/BackupTable.vue";

interface LocalState {
  backupName: string;
  backupPath: string;
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

    return {
      state,
      backupList,
      updateInstance,
    };
},
}
</script>