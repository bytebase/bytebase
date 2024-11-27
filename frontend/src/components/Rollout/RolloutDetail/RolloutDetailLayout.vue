<template>
  <router-view v-if="rollout.name !== unknownRollout().name" v-bind="$attrs" />
  <div v-else class="flex justify-center items-center py-10">
    <BBSpin />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";
import { BBSpin } from "@/bbkit";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { unknownRollout } from "@/types";
import { provideRolloutDetailContext } from "./context";

const route = useRoute();

const rolloutName = computed(
  () =>
    `${projectNamePrefix}${route.params.projectId}/rollouts/${route.params.rolloutId}`
);

const { rollout } = provideRolloutDetailContext(rolloutName.value);
</script>
