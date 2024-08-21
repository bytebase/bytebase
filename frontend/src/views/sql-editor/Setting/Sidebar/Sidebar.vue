<template>
  <div class="h-full flex flex-col overflow-y-auto bg-control-bg">
    <nav class="flex-1 flex flex-col overflow-y-hidden text-sm">
      <BytebaseLogo
        class="w-full px-4 shrink-0"
        :redirect="SQL_EDITOR_HOME_MODULE"
      />

      <div class="px-2.5 mb-2">
        <router-link
          class="group flex items-center gap-2 px-2 py-1.5 leading-normal font-medium rounded-md text-main outline-item !text-base"
          :to="{ name: SQL_EDITOR_HOME_MODULE }"
        >
          <ChevronLeftIcon class="w-5 h-5" />
          <span>{{ $t("common.setting") }}</span>
        </router-link>
      </div>

      <div class="flex-1 overflow-y-auto px-2.5 flex flex-col gap-1">
        <div v-for="item in itemList" :key="item.name">
          <router-link
            :to="{ path: item.path, name: item.name }"
            class="group flex items-center gap-2 px-2 py-1.5 leading-normal font-medium rounded-md text-gray-700 outline-item"
          >
            <component :is="item.icon" class="w-5 h-5 text-gray-500" />
            {{ item.title }}
          </router-link>
        </div>
      </div>
    </nav>
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
