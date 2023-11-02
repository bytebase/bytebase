<template>
  <MazPhoneNumberInput
    v-model="state.phoneNumber"
    no-example
    :preferred-countries="['CN', 'US']"
    :default-country-code="defaultCountryCode"
    size="sm"
    :translations="translation"
    @update="handleUpdate"
  />
</template>

<script lang="ts" setup>
import MazPhoneNumberInput from "maz-ui/components/MazPhoneNumberInput";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useLanguage } from "@/composables/useLanguage";

const props = defineProps<{
  value: string;
}>();

const emit = defineEmits<{
  (event: "update", value: string): void;
}>();

interface LocalState {
  phoneNumber: string;
}

const { t } = useI18n();
const { locale } = useLanguage();
const state = reactive<LocalState>({
  phoneNumber: props.value,
});
const results = ref();

const translation = computed(() => {
  return {
    countrySelector: {
      placeholder: t("settings.profile.country-code"),
      error: "",
      searchPlaceholder: "",
    },
    phoneInput: {
      placeholder: t("settings.profile.phone"),
      example: "",
    },
  };
});

const defaultCountryCode = computed(() => {
  return locale.value === "zh-CN" ? "CN" : "US";
});

const handleUpdate = (value: any) => {
  results.value = value;
  emit("update", value.e164);
};
</script>

<style>
.m-phone-number-input__input > .m-input-wrapper {
  @apply !-ml-px;
}
.m-input-wrapper {
  @apply !border rounded;
}
.m-select-list {
  @apply !bg-white !shadow;
}
.m-phone-number-input__select.m-input-input {
  @apply -mt-1;
}
.m-phone-number-input__country-flag {
  @apply !bottom-1.5;
}
</style>
