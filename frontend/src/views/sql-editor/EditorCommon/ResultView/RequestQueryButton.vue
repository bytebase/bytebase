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
import { useCurrentUserV1, useDatabaseV1Store, useTabStore } from "@/store";
import { UNKNOWN_ID, unknownDatabase } from "@/types";
import { hasPermissionToCreateRequestGrantIssue } from "@/utils";

const tabStore = useTabStore();
const me = useCurrentUserV1();
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
  if (database.value.uid === String(UNKNOWN_ID)) {
    return false;
  }

  return hasPermissionToCreateRequestGrantIssue(database.value, me.value);
});
</script>
