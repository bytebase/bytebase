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
              size="small"
              style="--n-padding: 0 4px"
              @click="$router.push({ name: INSTANCE_ROUTE_DASHBOARD })"
            >
              <template #icon>
                <SettingsIcon class="w-4" />
              </template>
            </NButton>
          </template>
          <template #default>
            {{ $t("sql-editor.manage-connections") }}
          </template>
        </NTooltip>
      </template>
      <ConnectionPane :show="show" />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { useWindowSize } from "@vueuse/core";
import { SettingsIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { hasWorkspacePermissionV2 } from "@/utils";
import ConnectionPane from "./ConnectionPane";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "update:show", show: boolean): void;
}>();

const { width: winWidth } = useWindowSize();
const contentWidth = computed(() => {
  if (winWidth.value >= 800) {
    return "50vw";
  }
  return "calc(100vw - 4rem)";
});

const allowManageInstance = computed(() => {
  return hasWorkspacePermissionV2("bb.instances.list");
});
</script>

<style scoped lang="postcss">
.connection-panel-content :deep(.n-drawer-header__main) {
  flex: 1 1 0%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
