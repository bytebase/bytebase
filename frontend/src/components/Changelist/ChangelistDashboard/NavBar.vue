<template>
  <div class="flex flex-row items-center gap-x-2">
    <div v-if="!disableProjectSelect" class="flex items-center justify-start">
      <ProjectSelect v-model:project-name="projectName" :include-all="true" />
    </div>
    <SearchBox
      v-model:value="filter.keyword"
      style="max-width: 100%"
      :placeholder="$t('common.filter-by-name')"
    />
    <NButton v-if="allowCreate" type="primary" @click="showCreatePanel = true">
      <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
      {{ $t("changelist.new") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, watch } from "vue";
import { ProjectSelect, SearchBox } from "@/components/v2";
import { UNKNOWN_PROJECT_NAME, isValidProjectName } from "@/types";
import { useChangelistDashboardContext } from "./context";

defineProps<{
  disableProjectSelect?: boolean;
  allowCreate: boolean;
}>();

const { filter, showCreatePanel, events } = useChangelistDashboardContext();

const projectName = computed({
  get() {
    const { project } = filter.value;
    if (project === "projects/-") return UNKNOWN_PROJECT_NAME;
    return project;
  },
  set(name) {
    if (!isValidProjectName(name)) {
      filter.value.project = "projects/-";
    } else {
      filter.value.project = name;
    }
  },
});

watch(projectName, () => events.emit("refresh"));
</script>
