<template>
  <div
    class="w-full h-full flex flex-col gap-4 py-4 overflow-y-auto"
    v-bind="$attrs"
  >
    <DatabaseDashboard :on-click-database="handleClickDatabase" />
  </div>

  <Drawer v-model:show="state.detail.show">
    <DrawerContent
      :title="$t('common.detail')"
      style="width: calc(100vw - 4rem)"
    >
      <Detail v-if="state.detail.database" :database="state.detail.database" />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { onMounted, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE,
  SQL_EDITOR_SETTING_DATABASES_MODULE,
} from "@/router/sqlEditor";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName, type ComposedDatabase } from "@/types";
import { extractDatabaseResourceName } from "@/utils";
import DatabaseDashboard from "@/views/DatabaseDashboard.vue";
import Detail from "./Detail.vue";

defineOptions({
  inheritAttrs: false,
});

type LocalState = {
  detail: {
    show: boolean;
    database?: ComposedDatabase;
  };
};

const route = useRoute();
const router = useRouter();
const state = reactive<LocalState>({
  detail: {
    show: false,
  },
});

const handleClickDatabase = (_: MouseEvent, db: ComposedDatabase) => {
  state.detail = {
    show: true,
    database: db,
  };
};

onMounted(async () => {
  const maybeDatabase = `instances/${route.params.instanceId || -1}/databases/${route.params.databaseName || -1}`;
  if (isValidDatabaseName(maybeDatabase)) {
    const db =
      await useDatabaseV1Store().getOrFetchDatabaseByName(maybeDatabase);
    if (db) {
      state.detail.show = true;
      state.detail.database = db;
    }
  }

  watch(
    [() => state.detail.show, () => state.detail.database?.name],
    ([show, database]) => {
      if (show && database) {
        const { instanceName: instanceId, databaseName } =
          extractDatabaseResourceName(database);
        router.replace({
          name: SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE,
          params: {
            instanceId,
            databaseName,
          },
        });
      } else {
        router.replace({
          name: SQL_EDITOR_SETTING_DATABASES_MODULE,
        });
      }
    }
  );
});
</script>
