<template>
  <div
    v-if="isValidReleaseName(release.name) && ready"
    class="w-full text-sm text-control-light flex gap-x-1 items-center"
  >
    <PackageIcon class="w-5 h-auto shrink-0" />
    <a
      :href="`/${release.name}`"
      target="_blank"
      class="normal-link truncate"
      :class="{
        'line-through opacity-60': release.state === State.DELETED,
      }"
    >
      {{ release.title || release.name }}
    </a>
  </div>
</template>

<script setup lang="ts">
import { PackageIcon } from "lucide-vue-next";
import { computed } from "vue";
import { specForTask, useIssueContext } from "@/components/IssueV1";
import { useReleaseByName } from "@/store";
import { isValidReleaseName } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";

const { issue, selectedTask } = useIssueContext();

const releaseName = computed(() => {
  const spec = specForTask(issue.value.planEntity, selectedTask.value);
  if (spec?.config?.case === "changeDatabaseConfig") {
    return spec.config.value.release;
  }
  return "";
});

const { release, ready } = useReleaseByName(
  computed(() => releaseName.value || "")
);
</script>
