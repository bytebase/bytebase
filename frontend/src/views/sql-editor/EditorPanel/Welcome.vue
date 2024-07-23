<template>
  <div
    class="w-full h-auto flex-grow flex flex-col items-center justify-center"
  >
    <BytebaseLogo v-if="!hideLogo" component="span" class="mb-4 -mt-[10vh]" />

    <div class="flex flex-col items-start gap-y-2 w-max">
      <NButton
        type="primary"
        class="!w-full !justify-start"
        @click="changeConnection"
      >
        <template #icon>
          <LinkIcon class="w-4 h-4" />
        </template>
        {{ $t("sql-editor.connect-to-a-database") }}
      </NButton>
      <NButton
        v-if="showCreateInstanceButton"
        type="default"
        class="!w-full !justify-start"
        @click="gotoInstanceCreatePage"
      >
        <template #icon>
          <LayersIcon class="w-4 h-4" />
        </template>
        {{ $t("sql-editor.add-a-new-instance") }}
      </NButton>
      <NButton
        type="default"
        class="!w-full !justify-start"
        @click="createNewWorksheet"
      >
        <template #icon>
          <SquarePenIcon class="w-4 h-4" />
        </template>
        {{ $t("sql-editor.create-a-worksheet") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LayersIcon, LinkIcon, SquarePenIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { SQL_EDITOR_SETTING_INSTANCE_MODULE } from "@/router/sqlEditor";
import { useAppFeature, useSQLEditorTabStore } from "@/store";
import { useSidebarItems as useSettingItems } from "../Setting/Sidebar";
import { useSQLEditorContext } from "../context";

const { showConnectionPanel } = useSQLEditorContext();
const { itemList: settingItemList } = useSettingItems();
const router = useRouter();
const hideLogo = useAppFeature("bb.feature.sql-editor.hide-bytebase-logo");

const showCreateInstanceButton = computed(() => {
  return (
    settingItemList.value.findIndex(
      (item) => item.name === SQL_EDITOR_SETTING_INSTANCE_MODULE
    ) >= 0
  );
});

const changeConnection = () => {
  showConnectionPanel.value = true;
};

const createNewWorksheet = () => {
  useSQLEditorTabStore().addTab();
};

const gotoInstanceCreatePage = () => {
  if (
    settingItemList.value.findIndex(
      (item) => item.name === SQL_EDITOR_SETTING_INSTANCE_MODULE
    ) < 0
  ) {
    return;
  }
  router.push({
    name: SQL_EDITOR_SETTING_INSTANCE_MODULE,
    hash: `#add`,
  });
};
</script>
