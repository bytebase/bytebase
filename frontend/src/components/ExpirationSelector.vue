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
    <div class="col-span-3 flex flex-row justify-start items-center">
      <NRadio :value="-1" :label="$t('issue.grant-request.custom')" />
      <NInputNumber
        v-model:value="state.customValue"
        class="!w-24 ml-2"
        :disabled="!useCustom"
        :min="1"
        :max="maximumRoleExpiration"
        :show-button="false"
        :placeholder="''"
      >
        <template #suffix>{{ $t("common.date.days") }}</template>
      </NInputNumber>

      <span v-if="maximumRoleExpiration" class="ml-3 textinfolabel">
        {{ $t("settings.general.workspace.maximum-role-expiration.self") }}:
        {{ $t("common.date.days", { days: maximumRoleExpiration }) }}
      </span>
    </div>
  </NRadioGroup>
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup, NInputNumber } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useSettingV1Store } from "@/store";

interface ExpirationOption {
  value: number;
  label: string;
}

interface LocalState {
  value: number;
  customValue: number;
}

const props = defineProps<{
  value: number;
}>();

const emit = defineEmits<{
  (event: "update", value: number): void;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

const state = reactive<LocalState>({
  value: props.value,
  customValue: 7,
});

const useCustom = computed(() => state.value === -1);

const maximumRoleExpiration = computed(() => {
  const seconds =
    settingV1Store.workspaceProfileSetting?.maximumRoleExpiration?.seconds?.toNumber();
  if (!seconds) {
    return undefined;
  }
  return Math.floor(seconds / (60 * 60 * 24));
});

const options = computed((): ExpirationOption[] => {
  let options = [
    {
      value: 1,
      label: t("common.date.days", { days: 1 }),
    },
    {
      value: 3,
      label: t("common.date.days", { days: 3 }),
    },
    {
      value: 30,
      label: t("common.date.days", { days: 30 }),
    },
    {
      value: 90,
      label: t("common.date.days", { days: 90 }),
    },
  ];
  if (maximumRoleExpiration.value) {
    options = options.filter(
      (option) => option.value < maximumRoleExpiration.value!
    );
    options.push({
      value: maximumRoleExpiration.value,
      label: t("common.date.days", { days: maximumRoleExpiration.value }),
    });
  } else {
    options.push({
      value: 0,
      label: t("project.members.never-expires"),
    });
  }
  return options;
});

watch(
  () => [state.value, state.customValue],
  () => {
    emit("update", useCustom.value ? state.customValue : state.value);
  }
);
</script>
