<template>
  <Drawer
    v-model:show="show"
    :mask-closable="true"
    placement="right"
    class="w-screen! sm:w-[80vw]!"
  >
    <DrawerContent :title="$t('plan.select-targets')" closable>
      <div class="flex flex-col gap-y-4">
        <div class="flex-1 overflow-hidden">
          <DatabaseAndGroupSelector
            :project="project"
            :value="databaseSelectState"
            @update:value="handleUpdateSelection"
          />
        </div>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="show = false" quaternary>
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!hasSelection"
            @click="handleConfirm"
          >
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import DatabaseAndGroupSelector from "@/components/DatabaseAndGroupSelector";
import { Drawer, DrawerContent } from "@/components/v2";
import { useCurrentProjectV1 } from "@/store";

const props = defineProps<{
  currentTargets: string[];
}>();

const emit = defineEmits<{
  (event: "confirm", targets: string[]): void;
}>();

const { project } = useCurrentProjectV1();
const show = defineModel<boolean>("show", { default: false });

const databaseSelectState = reactive<DatabaseSelectState>({
  changeSource: "DATABASE",
  selectedDatabaseNameList: [],
});

const hasSelection = computed(() => {
  if (databaseSelectState.changeSource === "DATABASE") {
    return databaseSelectState.selectedDatabaseNameList.length > 0;
  } else {
    return !!databaseSelectState.selectedDatabaseGroup;
  }
});

// Initialize selection when drawer opens
watch(show, (newVal) => {
  if (newVal && props.currentTargets.length > 0) {
    // Check if current targets are database group or databases
    const firstTarget = props.currentTargets[0];
    if (firstTarget.includes("/databaseGroups/")) {
      // It's a database group
      databaseSelectState.changeSource = "GROUP";
      databaseSelectState.selectedDatabaseGroup = firstTarget;
      databaseSelectState.selectedDatabaseNameList = [];
    } else {
      // It's database(s)
      databaseSelectState.changeSource = "DATABASE";
      databaseSelectState.selectedDatabaseNameList = [...props.currentTargets];
      databaseSelectState.selectedDatabaseGroup = undefined;
    }
  } else if (newVal) {
    // Reset to default state if no current targets
    databaseSelectState.changeSource = "DATABASE";
    databaseSelectState.selectedDatabaseNameList = [];
    databaseSelectState.selectedDatabaseGroup = undefined;
  }
});

const handleUpdateSelection = (newState: DatabaseSelectState) => {
  Object.assign(databaseSelectState, newState);
};

const handleConfirm = () => {
  const targets: string[] = [];

  if (databaseSelectState.changeSource === "DATABASE") {
    targets.push(...databaseSelectState.selectedDatabaseNameList);
  } else if (databaseSelectState.selectedDatabaseGroup) {
    targets.push(databaseSelectState.selectedDatabaseGroup);
  }

  emit("confirm", targets);
  show.value = false;
};
</script>
