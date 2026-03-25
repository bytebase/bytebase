<template>
  <div v-if="workspaceStore.workspaceList.length > 1" class="px-2.5 pb-2">
    <NPopselect
      :value="currentWorkspaceName"
      :options="workspaceOptions"
      trigger="click"
      @update:value="onSwitch"
    >
      <button
        class="w-full flex items-center gap-x-2 px-2 py-1.5 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-100 cursor-pointer"
      >
        <Building2 class="w-4 h-4 text-gray-500 shrink-0" />
        <span class="truncate flex-1 text-left">
          {{ workspaceStore.currentWorkspace?.title }}
        </span>
        <ChevronsUpDown class="w-3.5 h-3.5 text-gray-400 shrink-0" />
      </button>
    </NPopselect>
  </div>
</template>

<script lang="ts" setup>
import { Building2, ChevronsUpDown } from "lucide-vue-next";
import { NPopselect } from "naive-ui";
import { computed } from "vue";
import { useWorkspaceV1Store } from "@/store";

const workspaceStore = useWorkspaceV1Store();

const currentWorkspaceName = computed(
  () => workspaceStore.currentWorkspace?.name ?? ""
);

const workspaceOptions = computed(() =>
  workspaceStore.workspaceList.map((ws) => ({
    label: ws.title,
    value: ws.name,
  }))
);

const onSwitch = (workspaceName: string) => {
  if (workspaceName === currentWorkspaceName.value) return;
  workspaceStore.switchWorkspace(workspaceName);
};
</script>
