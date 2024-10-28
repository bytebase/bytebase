<template>
  <div
    v-if="isValidReleaseName(release.name) && ready"
    class="w-full text-sm text-control-light flex space-x-1 items-center"
  >
    <PackageIcon class="w-5 h-auto shrink-0" />
    <a :href="`/${release.name}`" target="_blank" class="normal-link truncate">
      {{ release.title }}
    </a>
    <NTag
      v-if="release.state === State.DELETED"
      class="shrink-0"
      type="warning"
      size="small"
      round
    >
      {{ $t("common.archived") }}
    </NTag>
  </div>
</template>

<script setup lang="ts">
import { PackageIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import { useReleaseByName } from "@/store";
import { isValidReleaseName } from "@/types";
import { State } from "@/types/proto/v1/common";

const { issue } = useIssueContext();

const { release, ready } = useReleaseByName(
  issue.value.planEntity?.releaseSource?.release || ""
);

defineExpose({
  shown: computed(() => isValidReleaseName(release.value.name)),
});
</script>
