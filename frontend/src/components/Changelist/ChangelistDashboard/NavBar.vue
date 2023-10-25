<template>
  <div class="flex flex-row items-center justify-between py-0.5">
    <div class="flex items-center justify-start">
      <ProjectSelect
        v-if="!disableProjectSelect"
        v-model:project="projectUID"
        :include-all="true"
      />
    </div>
    <div class="flex items-center justify-end gap-x-2">
      <SearchBox
        v-model:value="filter.keyword"
        :placeholder="$t('common.filter-by-name')"
      />
      <NButton type="primary" @click="showCreatePanel = true">
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
import { useProjectV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { useChangelistDashboardContext } from "./context";

const { filter, showCreatePanel, events } = useChangelistDashboardContext();

const projectUID = computed({
  get() {
    const { project } = filter.value;
    if (project === "projects/-") return String(UNKNOWN_ID);
    return useProjectV1Store().getProjectByName(project).uid;
  },
  set(uid) {
    if (!uid || uid === String(UNKNOWN_ID)) {
      filter.value.project = "projects/-";
    } else {
      filter.value.project = useProjectV1Store().getProjectByUID(uid).name;
    }
  },
});

watch(projectUID, () => events.emit("refresh"));
</script>
