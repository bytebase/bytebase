<template>
  <NButton v-if="show" @click="goSetting">
    <template #icon>
      <SettingsIcon />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { SettingsIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { SQL_EDITOR_SETTING_MODULE } from "@/router/sqlEditor";
import { useSidebarItems } from "./Sidebar";
import { NButton } from "naive-ui";

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
