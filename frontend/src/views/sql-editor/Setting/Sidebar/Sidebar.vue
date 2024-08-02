<template>
  <div class="h-full flex flex-col overflow-y-auto bg-control-bg">
    <nav class="flex-1 flex flex-col overflow-y-hidden">
      <BytebaseLogo class="w-full px-4 shrink-0" />

      <div class="flex-1 overflow-y-auto px-2.5 space-y-1">
        <div v-for="item in itemList" :key="item.name">
          <router-link
            :to="{ path: item.path, name: item.name }"
            class="group flex items-center px-2 py-1.5 leading-normal font-medium rounded-md text-gray-700 outline-item !text-sm"
          >
            <component :is="item.icon" class="mr-2 w-5 h-5 text-gray-500" />
            {{ item.title }}
          </router-link>
        </div>
      </div>
    </nav>

    <router-link
      class="flex-shrink-0 flex gap-x-2 justify-start items-center border-t border-block-border px-3 py-2 hover:bg-control-bg-hover cursor-pointer text-sm"
      :to="{ name: SQL_EDITOR_HOME_MODULE }"
    >
      <ChevronLeftIcon class="w-5 h-5" />
      <span>{{ $t("common.back") }}</span>
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { head } from "lodash-es";
import { ChevronLeftIcon } from "lucide-vue-next";
import { watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import {
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_SETTING_MODULE,
} from "@/router/sqlEditor";
import { useSidebarItems } from "./common";

const route = useRoute();
const router = useRouter();
const { itemList } = useSidebarItems();

watch(
  () => route.name,
  (name) => {
    if (name === SQL_EDITOR_SETTING_MODULE) {
      const first = head(itemList.value);
      if (first) {
        router.replace({ name: first.name });
      } else {
        router.replace({ name: "error.404" });
      }
    }
  },
  { immediate: true }
);
</script>
