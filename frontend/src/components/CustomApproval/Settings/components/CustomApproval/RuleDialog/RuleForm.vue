<template>
  <div
    class="w-[calc(100vw-8rem)] lg:max-w-[35vw] max-h-[calc(100vh-10rem)] flex flex-col gap-y-4 text-sm"
  >
    <div class="flex-1 flex flex-col px-0.5 overflow-hidden space-y-4">
      <div class="space-y-1">
        <label class="block font-medium text-control space-x-1">
          <RequiredStar />
          {{ $t("common.name") }}
        </label>
        <NInput
          ref="nameInputRef"
          v-model:value="state.rule.template.title"
          :show-count="true"
          :maxlength="64"
          :disabled="!allowEditRule"
          @update:value="$emit('update')"
        />
      </div>
      <div class="space-y-1">
        <label class="block font-medium text-control space-x-1">
          <RequiredStar />
          {{ $t("common.description") }}
        </label>
        <NInput
          v-model:value="state.rule.template.description"
          type="textarea"
          :autosize="{
            minRows: 3,
            maxRows: 5,
          }"
          :disabled="!allowEditRule"
          @update:value="$emit('update')"
        />
      </div>
      <div class="w-full flex-1 space-y-1 overflow-y-auto">
        <label class="block font-medium text-control space-x-1">
          <RequiredStar />
          {{ $t("custom-approval.approval-flow.node.nodes") }}
        </label>
        <div class="text-control-light">
          {{ $t("custom-approval.approval-flow.node.description") }}
        </div>
        <div class="py-1 w-[30rem] space-y-2">
          <StepsTable
            v-if="state.rule.template.flow"
            :flow="state.rule.template.flow"
            :editable="allowEditRule"
            @update="$emit('update')"
          />
        </div>
      </div>
    </div>

    <footer
      v-if="allowEditRule"
      class="flex items-center justify-end gap-x-2 pt-4 border-t"
    >
      <NButton @click="$emit('cancel')">{{ $t("common.cancel") }}</NButton>
      <NButton
        type="primary"
        :disabled="!allowCreateOrUpdate"
        @click="handleUpsert"
      >
        {{ mode === "CREATE" ? $t("common.create") : $t("common.update") }}
      </NButton>
    </footer>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NInput } from "naive-ui";
import { computed, nextTick, onMounted, ref } from "vue";
import { LocalApprovalRule, SYSTEM_BOT_USER_NAME } from "@/types";
import { creatorOfRule } from "@/utils";
import { RequiredStar } from "../../common";
import { StepsTable } from "../common";
import { useCustomApprovalContext } from "../context";
import { validateApprovalTemplate } from "../logic";

type LocalState = {
  rule: LocalApprovalRule;
};

const props = defineProps<{
  dirty?: boolean;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "update"): void;
  (
    event: "save",
    newRule: LocalApprovalRule,
    oldRule: LocalApprovalRule | undefined
  ): void;
}>();

const nameInputRef = ref<InstanceType<typeof NInput>>();
const context = useCustomApprovalContext();
const { allowAdmin, dialog } = context;

const mode = computed(() => dialog.value!.mode);
const rule = computed(() => dialog.value!.rule);

const resolveLocalState = (): LocalState => {
  return {
    rule: cloneDeep(rule.value),
  };
};

const state = ref(resolveLocalState());

const allowEditRule = computed(() => {
  if (!allowAdmin.value) return false;
  return creatorOfRule(rule.value).name !== SYSTEM_BOT_USER_NAME;
});

const allowCreateOrUpdate = computed(() => {
  if (!validateApprovalTemplate(state.value.rule.template)) {
    return false;
  }
  if (mode.value === "EDIT") {
    if (!props.dirty) return false;
  }
  return true;
});

const handleUpsert = () => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }

  const oldRule = mode.value === "EDIT" ? rule.value : undefined;
  const newRule = cloneDeep(state.value.rule);
  emit("save", newRule, oldRule);
};

onMounted(() => {
  if (allowEditRule.value) {
    nextTick(() => {
      nameInputRef.value?.inputElRef?.focus();
    });
  }
});
</script>
