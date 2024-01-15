<template>
  <nav class="flex-1 flex flex-col overflow-y-hidden">
    <BytebaseLogo v-if="showLogo" class="w-full px-4 shrink-0" />
    <div class="flex-1 overflow-y-auto px-2.5 space-y-1">
      <div v-for="(item, i) in filteredSidebarList" :key="i">
        <router-link
          v-if="item.type === 'route'"
          :to="item.path ?? ''"
          :class="[parentRouteClass, getItemClass(item.path)]"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
        </router-link>
        <div
          v-else-if="item.type === 'div'"
          :class="[parentRouteClass, getItemClass(item.path)]"
          @click="onClick(item, `${i}`, $event)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
          <div v-if="item.children.length > 0" class="ml-auto text-gray-500">
            <ChevronRight
              v-if="!state.expandedSidebar.has(`${i}`)"
              class="w-4 h-4"
            />
            <ChevronDown v-else class="w-4 h-4" />
          </div>
        </div>
        <a
          v-if="item.type === 'link'"
          :class="[parentRouteClass, getItemClass(item.path)]"
          :href="item.path"
          @click="$emit('select', item.path, $event)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
        </a>
        <div
          v-else-if="item.type === 'divider'"
          class="border-t border-gray-300 my-2.5 mr-4 ml-2"
        />
        <div
          v-if="item.children.length > 0 && state.expandedSidebar.has(`${i}`)"
          class="space-y-1 mt-1"
        >
          <template v-for="(child, j) in item.children" :key="`${i}-${j}`">
            <a
              v-if="child.type === 'link'"
              :class="[childRouteClass, getItemClass(child.path)]"
              :href="child.path"
              @click="$emit('select', child.path, $event)"
            >
              {{ child.title }}
            </a>
            <router-link
              v-else-if="child.type === 'route' && child.path"
              :to="child.path"
              :class="[childRouteClass, getItemClass(child.path)]"
            >
              {{ child.title }}
            </router-link>
            <div
              v-else-if="child.type === 'div'"
              :class="[childRouteClass, getItemClass(child.path)]"
              @click="onClick(child, `${i}-${j}`, $event)"
            >
              {{ child.title }}
            </div>
          </template>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ChevronDown, ChevronRight } from "lucide-vue-next";
import { computed, VNode, reactive, watch } from "vue";

export interface SidebarItem {
  title?: string;
  path?: string;
  icon?: VNode;
  hide?: boolean;
  type: "route" | "div" | "divider" | "link";
  children?: SidebarItem[];
}

interface LocalState {
  expandedSidebar: Set<string>;
}

const props = withDefaults(
  defineProps<{
    itemList: SidebarItem[];
    showLogo?: boolean;
    getItemClass: (path: string | undefined) => string[];
  }>(),
  {
    showLogo: true,
    getItemClass: (_: string | undefined) => [],
  }
);

const emit = defineEmits<{
  (event: "select", path: string | undefined, e: MouseEvent): void;
}>();

const state = reactive<LocalState>({
  expandedSidebar: new Set(),
});

const parentRouteClass = computed(() => {
  return "group flex items-center px-2 py-1.5 leading-normal font-medium rounded-md text-gray-700 outline-item !text-sm";
});

const childRouteClass = computed(() => {
  return "group w-full flex items-center pl-9 pr-2 py-1 outline-item mb-0.5 rounded-md";
});

const filteredSidebarList = computed(() => {
  return props.itemList
    .map((item) => ({
      ...item,
      children: (item.children ?? []).filter((child) => !child.hide),
    }))
    .filter((item) => {
      if (item.type === "divider") {
        return true;
      }
      return !item.hide && (!!item.path || item.children.length > 0);
    });
});

watch(
  () => filteredSidebarList.value,
  () => {
    state.expandedSidebar.clear();
    for (let i = 0; i < filteredSidebarList.value.length; i++) {
      if (filteredSidebarList.value[i].children.length > 0) {
        state.expandedSidebar.add(`${i}`);
      }
    }
  },
  { immediate: true }
);

const onClick = (sidebar: SidebarItem, key: string, e: MouseEvent) => {
  if (sidebar.path) {
    return emit("select", sidebar.path, e);
  }
  if (state.expandedSidebar.has(key)) {
    state.expandedSidebar.delete(key);
  } else {
    state.expandedSidebar.add(key);
  }
};
</script>
