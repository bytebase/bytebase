<template>
  <NRadioGroup
    v-bind="$attrs"
    :value="state.selected"
    class="w-full !grid grid-cols-3 gap-2"
    name="radiogroup"
    @update:value="onSelect"
  >
    <div
      v-for="option in options"
      :key="option.value"
      class="col-span-1 h-8 flex flex-row justify-start items-center"
    >
      <NRadio :value="option.value" :label="option.label" />
    </div>
    <div class="col-span-2 flex flex-row justify-start items-center">
      <NRadio :value="-2" :label="$t('issue.grant-request.custom')" />
      <NInputNumber
        size="small"
        style="width: 5rem"
        :min="1"
        :placeholder="$t('common.date.days')"
        :disabled="state.selected !== -2"
        :value="customExpirationDays"
        @update:value="handleCustomExpirationDaysChange"
      />
      <span class="ml-2">{{ $t("common.date.days") }}</span>
    </div>
    <div class="col-span-3 flex flex-row justify-start items-center">
      <NRadio :value="-1" :label="$t('issue.grant-request.custom')" />
      <NDatePicker
        size="small"
        :value="
          state.selected === -1 ? state.expirationTimestampInMS : undefined
        "
        :disabled="state.selected !== -1"
        :actions="null"
        :update-value-on-close="true"
        type="datetime"
        :is-date-disabled="isDateDisabled"
        clearable
        @update:value="(val) => (state.expirationTimestampInMS = val)"
      />
      <span
        v-if="maximumRoleExpiration && enableExpirationLimit"
        class="ml-3 textinfolabel"
      >
        {{ $t("settings.general.workspace.maximum-role-expiration.self") }}:
        {{ $t("common.date.days", { days: maximumRoleExpiration }) }}
      </span>
    </div>
  </NRadioGroup>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import dayjs from "dayjs";
import { NRadio, NRadioGroup, NDatePicker, NInputNumber } from "naive-ui";
import { computed, reactive, watch, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useSettingV1Store } from "@/store";
import { PresetRoleType } from "@/types";

interface ExpirationOption {
  value: number;
  label: string;
}

interface LocalState {
  selected: number;
  expirationTimestampInMS?: number;
}

const props = defineProps<{
  timestampInMs?: number;
  role?: string;
}>();

const emit = defineEmits<{
  (event: "update:timestampInMs", timestampInMS: number | undefined): void;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

const state = reactive<LocalState>({
  selected: props.timestampInMs === undefined ? 0 : -1,
});
const customExpirationDays = useLocalStorage(
  "bb.roles.custom-expiration-days",
  1
);

const handleCustomExpirationDaysChange = (val: number | null) => {
  if (!val) {
    return;
  }
  state.expirationTimestampInMS =
    new Date().getTime() + val * 24 * 60 * 60 * 1000;
  customExpirationDays.value = val;
};

const enableExpirationLimit = computed(() => {
  return (
    props.role === PresetRoleType.SQL_EDITOR_USER ||
    props.role === PresetRoleType.PROJECT_EXPORTER
  );
});

const maximumRoleExpiration = computed(() => {
  const seconds =
    settingV1Store.workspaceProfileSetting?.maximumRoleExpiration?.seconds?.toNumber();
  if (!seconds) {
    return undefined;
  }
  return Math.floor(seconds / (60 * 60 * 24));
});

const isDateDisabled = (date: number) => {
  if (date < Date.now()) {
    return true;
  }
  if (!maximumRoleExpiration.value) {
    return false;
  }
  return date > dayjs().add(maximumRoleExpiration.value, "days").valueOf();
};

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
  if (maximumRoleExpiration.value && enableExpirationLimit.value) {
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

const onSelect = (value: number) => {
  if (value > 0) {
    state.expirationTimestampInMS =
      new Date().getTime() + value * 24 * 60 * 60 * 1000;
  } else {
    state.expirationTimestampInMS = undefined;
  }
  state.selected = value;
};

onMounted(() => {
  let value = state.selected;
  if (!options.value.find((o) => o.value === state.selected)) {
    value = options.value[0].value;
  }
  onSelect(value);
});

watch(
  () => props.role,
  () => onSelect(options.value[0].value)
);

watch(
  () => state.expirationTimestampInMS,
  () => {
    emit("update:timestampInMs", state.expirationTimestampInMS);
  }
);
</script>
