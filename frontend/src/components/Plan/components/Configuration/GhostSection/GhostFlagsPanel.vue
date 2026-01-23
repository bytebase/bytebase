<template>
  <Drawer :show="show" @close="close">
    <DrawerContent
      :title="title"
      class="w-screen md:max-w-[calc(100vw-8rem)] md:w-[40vw]"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <p class="font-medium text-control">
            {{ $t("task.online-migration.ghost-parameters") }}
            <LearnMoreLink
              class="text-sm ml-1"
              url="https://github.com/github/gh-ost/blob/master/doc/command-line-flags.md"
            />
          </p>
          <FlagsForm v-model:flags="flags" :readonly="readonly" />
        </div>
      </template>
      <template #footer>
        <div class="flex flex-row justify-end gap-x-2">
          <NButton @click="close">{{ $t("common.cancel") }}</NButton>
          <NTooltip :disabled="errors.length === 0">
            <template #trigger>
              <NButton type="primary" :disabled="!isDirty" @click="trySave">
                {{ $t("common.save") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="errors" />
            </template>
          </NTooltip>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import { updateSpecSheetWithStatement } from "@/components/Plan/logic";
import { Drawer, DrawerContent } from "@/components/v2";
import { pushNotification } from "@/store";
import { setSheetStatement } from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";
import {
  getGhostConfig,
  updateGhostConfig,
} from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import { useGhostSettingContext } from "./context";
import FlagsForm from "./FlagsForm";

defineProps<{
  show: boolean;
}>();

const emits = defineEmits<{
  (e: "update:show", value: boolean): void;
}>();

const { t } = useI18n();
const { isCreating, allowChange, plan, events } = useGhostSettingContext();
const { selectedSpec } = useSelectedSpec();
const { sheet, sheetStatement, sheetReady, isSheetOversize } =
  useSpecSheet(selectedSpec);

const title = computed(() => {
  return t("task.online-migration.configure-ghost-parameters");
});

const flags = ref<Record<string, string>>({});

// Get current flags from sheet directive
const currentFlags = computed(() => {
  if (!sheetReady.value) return {};
  return getGhostConfig(sheetStatement.value) ?? {};
});

const isDirty = computed(() => {
  return !isEqual(currentFlags.value, flags.value);
});

const errors = computed(() => {
  const errors: string[] = [];
  if (!isDirty.value) {
    errors.push(t("task.online-migration.error.nothing-changed"));
  }
  return errors;
});

const readonly = computed(() => {
  if (isCreating.value) return false;
  return !allowChange.value || isSheetOversize.value;
});

const close = () => {
  flags.value = cloneDeep(currentFlags.value);
  emits("update:show", false);
};

const trySave = async () => {
  if (errors.value.length > 0) {
    return;
  }

  const updatedStatement = updateGhostConfig(
    sheetStatement.value,
    cloneDeep(flags.value)
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
    events.emit("update");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
  close();
};

watch(
  currentFlags,
  (newFlags, oldFlags) => {
    if (isEqual(newFlags, oldFlags)) {
      return;
    }
    flags.value = cloneDeep(newFlags);
  },
  { immediate: true, deep: true }
);
</script>
