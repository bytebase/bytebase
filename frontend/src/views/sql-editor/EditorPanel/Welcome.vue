<template>
  <div
    class="w-full h-auto flex-grow flex flex-col items-center justify-center"
  >
    <HideInStandaloneMode>
      <BytebaseLogo component="span" class="mb-4 -mt-[20%]" />
    </HideInStandaloneMode>
    <div class="flex flex-col items-start gap-y-1 w-auto">
      <NButton type="primary" @click="changeConnection">
        <template #icon>
          <LinkIcon class="w-4 h-4" />
        </template>
        {{ $t("sql-editor.connect-to-a-database") }}
      </NButton>
      <NButton quaternary @click="createNewWorksheet">
        <template #icon>
          <SquarePenIcon class="w-4 h-4" />
        </template>
        {{ $t("sql-editor.create-a-worksheet") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LinkIcon, SquarePenIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { useSQLEditorTabStore } from "@/store";
import { useSQLEditorContext } from "../context";

const { showConnectionPanel } = useSQLEditorContext();

const changeConnection = () => {
  showConnectionPanel.value = true;
};

const createNewWorksheet = () => {
  useSQLEditorTabStore().addTab();
};
</script>
