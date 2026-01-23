<template>
  <div class="flex flex-col">
    <NInputGroup v-if="enforceDomain && !disabled">
      <NInput
        v-model:value="state.shortValue"
        :size="size"
        :disabled="disabled"
        :status="hasEmailError ? 'error' : undefined"
      />
      <NInputGroupLabel :size="size"> @ </NInputGroupLabel>
      <NSelect
        :size="size"
        v-model:value="state.domain"
        :options="domainSelectOptions"
        :disabled="disabled"
      />
    </NInputGroup>
    <NInput
      v-else
      v-model:value="state.value"
      :size="size"
      :disabled="disabled"
      :status="hasEmailError ? 'error' : undefined"
    />
    <span v-if="hasEmailError" class="text-error text-sm mt-1">
      {{ t("common.email-ascii-only") }}
    </span>
  </div>
</template>

<script lang="ts" setup>
import { NInput, NInputGroup, NInputGroupLabel, NSelect } from "naive-ui";
import { computed, reactive, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useSettingV1Store } from "@/store";

const { t } = useI18n();

// WHATWG HTML spec email validation (lowercase only).
// https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
const emailRegex =
  /^[a-z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$/;

const isValidEmail = (email: string): boolean => {
  return emailRegex.test(email);
};

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
    disabled?: boolean;
    domain?: string;
    showDomain?: boolean;
  }>(),
  {
    size: "medium",
    value: "",
    disabled: false,
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
    settingV1Store.workspaceProfile.enforceIdentityDomain || props.showDomain
  );
});

const domainSelectOptions = computed(() => {
  if (props.domain) {
    return [
      {
        label: props.domain,
        value: props.domain,
      },
    ];
  }
  const domains = settingV1Store.workspaceProfile.domains.filter(
    (domain) => domain && domain.trim() !== ""
  );
  return domains.map((domain) => {
    const value = domain.trim();
    return {
      label: value,
      value,
    };
  });
});

const hasEmailError = computed(() => {
  const email = enforceDomain.value
    ? `${state.shortValue}@${state.domain}`
    : state.value;
  if (!email || !email.includes("@")) {
    return false;
  }
  return !isValidEmail(email);
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
