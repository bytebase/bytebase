<template>
  <div class="flex flex-col space-y-4">
    <div class="text-lg leading-6 font-medium text-main">Migration History</div>
    <MigrationHistoryTable
      v-if="state.migrationSetupStatus == 'OK'"
      :databaseSectionList="[database]"
      :historySectionList="migrationHistorySectionList"
    />
    <BBAttention
      v-else
      :style="`WARN`"
      :title="attentionTitle"
      :actionText="allowConfigInstance ? 'Config instance' : ''"
      @click-action="configInstance"
    />
  </div>
</template>

<script lang="ts">
import { computed, PropType, reactive, watchEffect } from "@vue/runtime-core";
import { useStore } from "vuex";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import {
  Database,
  InstanceMigration,
  MigrationHistory,
  MigrationSchemaStatus,
} from "../types";
import { useRouter } from "vue-router";
import { BBTableSectionDataSource } from "../bbkit/types";
import { instanceSlug, isDBAOrOwner } from "../utils";

interface LocalState {
  migrationSetupStatus: MigrationSchemaStatus;
}

export default {
  name: "DatabaseMigrationHistoryPanel",
  components: { MigrationHistoryTable },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      migrationSetupStatus: "OK",
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareMigrationHistoryList = () => {
      store
        .dispatch("instance/checkMigrationSetup", props.database.instance.id)
        .then((migration: InstanceMigration) => {
          state.migrationSetupStatus = migration.status;
          if (state.migrationSetupStatus == "OK") {
            store.dispatch("instance/fetchMigrationHistory", {
              instanceId: props.database.instance.id,
              databaseName: props.database.name,
            });
          }
        });
    };

    watchEffect(prepareMigrationHistoryList);

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowConfigInstance = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const attentionTitle = computed((): string => {
      if (state.migrationSetupStatus == "NOT_EXIST") {
        return `Missing migration history schema on instance "${props.database.instance.name}"`;
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return `Unable to connect instance "${props.database.instance.name}" to retrieve migration history`;
      }
      return "";
    });

    const migrationHistorySectionList = computed(
      (): BBTableSectionDataSource<MigrationHistory>[] => {
        return [
          {
            title: "",
            list: store.getters[
              "instance/migrationHistoryListByInstanceIdAndDatabaseName"
            ](props.database.instance.id, props.database.name),
          },
        ];
      }
    );

    const configInstance = () => {
      router.push(`/instance/${instanceSlug(props.database.instance)}`);
    };

    return {
      state,
      allowConfigInstance,
      attentionTitle,
      migrationHistorySectionList,
      configInstance,
    };
  },
};
</script>
