<template>
  <div
    v-if="isValidReleaseName(release.name) && ready"
    class="w-full text-sm text-control-light flex space-x-1 items-center"
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
      {{ release.title }}
    </a>
  </div>
</template>

<script setup lang="ts">
import { PackageIcon } from "lucide-vue-next";
import { useIssueContext } from "@/components/IssueV1";
import { useReleaseByName } from "@/store";
import { isValidReleaseName } from "@/types";
import { State } from "@/types/proto/v1/common";

const { issue } = useIssueContext();

const { release, ready } = useReleaseByName(
  (() => {
    return issue.value.planEntity?.specs.find(spec => 
      spec.changeDatabaseConfig?.release
    )?.changeDatabaseConfig?.release || "";
  })()
);
</script>
