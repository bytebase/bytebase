<template>
  <NSelect
    v-model:value="selectedIsolation"
    class="w-36!"
    size="small"
    :options="options"
    :placeholder="$t('plan.select-isolation-level')"
    :filterable="true"
    :virtual-scroll="true"
    :fallback-option="false"
    :consistent-menu-width="false"
    :clearable="true"
    :disabled="!allowChange"
  />
</template>

<script setup lang="tsx">
import { NSelect, type SelectOption } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import {
  updateSpecSheetWithStatement,
  usePlanContext,
} from "@/components/Plan/logic";
import { setSheetStatement } from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";
import {
  type IsolationLevel,
  parseStatement,
  updateIsolationLevel,
} from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import { useIsolationLevelSettingContext } from "./context";

const {
  allowChange,
  isolationLevel: contextIsolationLevel,
  events,
} = useIsolationLevelSettingContext();

const { selectedSpec } = useSelectedSpec();
const { sheetStatement, sheet, sheetReady } = useSpecSheet(selectedSpec);
const { plan, isCreating } = usePlanContext();

// Flag to prevent circular updates
const isUpdatingFromUI = ref(false);

const selectedIsolation = computed({
  get: () => contextIsolationLevel.value,
  set: (value: IsolationLevel | null) => {
    isUpdatingFromUI.value = true;
    contextIsolationLevel.value = value || undefined;
    setIsolationInStatement(value || undefined);
    // Reset the flag after Vue's next tick to allow statement updates to propagate
    nextTick(() => {
      isUpdatingFromUI.value = false;
    });
  },
});

const options = computed(() => {
  const opts: SelectOption[] = [
    {
      value: "READ_UNCOMMITTED",
      label: "Read Uncommitted",
    },
    {
      value: "READ_COMMITTED",
      label: "Read Committed",
    },
    {
      value: "REPEATABLE_READ",
      label: "Repeatable Read",
    },
    {
      value: "SERIALIZABLE",
      label: "Serializable",
    },
  ];
  return opts;
});

// Initialize isolation level from statement
const initializeFromStatement = () => {
  if (!sheetReady.value || !sheetStatement.value) {
    // If sheet is not ready or statement is empty, reset to undefined
    contextIsolationLevel.value = undefined;
    return;
  }

  const parsed = parseStatement(sheetStatement.value);
  contextIsolationLevel.value = parsed.isolationLevel;
};

watch(
  () => selectedSpec.value?.id,
  () => {
    initializeFromStatement();
  },
  {
    immediate: true,
  }
);

// Watch for sheet statement changes to update isolation level
// Only update when statement changes externally, not when we change it ourselves
watch(
  [sheetStatement, sheetReady],
  () => {
    if (!isUpdatingFromUI.value) {
      initializeFromStatement();
    }
  },
  { immediate: true }
);

const setIsolationInStatement = async (
  isolationLevel: IsolationLevel | undefined
) => {
  const updatedStatement = updateIsolationLevel(
    sheetStatement.value,
    isolationLevel
  );

  if (isCreating.value) {
    // When creating a plan, update the local sheet directly.
    if (!sheet.value) return;
    setSheetStatement(sheet.value, updatedStatement);
  } else {
    // For created plans, create new sheet and update plan/spec
    await updateSpecSheetWithStatement(
      plan.value,
      selectedSpec.value,
      updatedStatement
    );
  }
  events.emit("update");
};
</script>
