<template>
  <NButton v-if="show" @click="goSetting">
    <SettingsIcon class="w-4 h-4" />
  </NButton>
</template>

<script setup lang="ts">
import { SettingsIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { SQL_EDITOR_SETTING_MODULE } from "@/router/sqlEditor";
import { useSidebarItems } from "./Sidebar";

const router = useRouter();
const { itemList } = useSidebarItems();

const show = computed(() => {
  return itemList.value.length > 0;
});

const goSetting = () => {
  router.push({
    name: SQL_EDITOR_SETTING_MODULE,
  });
};
</script>
