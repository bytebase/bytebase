<template>
  <nav
    class="flex-1 flex flex-col overflow-y-hidden border-r border-block-border"
  >
    <BytebaseLogo
      v-if="showLogo"
      class="p-2 shrink-0 m-auto"
      :redirect="logoRedirect"
    />
    <slot name="prefix" />
    <NScrollbar>
      <div class="flex-1 px-2.5 flex flex-col gap-y-1">
        <div v-for="(item, i) in filteredSidebarList" :key="i">
          <router-link
            v-if="item.type === 'route'"
            :to="{ path: item.path, name: item.name }"
            :class="[parentRouteClass, getItemClass(item)]"
          >
            <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
            {{ item.title }}
          </router-link>
          <div
            v-else-if="item.type === 'div'"
            :class="[parentRouteClass, getItemClass(item)]"
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
            :class="[parentRouteClass, getItemClass(item)]"
            :href="item.path"
            @click="$emit('select', item, $event)"
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
            class="flex flex-col gap-y-1 mt-1"
          >
            <template v-for="(child, j) in item.children" :key="`${i}-${j}`">
              <a
                v-if="child.type === 'link'"
                :class="[childRouteClass, getItemClass(child)]"
                :href="child.path"
                @click="$emit('select', child, $event)"
              >
                {{ child.title }}
              </a>
              <router-link
                v-else-if="child.type === 'route'"
                :to="{ path: child.path, name: child.name }"
                :class="[childRouteClass, getItemClass(child)]"
              >
                {{ child.title }}
              </router-link>
              <div
                v-else-if="child.type === 'div'"
                :class="[childRouteClass, getItemClass(child)]"
                @click="onClick(child, `${i}-${j}`, $event)"
              >
                {{ child.title }}
              </div>
            </template>
          </div>
        </div>
      </div>
    </NScrollbar>
  </nav>
</template>

<script setup lang="ts">
import { ChevronDown, ChevronRight } from "lucide-vue-next";
import { NScrollbar } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useRoute } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import type { SidebarItem } from "./type";

interface LocalState {
  expandedSidebar: Set<string>;
}

const props = withDefaults(
  defineProps<{
    itemList: SidebarItem[];
    showLogo?: boolean;
    logoRedirect?: string;
    getItemClass?: (item: SidebarItem) => string[];
  }>(),
  {
    showLogo: true,
    logoRedirect: "",
    getItemClass: (_: SidebarItem) => [],
  }
);

const emit = defineEmits<{
  (event: "select", item: SidebarItem, e: MouseEvent): void;
}>();

const state = reactive<LocalState>({
  expandedSidebar: new Set(),
});

// Track groups auto-expanded by route so we can collapse them on navigation
// without collapsing groups the user manually opened.
const autoExpanded = new Set<string>();
// Track groups the user has manually toggled — never override these.
const manualToggled = new Set<string>();

const parentRouteClass = computed(() => {
  return "group flex items-center px-2 py-1.5 leading-normal font-medium rounded-md text-gray-700 outline-item text-sm!";
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
      if (item.hide) {
        return false;
      }
      if (item.children.length > 0) {
        return true;
      }
      if (item.type === "div" || item.type === "link") {
        return !!item.path;
      }
      return !!item.path || !!item.name;
    });
});

const currentRoute = useRoute();

const expandForActiveRoute = () => {
  const currentName = currentRoute.name?.toString() ?? "";
  // Remove previous auto-expansions (preserve manual ones)
  for (const key of autoExpanded) {
    state.expandedSidebar.delete(key);
  }
  autoExpanded.clear();
  for (let i = 0; i < filteredSidebarList.value.length; i++) {
    const item = filteredSidebarList.value[i];
    const key = `${i}`;
    if (item.children.length === 0) continue;
    // Never override groups the user has manually toggled.
    if (manualToggled.has(key)) continue;
    if (item.expand) {
      state.expandedSidebar.add(key);
      continue;
    }
    const hasActiveChild = item.children.some(
      (child) =>
        child.name === currentName || currentName.startsWith(`${child.name}.`)
    );
    if (hasActiveChild && !state.expandedSidebar.has(key)) {
      state.expandedSidebar.add(key);
      autoExpanded.add(key);
    }
  }
};

watch(
  () => filteredSidebarList.value,
  () => {
    state.expandedSidebar.clear();
    manualToggled.clear();
    expandForActiveRoute();
  },
  { immediate: true }
);

watch(
  () => currentRoute.name,
  () => {
    expandForActiveRoute();
  }
);

const onClick = (sidebar: SidebarItem, key: string, e: MouseEvent) => {
  if (sidebar.path) {
    return emit("select", sidebar, e);
  }
  // Mark as manually toggled so route changes won't fight user intent.
  manualToggled.add(key);
  autoExpanded.delete(key);
  if (state.expandedSidebar.has(key)) {
    state.expandedSidebar.delete(key);
  } else {
    state.expandedSidebar.add(key);
  }
};
</script>
