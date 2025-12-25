<template>
  <NInputGroup v-if="enforceDomain && !readonly">
    <NInput
      v-model:value="state.shortValue"
      :size="size"
      :disabled="readonly"
    />
    <NInputGroupLabel :size="size"> @ </NInputGroupLabel>
    <NSelect
      :size="size"
      v-model:value="state.domain"
      :options="domainSelectOptions"
      :disabled="readonly"
    />
  </NInputGroup>
  <NInput
    v-else
    v-model:value="state.value"
    :size="size"
    :disabled="readonly"
  />
</template>

<script lang="ts" setup>
import { NInput, NInputGroup, NInputGroupLabel, NSelect } from "naive-ui";
import { computed, reactive, watch, watchEffect } from "vue";
import { useSettingV1Store } from "@/store";

interface LocalState {
  // The full email value.
  value: string;
  // The short value without the domain.
  shortValue: string;
  domain: string;
}

const props = withDefaults(
  defineProps<{
    size?: "small" | "medium" | "large";
    value?: string;
    readonly?: boolean;
    domainPrefix?: string;
    fallbackDomain?: string;
    showDomain?: boolean;
  }>(),
  {
    size: "medium",
    value: "",
    readonly: false,
    domainPrefix: "",
    fallbackDomain: "",
    showDomain: false,
  }
);

const emit = defineEmits<{
  (event: "update:value", value: string): void;
}>();

const state: LocalState = reactive({
  value: props.value,
  shortValue: props.value.split("@")[0],
  domain: props.value.split("@")[1],
});
const settingV1Store = useSettingV1Store();

const enforceDomain = computed(() => {
  return (
    (settingV1Store.workspaceProfileSetting?.enforceIdentityDomain ?? false) ||
    props.showDomain
  );
});

const domainSelectOptions = computed(() => {
  const domains = (
    settingV1Store.workspaceProfileSetting?.domains ?? []
  ).filter((domain) => domain && domain.trim() !== "");
  if (domains.length === 0 && props.fallbackDomain) {
    domains.push(props.fallbackDomain);
  }
  return domains.map((domain) => {
    const value = props.domainPrefix
      ? `${props.domainPrefix}.${domain.trim()}`
      : domain.trim();
    return {
      label: value,
      value,
    };
  });
});

watchEffect(() => {
  if (domainSelectOptions.value.length > 0) {
    if (
      !domainSelectOptions.value.find((option) => option.value === state.domain)
    ) {
      state.domain = domainSelectOptions.value[0].value;
    }
  }
});

watch([() => state.value, () => state.shortValue, () => state.domain], () => {
  const email = enforceDomain.value
    ? state.shortValue
      ? `${state.shortValue}@${state.domain}`
      : ""
    : state.value;
  emit("update:value", email);
});
</script>
