<template>
  <div v-if="available">
    <NButton text type="primary" :size="size" @click="onClick">
      {{ $t("sql-editor.request-query") }}
    </NButton>

    <RequestQueryPanel
      :show="showPanel"
      :project-name="database.project"
      :database="database"
      :placement="panelPlacement"
      @close="showPanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import RequestQueryPanel from "@/components/Issue/panel/RequestQueryPanel/index.vue";
import { isValidDatabaseName, type ComposedDatabase } from "@/types";
import { hasPermissionToCreateRequestGrantIssue } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    panelPlacement: "left" | "right";
    size?: "tiny" | "medium";
  }>(),
  {
    panelPlacement: "right",
    size: "medium",
  }
);

const showPanel = ref(false);

const available = computed(() => {
  if (!isValidDatabaseName(props.database.name)) {
    return false;
  }

  return hasPermissionToCreateRequestGrantIssue(props.database);
});

const onClick = (e: MouseEvent) => {
  e.stopPropagation();
  e.preventDefault();
  showPanel.value = true;
};
</script>
