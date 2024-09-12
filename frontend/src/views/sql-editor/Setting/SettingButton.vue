<template>
  <NButton v-if="show" @click="goSetting">
    <SettingsIcon class="w-4 h-4" />
  </NButton>
</template>

<script setup lang="ts">
import { head } from "lodash-es";
import { SettingsIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useSidebarItems } from "./Sidebar";

const router = useRouter();
const { itemList } = useSidebarItems();

const flattenItems = computed(() => {
  return itemList.value.flatMap((item) =>
    item.type === "div" ? (item.children ?? []) : item
  );
});
const firstItem = computed(() => {
  return head(
    flattenItems.value.filter(
      (item) => item.type === "route" || item.type === "link"
    )
  );
});

const show = computed(() => {
  return !!firstItem.value;
});

const goSetting = () => {
  if (!firstItem.value) return;
  if (firstItem.value.type === "link") {
    router.push({
      path: firstItem.value.path,
    });
    return;
  }
  if (firstItem.value.type === "route") {
    router.push({
      name: firstItem.value.name,
    });
  }
};
</script>
