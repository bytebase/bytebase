<template>
  <router-view v-bind="$attrs" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { provideRolloutDetailContext } from "./context";
import { usePollRollout } from "./poll";

const route = useRoute();

const rolloutName = computed(
  () =>
    `${projectNamePrefix}${route.params.projectId}/rollouts/${route.params.rolloutId}`
);

provideRolloutDetailContext(rolloutName.value);

usePollRollout(rolloutName.value);
</script>
