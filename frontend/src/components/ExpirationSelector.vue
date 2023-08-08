<template>
  <NRadioGroup
    v-bind="$attrs"
    v-model:value="state.value"
    class="w-full !grid grid-cols-3 gap-2"
    name="radiogroup"
  >
    <div
      v-for="option in options"
      :key="option.value"
      class="col-span-1 h-8 flex flex-row justify-start items-center"
    >
      <NRadio :value="option.value" :label="option.label" />
    </div>
    <div class="col-span-2 flex flex-row justify-start items-center">
      <NRadio :value="-1" :label="$t('issue.grant-request.customize')" />
      <NInputNumber
        v-model:value="state.customValue"
        class="!w-24 ml-2"
        :disabled="!useCustom"
        :min="1"
        :show-button="false"
        :placeholder="''"
      >
        <template #suffix>{{ $t("common.date.days") }}</template>
      </NInputNumber>
    </div>
  </NRadioGroup>
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup, NInputNumber } from "naive-ui";
import { computed, reactive, watch } from "vue";

interface ExpirationOption {
  value: number;
  label: string;
}

interface LocalState {
  value: number;
  customValue: number;
}

const props = defineProps<{
  options: ExpirationOption[];
  value: number;
}>();

const emit = defineEmits<{
  (event: "update", value: number): void;
}>();

const state = reactive<LocalState>({
  value: props.value,
  customValue: props.value,
});

const useCustom = computed(() => state.value === -1);

watch(
  () => [state.value, state.customValue],
  () => {
    emit("update", useCustom.value ? state.customValue : state.value);
  }
);
</script>
