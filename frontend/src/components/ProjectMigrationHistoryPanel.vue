<template>
  <div class="flex flex-col space-y-2">
    <div class="textinfolabel">
      For database having migration history, we list up to 5 most recent
      histories below. You can click the database name to view all histories.
    </div>
    <MigrationHistoryTable
      :mode="'PROJECT'"
      :historySectionList="state.migrationHistorySectionList"
    />
  </div>
</template>

<script lang="ts">
import { PropType, reactive, ref, watchEffect } from "@vue/runtime-core";
import { useStore } from "vuex";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import {
  Database,
  InstanceMigration,
  MigrationHistory,
  Project,
} from "../types";
import { useRouter } from "vue-router";
import { BBTableSectionDataSource } from "../bbkit/types";
import { fullDatabasePath } from "../utils";

// Show at most 5 recent migration history for each database
const MAX_MIGRAION_HISTORY_COUNT = 5;

interface LocalState {
  migrationHistorySectionList: BBTableSectionDataSource<MigrationHistory>[];
}

export default {
  name: "ProjectMigrationHistoryPanel",
  components: { MigrationHistoryTable },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
    databaseList: {
      required: true,
      type: Object as PropType<Database[]>,
    },
  },
  setup(props, ctx) {
    const searchField = ref();

    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      migrationHistorySectionList: [],
    });

    const fetchMigrationHistory = (databaseList: Database[]) => {
      state.migrationHistorySectionList = [];
      for (const database of databaseList) {
        store
          .dispatch("instance/checkMigrationSetup", database.instance.id)
          .then((migration: InstanceMigration) => {
            if (migration.status == "OK") {
              store
                .dispatch("instance/fetchMigrationHistory", {
                  instanceId: database.instance.id,
                  databaseName: database.name,
                  limit: MAX_MIGRAION_HISTORY_COUNT,
                })
                .then((migrationHistoryList: MigrationHistory[]) => {
                  if (migrationHistoryList.length > 0) {
                    const title = `${database.name} (${database.instance.environment.name})`;
                    const index = state.migrationHistorySectionList.findIndex(
                      (item: BBTableSectionDataSource<MigrationHistory>) => {
                        return item.title == title;
                      }
                    );
                    const newItem = {
                      title: title,
                      link: fullDatabasePath(database),
                      list: migrationHistoryList,
                    };
                    if (index >= 0) {
                      state.migrationHistorySectionList[index] = newItem;
                    } else {
                      state.migrationHistorySectionList.push(newItem);
                    }
                  }
                });
            }
          });
      }
    };

    const prepareMigrationHistoryList = () => {
      fetchMigrationHistory(props.databaseList);
    };
    watchEffect(prepareMigrationHistoryList);

    return {
      searchField,
      state,
    };
  },
};
</script>
