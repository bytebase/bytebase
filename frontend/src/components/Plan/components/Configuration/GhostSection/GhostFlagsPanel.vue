<template>
  <Drawer :show="show" @close="close">
    <DrawerContent
      :title="title"
      class="w-screen md:max-w-[calc(100vw-8rem)] md:w-[40vw]"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <p class="font-medium text-control">
            {{ $t("task.online-migration.gho\st-parameters") }}
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
import { create } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import ErrorList from "@/components/misc/ErrorList.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { planServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { useGhostSettingContext } from "./context";
import FlagsForm from "./FlagsForm";

defineProps<{
  show: boolean;
}>();

const emits = defineEmits<{
  (e: "update:show", value: boolean): void;
}>();

const { t } = useI18n();
const { isCreating, allowChange, plan, selectedSpec, events } =
  useGhostSettingContext();

const title = computed(() => {
  return t("task.online-migration.configure-ghost-parameters");
});
const config = computed(() => {
  return selectedSpec.value?.config?.case === "changeDatabaseConfig"
    ? selectedSpec.value.config.value
    : undefined;
});
const flags = ref<Record<string, string>>({});

const isDirty = computed(() => {
  return !isEqual(config.value?.ghostFlags ?? {}, flags.value);
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
  return !allowChange.value;
});

const close = () => {
  flags.value = cloneDeep(config.value?.ghostFlags ?? {});
  emits("update:show", false);
};

const trySave = async () => {
  if (errors.value.length > 0) {
    return;
  }

  if (isCreating.value) {
    if (
      !selectedSpec.value ||
      selectedSpec.value.config?.case !== "changeDatabaseConfig"
    )
      return;
    selectedSpec.value.config.value.ghostFlags = cloneDeep(flags.value);
  } else {
    const planPatch = cloneDeep(plan.value);
    const spec = (planPatch?.specs || []).find((spec) => {
      return spec.id === selectedSpec.value?.id;
    });
    if (!planPatch || !spec || spec.config?.case !== "changeDatabaseConfig") {
      // Should not reach here.
      throw new Error(
        "Plan or spec is not defined. Cannot update gh-ost flags."
      );
    }

    spec.config.value.ghostFlags = cloneDeep(flags.value);
    const request = create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["specs"] },
    });
    await planServiceClientConnect.updatePlan(request);
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
  () => config.value?.ghostFlags,
  (newFlags, oldFlags) => {
    if (isEqual(newFlags, oldFlags)) {
      return;
    }
    flags.value = cloneDeep(newFlags ?? {});
  },
  { immediate: true, deep: true }
);
</script>
