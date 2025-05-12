<template>
  <div
    class="h-full flex flex-col items-stretch justify-between overflow-hidden text-sm p-1 shrink-0"
  >
    <div class="flex flex-col gap-y-1">
      <TabItem
        v-for="action in availableActions"
        :key="action.view"
        :action="action"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useCurrentTabViewStateContext } from "../../EditorPanel/context/viewState.tsx";
import TabItem from "./TabItem.vue";

const { availableActions } = useCurrentTabViewStateContext();
</script>
