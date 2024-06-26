<template>
  <NInputGroup v-if="showDomainLabel">
    <NInput
      v-model:value="state.shortValue"
      :size="size"
      :readonly="readonly"
    />
    <NInputGroupLabel :size="size"> @{{ domain }} </NInputGroupLabel>
  </NInputGroup>
  <NInput
    v-else
    v-model:value="state.value"
    :size="size"
    :readonly="readonly"
  />
</template>

<script lang="ts" setup>
import { NInput, NInputGroup, NInputGroupLabel } from "naive-ui";
import { reactive, watch, ref, onMounted } from "vue";

interface LocalState {
  // The full email value.
  value: string;
  // The short value without the domain.
  shortValue: string;
}

const props = withDefaults(
  defineProps<{
    size: "small" | "medium" | "large";
    domain?: string;
    value?: string;
    readonly?: boolean;
  }>(),
  {
    size: "medium",
    domain: undefined,
    value: "",
    readonly: false,
  }
);

const emit = defineEmits<{
  (event: "update:value", value: string): void;
}>();

const state: LocalState = reactive({
  value: props.value,
  shortValue: props.value.split("@")[0],
});
const showDomainLabel = ref(false);

onMounted(() => {
  if (props.domain) {
    if (!props.value || props.value.endsWith(`@${props.domain}`)) {
      showDomainLabel.value = true;
    }
  }
});

watch([() => state.value, () => state.shortValue], () => {
  const email = showDomainLabel.value
    ? state.shortValue
      ? `${state.shortValue}@${props.domain}`
      : ""
    : state.value;
  emit("update:value", email);
});
</script>
