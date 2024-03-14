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
import { useConnectionOfCurrentSQLEditorTab, useCurrentUserV1 } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { hasPermissionToCreateRequestGrantIssue } from "@/utils";

const me = useCurrentUserV1();
const showPanel = ref(false);
const { database } = useConnectionOfCurrentSQLEditorTab();

const available = computed(() => {
  if (database.value.uid === String(UNKNOWN_ID)) {
    return false;
  }

  return hasPermissionToCreateRequestGrantIssue(database.value, me.value);
});
</script>
