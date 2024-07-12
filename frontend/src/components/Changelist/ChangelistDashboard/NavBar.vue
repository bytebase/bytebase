<template>
  <div class="flex flex-row items-center justify-between">
    <div class="flex items-center justify-start">
      <ProjectSelect
        v-if="!disableProjectSelect"
        v-model:project-name="projectName"
        :include-all="true"
      />
    </div>
    <div class="flex items-center justify-end gap-x-2">
      <SearchBox
        v-model:value="filter.keyword"
        :placeholder="$t('common.filter-by-name')"
      />
      <NButton
        v-if="allowCreate"
        type="primary"
        @click="showCreatePanel = true"
      >
        <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
        {{ $t("changelist.new") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, watch } from "vue";
import { ProjectSelect } from "@/components/v2";
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
