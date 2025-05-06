<template>
  <Drawer
    :show="show"
    :close-on-esc="false"
    placement="right"
    style="--n-body-padding: 4px 0"
    @update:show="$emit('update:show', $event)"
  >
    <DrawerContent
      :style="{
        width: contentWidth,
        maxWidth: '800px',
      }"
      class="connection-panel-content"
    >
      <template #header>
        <span>{{ $t("common.connection") }}</span>
        <NTooltip v-if="allowManageInstance" placement="bottom">
          <template #trigger>
            <NButton
              quaternary
              size="tiny"
              style="--n-padding: 0 4px"
              @click="
                $router.push({ name: SQL_EDITOR_SETTING_INSTANCE_MODULE })
              "
            >
              <template #icon>
                <SettingsIcon class="w-4 h-4" />
              </template>
            </NButton>
          </template>
          <template #default>
            {{ $t("sql-editor.manage-connections") }}
          </template>
        </NTooltip>
      </template>
      <ConnectionPane />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { useWindowSize } from "@vueuse/core";
import { SettingsIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { SQL_EDITOR_SETTING_INSTANCE_MODULE } from "@/router/sqlEditor";
import { useSidebarItems as useSettingItems } from "../Setting/Sidebar";
import ConnectionPane from "./ConnectionPane";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "update:show", show: boolean): void;
}>();

const { width: winWidth } = useWindowSize();
const { itemList: settingItemList } = useSettingItems();
const contentWidth = computed(() => {
  if (winWidth.value >= 800) {
    return "50vw";
  }
  return "calc(100vw - 4rem)";
});

const allowManageInstance = computed(() => {
  return (
    settingItemList.value.findIndex(
      (item) => item.name === SQL_EDITOR_SETTING_INSTANCE_MODULE
    ) >= 0
  );
});
</script>

<style scoped lang="postcss">
.connection-panel-content :deep(.n-drawer-header__main) {
  @apply flex-1 flex items-center justify-between;
}
</style>
