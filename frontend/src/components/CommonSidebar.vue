<template>
  <nav class="flex-1 flex flex-col overflow-y-hidden">
    <BytebaseLogo class="w-full px-4 shrink-0" />
    <div class="space-y-1 flex-1 overflow-y-auto px-2 pb-4">
      <button
        v-if="showGoBack"
        class="group shrink-0 flex items-center px-2 py-2 text-base leading-5 font-normal rounded-md text-gray-700 hover:opacity-80 focus:outline-none"
        @click.prevent="goBack"
      >
        <heroicons-outline:chevron-left
          class="mr-1 w-5 h-auto text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
        />
        {{ $t("common.back") }}
      </button>

      <div v-for="(item, i) in filteredSidebarList" :key="i">
        <router-link
          v-if="type === 'route' && item.path"
          :to="item.path"
          class="outline-item group w-full flex items-center pl-11 pr-2 py-1.5"
          :class="getItemClass(item.path)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
        </router-link>
        <div
          v-else
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
          :class="getItemClass(item.path)"
          @click="$emit('select', item.path)"
        >
          <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
          {{ item.title }}
        </div>
        <div v-if="item.children" class="space-y-1">
          <template v-for="(child, j) in item.children" :key="`${i}-${j}`">
            <div
              v-if="type === 'div'"
              class="group w-full flex items-center pl-11 pr-2 py-1.5 rounded-md"
              :class="getItemClass(child.path)"
              @click="$emit('select', child.path)"
            >
              {{ child.title }}
            </div>
            <router-link
              v-else
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
import { computed, VNode } from "vue";
import { useRouter } from "vue-router";
import { useRouterStore } from "@/store";

export interface SidebarItem {
  title: string;
  path?: string;
  icon: VNode;
  hide?: boolean;
  children?: {
    title: string;
    path: string;
    hide?: boolean;
  }[];
}

const props = withDefaults(
  defineProps<{
    itemList: SidebarItem[];
    type: "route" | "div";
    showGoBack?: boolean;
    getItemClass: (path: string | undefined) => string[];
  }>(),
  {
    showGoBack: false,
    getItemClass: (_: string | undefined) => [],
  }
);

defineEmits<{
  (event: "select", path: string | undefined): void;
}>();

const routerStore = useRouterStore();
const router = useRouter();

const goBack = () => {
  router.push(routerStore.backPath());
};

const filteredSidebarList = computed(() => {
  return props.itemList
    .map((item) => ({
      ...item,
      children: (item.children ?? []).filter((child) => !child.hide),
    }))
    .filter((item) => !item.hide && (!!item.path || item.children.length > 0));
});
</script>
