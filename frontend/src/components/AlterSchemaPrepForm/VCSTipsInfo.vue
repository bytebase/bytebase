<template>
  <template v-if="project">
    <div v-if="project.workflowType == 'VCS'" class="textlabel">
      {{ $t("alter-schema.vcs-enabled") }}
    </div>
  </template>
  <template v-else>
    <div class="flex flex-row space-x-2 align-top">
      <heroicons-outline:collection class="w-4 h-4 text-control mt-0.5" />
      <p class="textlabel flex-1">
        {{ $t("alter-schema.vcs-info") }}
      </p>
    </div>
  </template>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Project } from "../../types";

export default defineComponent({
  name: "VCSTipsInfo",
  props: {
    project: {
      type: Object as PropType<Project>,
      default: undefined,
    },
  },
});
</script>
