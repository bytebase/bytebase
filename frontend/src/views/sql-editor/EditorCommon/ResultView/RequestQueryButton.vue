<template>
  <div v-if="available">
    <NButton text type="primary" @click="showPanel = true">
      {{ $t("sql-editor.request-query-permission") }}
    </NButton>

    <RequestQueryPanel
      :show="showPanel"
      :project-id="database?.projectEntity.uid"
      :database="database"
      @close="showPanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import RequestQueryPanel from "@/components/Issue/panel/RequestQueryPanel/index.vue";
import { useDatabaseV1Store, useTabStore } from "@/store";
import { UNKNOWN_ID, unknownDatabase } from "@/types";

const tabStore = useTabStore();
const connection = computed(() => tabStore.currentTab.connection);
const showPanel = ref(false);

const database = computed(() => {
  const { databaseId } = connection.value;
  if (databaseId !== String(UNKNOWN_ID)) {
    return useDatabaseV1Store().getDatabaseByUID(databaseId);
  }
  return unknownDatabase();
});

const available = computed(() => {
  return database.value.uid !== String(UNKNOWN_ID);
});
</script>
