<template>
  <nav class="flex-1 flex flex-col overflow-y-hidden">
    <BytebaseLogo v-if="showLogo" class="w-full px-4 shrink-0" />
    <div class="flex-1 overflow-y-auto px-2">
      <div v-for="(item, i) in filteredSidebarList" :key="i">
        <router-link
          v-if="item.type === 'route' && item.path"
          :to="item.path"
          class="outline-item group w-full font-medium flex items-center px-2 py-1.5 !text-base"
          :class="getItemClass(item.path)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
        </router-link>
        <div
          v-else-if="item.type === 'div'"
          class="group flex items-center px-2 py-1.5 leading-5 font-medium rounded-md text-gray-700 outline-item !text-base"
          :class="getItemClass(item.path)"
          @click="onClick(i)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
          <div v-if="item.children.length > 0" class="ml-auto text-gray-500">
            <ChevronRight
              v-if="!state.expandedSidebar.has(i)"
              class="w-4 h-4"
            />
            <ChevronDown v-else class="w-4 h-4" />
          </div>
        </div>
        <div
          v-else-if="item.type === 'divider'"
          class="border-t border-gray-300 my-2"
        />
        <div
          v-if="item.children.length > 0 && state.expandedSidebar.has(i)"
          class=""
        >
          <template v-for="(child, j) in item.children" :key="`${i}-${j}`">
            <div
              v-if="child.type === 'div'"
              class="group w-full flex items-center pl-11 pr-2 py-1.5 rounded-md outline-item"
              :class="getItemClass(child.path)"
              @click="$emit('select', child.path)"
            >
              {{ child.title }}
            </div>
            <router-link
              v-else-if="child.type === 'route'"
              :to="child.path"
              class="outline-item group w-full flex items-center pl-11 pr-2 py-1.5"
              :class="getItemClass(child.path)"
            >
              {{ child.title }}
            </router-link>
          </template>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { ChevronDown, ChevronRight } from "lucide-vue-next";
import { computed, VNode, reactive, onMounted } from "vue";

export interface SidebarItem {
  title?: string;
  path?: string;
  icon?: VNode;
  hide?: boolean;
  type: "route" | "div" | "divider";
  children?: {
    title: string;
    path: string;
    hide?: boolean;
    type: "route" | "div" | "divider";
  }[];
}

interface LocalState {
  expandedSidebar: Set<number>;
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
  (event: "select", path: string | undefined): void;
}>();

const state = reactive<LocalState>({
  expandedSidebar: new Set(),
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

onMounted(() => {
  state.expandedSidebar.clear();
  for (let i = 0; i < filteredSidebarList.value.length; i++) {
    if (filteredSidebarList.value[i].children.length > 0) {
      state.expandedSidebar.add(i);
    }
  }
});

const onClick = (index: number) => {
  const sidebar = filteredSidebarList.value[index];
  if (sidebar.path) {
    return emit("select", sidebar.path);
  }
  if (state.expandedSidebar.has(index)) {
    state.expandedSidebar.delete(index);
  } else {
    state.expandedSidebar.add(index);
  }
};
</script>
