<template>
  <div class="flex flex-col items-end">
    <CreateButton v-if="actionType === 'CREATE'" />
  </div>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { usePlanContext } from "@/components/Plan/logic";
import { CreateButton } from "./create";

type ActionType = "CREATE";

const { isCreating } = usePlanContext();

const actionType = asyncComputed(async (): Promise<ActionType | undefined> => {
  if (isCreating.value) {
    return "CREATE";
  }
}, undefined);
</script>
