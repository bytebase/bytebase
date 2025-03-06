<template>
  <div v-if="available">
    <NButton text type="primary" :size="size" @click="onClick">
      {{ $t("sql-editor.request-query") }}
    </NButton>

    <GrantRequestPanel
      v-if="showPanel"
      :project-name="database.project"
      :database-resource="databaseResource"
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
  isValidDatabaseName,
  PresetRoleType,
  type DatabaseResource,
} from "@/types";
import { hasPermissionToCreateRequestGrantIssue } from "@/utils";

const props = withDefaults(
  defineProps<{
    databaseResource: DatabaseResource;
    size?: "tiny" | "medium";
  }>(),
  {
    size: "medium",
  }
);

const showPanel = ref(false);

const { database } = useDatabaseV1ByName(
  computed(() => props.databaseResource.databaseFullName)
);

const available = computed(() => {
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
