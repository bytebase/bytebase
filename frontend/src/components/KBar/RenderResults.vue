<template>
  <KBarResults
    :items="matches.results"
    :item-height="itemHeight"
    class="max-h-72"
  >
    <template #item="{ item, index, active }">
      <div v-if="typeof item === 'string'" class="section" :data-index="index">
        {{ item }}
      </div>
      <div v-else class="item" :class="{ active }" :data-index="index">
        <div class="content">
          <div class="main">
            <span v-if="isDeepAction(item)" class="parent">
              {{ findParent(item).name }}
            </span>

            <span class="name">{{ item.name }}</span>

            <span v-for="(tag, i) in item.data?.tags" :key="i" class="tag">
              {{ tag }}
            </span>
          </div>
          <span v-if="item.subtitle" class="subtitle">
            {{ item.subtitle }}
          </span>
        </div>

        <div v-if="item.shortcut?.length > 0" aria-hidden class="shortcut">
          <kbd v-for="(sc, j) in item.shortcut" :key="j">{{ sc }}</kbd>
        </div>
      </div>
    </template>
  </KBarResults>
</template>

<script lang="ts">
import {
  useKBarMatches,
  KBarResults,
  ActionImpl,
  useKBarState,
} from "@bytebase/vue-kbar";
import { defineComponent } from "vue";

export default defineComponent({
  name: "RenderResults",
  components: { KBarResults },
  setup() {
    const state = useKBarState();
    const matches = useKBarMatches();
    const itemHeight = (params: { item: any; index: number }) => {
      if (typeof params.item === "string") return 32;
      return 48;
    };

    const isDeepAction = (action: ActionImpl): boolean => {
      return (
        !!action.parent && action.parent !== state.value.currentRootActionId
      );
    };

    const findParent = (child: ActionImpl): ActionImpl => {
      return state.value.actions.find((action) => action.id === child.parent)!;
    };

    return { matches, itemHeight, isDeepAction, findParent };
  },
});
</script>

<style scoped>
.section {
  @apply h-8 px-4 text-xs uppercase text-gray-500 flex items-center bg-gray-50;
}
.item {
  @apply h-12 box-border px-3 py-4 text-lg bg-transparent border-l-4 border-transparent cursor-pointer flex justify-between items-center gap-2;
}
.item.active {
  @apply bg-control-bg-hover border-current;
}
.content {
  @apply flex flex-col overflow-x-hidden;
}
.main {
  @apply flex items-center text-base gap-1;
}
.tag {
  @apply inline-block text-xs px-1 py-0.5 bg-black bg-opacity-10 rounded-sm;
}
.name {
  @apply overflow-x-hidden overflow-ellipsis whitespace-nowrap;
}
.subtitle {
  @apply text-xs text-gray-500 overflow-x-hidden overflow-ellipsis whitespace-nowrap;
}
.parent {
  @apply text-gray-500;
}
.parent::after {
  content: "›";
  @apply text-gray-500 mx-1;
}
.shortcut {
  @apply grid grid-flow-col gap-1 justify-self-end text-gray-500;
}
.shortcut kbd {
  @apply w-6 h-6 flex items-center justify-center bg-black bg-opacity-10 rounded text-sm;
}
</style>
