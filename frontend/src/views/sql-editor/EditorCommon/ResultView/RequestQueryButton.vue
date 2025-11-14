<template>
  <div v-if="available">
    <NButton text type="primary" :size="size" @click="onClick">
      {{ $t("sql-editor.request-query") }}
    </NButton>

    <GrantRequestPanel
      v-if="showPanel"
      :project-name="database.project"
      :database-resources="databaseResources"
      :placement="'right'"
      :role="PresetRoleType.SQL_EDITOR_USER"
      @close="showPanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import { useDatabaseV1ByName } from "@/store";
import {
  type DatabaseResource,
  isValidDatabaseName,
  PresetRoleType,
} from "@/types";
import { hasPermissionToCreateRequestGrantIssue } from "@/utils";

const props = withDefaults(
  defineProps<{
    databaseResources: DatabaseResource[];
    size?: "tiny" | "medium";
  }>(),
  {
    size: "medium",
  }
);

const showPanel = ref(false);

const primaryDatabase = computed(() => {
  return props.databaseResources[0];
});

const { database } = useDatabaseV1ByName(
  computed(() => primaryDatabase.value?.databaseFullName ?? "")
);

const available = computed(() => {
  if (props.databaseResources.length === 0) {
    return false;
  }

  if (!isValidDatabaseName(database.value.name)) {
    return false;
  }

  return hasPermissionToCreateRequestGrantIssue(database.value);
});

const onClick = (e: MouseEvent) => {
  e.stopPropagation();
  e.preventDefault();
  showPanel.value = true;
};
</script>
