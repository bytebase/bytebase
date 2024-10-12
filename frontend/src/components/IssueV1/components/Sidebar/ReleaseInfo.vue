<template>
  <div
    v-if="isValidReleaseName(release.name) && ready"
    class="text-sm text-control-light flex space-x-1 items-center"
  >
    <PackageIcon class="w-5 h-auto" />
    <EllipsisText>
      <a :href="`/${release.name}`" target="_blank" class="normal-link">
        <span>{{ release.title }}</span>
      </a>
    </EllipsisText>
  </div>
</template>

<script setup lang="ts">
import { PackageIcon } from "lucide-vue-next";
import { computed } from "vue";
import EllipsisText from "@/components/EllipsisText.vue";
import { useIssueContext } from "@/components/IssueV1";
import { useReleaseByName } from "@/store";
import { isValidReleaseName } from "@/types";

const { issue } = useIssueContext();

const { release, ready } = useReleaseByName(
  issue.value.planEntity?.releaseSource?.release || ""
);

defineExpose({
  shown: computed(() => isValidReleaseName(release.value.name)),
});
</script>
