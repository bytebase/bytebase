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
    <div class="col-span-3 flex flex-row justify-start items-center">
      <NRadio :value="-1" :label="$t('issue.grant-request.custom')" />
      <NDatePicker
        :value="
          state.selected === -1 ? state.expirationTimestampInMS : undefined
        "
        :disabled="state.selected !== -1"
        :actions="null"
        :update-value-on-close="true"
        type="datetime"
        :is-date-disabled="(date: number) => date < Date.now()"
        clearable
        @update:value="(val) => (state.expirationTimestampInMS = val)"
      />
      <span v-if="maximumRoleExpiration" class="ml-3 textinfolabel">
        {{ $t("settings.general.workspace.maximum-role-expiration.self") }}:
        {{ $t("common.date.days", { days: maximumRoleExpiration }) }}
      </span>
    </div>
  </NRadioGroup>
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup, NDatePicker } from "naive-ui";
import { computed, reactive, watch, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useSettingV1Store } from "@/store";

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
}>();

const emit = defineEmits<{
  (event: "update:timestampInMs", timestampInMS: number | undefined): void;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

const state = reactive<LocalState>({
  selected: props.timestampInMs === undefined ? 0 : -1,
});

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

const onSelect = (value: number) => {
  if (value >= 0) {
    state.expirationTimestampInMS = value * 24 * 60 * 60 * 1000;
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
  () => state.expirationTimestampInMS,
  () => {
    emit("update:timestampInMs", state.expirationTimestampInMS);
  }
);
</script>
