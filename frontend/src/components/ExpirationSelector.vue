<template>
  <div class="w-full flex flex-col gap-y-3">
    <NSelect
      v-bind="$attrs"
      :value="state.selected"
      :options="selectOptions"
      @update:value="onSelect"
    />

    <template v-if="state.selected === -1">
      <div class="flex flex-col gap-y-2">
        <NDatePicker
          size="medium"
          class="w-full"
          :value="state.expirationTimestampInMS"
          :actions="null"
          :update-value-on-close="true"
          type="datetime"
          :is-date-disabled="isDateDisabled"
          clearable
          :placeholder="$t('issue.grant-request.custom-date-placeholder')"
          @update:value="handleCustomDateChange"
        />
        <div v-if="maximumRoleExpiration" class="text-xs text-gray-500">
          {{ $t("settings.general.workspace.maximum-role-expiration.self") }}:
          {{ $t("common.date.days", { days: maximumRoleExpiration }) }}
        </div>
      </div>
    </template>

    <div
      v-if="state.selected !== -1"
      class="p-3 bg-gray-50 rounded-md text-sm text-gray-600"
    >
      <div v-if="state.expirationTimestampInMS">
        {{ $t("common.access-expires") }}:
        <span class="font-medium text-gray-900">
          {{ formatExpirationDisplay(state.expirationTimestampInMS) }}
        </span>
      </div>
      <div v-else>
        {{ $t("project.members.role-never-expires") }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import dayjs from "dayjs";
import { NDatePicker, NSelect } from "naive-ui";
import { computed, reactive, watch } from "vue";
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
  role: string;
}>();

const emit = defineEmits<{
  (event: "update:timestampInMs", timestampInMS: number | undefined): void;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();

// Remember the last selected expiration option
const lastSelectedExpiration = useLocalStorage<number>(
  "bb.roles.last-expiration-selection",
  7 // Default to 1 week
);

// Initialize state with intelligent defaults
const getInitialSelection = () => {
  if (props.timestampInMs !== undefined) {
    // If timestamp is provided, use custom date
    return -1;
  }
  // Otherwise, use the last selected value if it's still valid
  return lastSelectedExpiration.value;
};

const state = reactive<LocalState>({
  selected: getInitialSelection(),
  expirationTimestampInMS: props.timestampInMs,
});

const handleCustomDateChange = (val: number | null) => {
  state.expirationTimestampInMS = val || undefined;
};

const maximumRoleExpiration = computed(() => {
  if (props.role === PresetRoleType.PROJECT_OWNER) {
    return undefined;
  }
  const seconds = settingV1Store.workspaceProfileSetting?.maximumRoleExpiration
    ?.seconds
    ? Number(
        settingV1Store.workspaceProfileSetting.maximumRoleExpiration.seconds
      )
    : undefined;
  if (!seconds) {
    return undefined;
  }
  return Math.floor(seconds / (60 * 60 * 24));
});

const formatExpirationDisplay = (timestampMs: number) => {
  return dayjs(timestampMs).format("YYYY-MM-DD HH:mm:ss");
};

const isDateDisabled = (date: number) => {
  if (date < dayjs().startOf("day").valueOf()) {
    return true;
  }
  if (!maximumRoleExpiration.value) {
    return false;
  }
  return date > dayjs().add(maximumRoleExpiration.value, "days").valueOf();
};

const options = computed((): ExpirationOption[] => {
  const baseOptions = [
    {
      value: 1,
      label: t("common.date.days", { days: 1 }),
    },
    {
      value: 3,
      label: t("common.date.days", { days: 3 }),
    },
    {
      value: 7,
      label: t("common.date.week", { weeks: 1 }),
    },
    {
      value: 30,
      label: t("common.date.month", { months: 1 }),
    },
    {
      value: 90,
      label: t("common.date.months", { months: 3 }),
    },
    {
      value: 180,
      label: t("common.date.months", { months: 6 }),
    },
    {
      value: 365,
      label: t("common.date.year", { years: 1 }),
    },
  ];

  const availableOptions: ExpirationOption[] = [];

  // Add "Never expires" at the top if no maximum role expiration is set
  if (!maximumRoleExpiration.value) {
    availableOptions.push({
      value: 0,
      label: t("project.members.never-expires"),
    });
  }

  // Add custom date option prominently after "Never expires"
  availableOptions.push({
    value: -1,
    label: t("issue.grant-request.custom-date"),
  });

  // Add time-based options
  if (maximumRoleExpiration.value) {
    availableOptions.push(
      ...baseOptions.filter(
        (option) => option.value <= maximumRoleExpiration.value!
      )
    );
  } else {
    availableOptions.push(...baseOptions);
  }

  return availableOptions;
});

const selectOptions = computed(() => {
  return options.value.map((option) => ({
    label: option.label,
    value: option.value,
  }));
});

const onSelect = (value: number) => {
  if (value > 0) {
    state.expirationTimestampInMS =
      new Date().getTime() + value * 24 * 60 * 60 * 1000;
    // Save preset selections to localStorage
    lastSelectedExpiration.value = value;
  } else if (value === 0) {
    // Never expires - clear the timestamp
    state.expirationTimestampInMS = undefined;
    lastSelectedExpiration.value = value;
  }
  // For value === -1 (custom date), the timestamp is set by the date picker
  // Don't save custom date selection as it's not reusable
  state.selected = value;

  // Emit the change immediately for pre-defined options
  if (value !== -1) {
    emit("update:timestampInMs", state.expirationTimestampInMS);
  }
};

watch(
  () => props.role,
  () => {
    let value = state.selected;
    if (!options.value.find((o) => o.value === state.selected)) {
      const neverExpire = options.value.find((o) => o.value === 0);
      if (neverExpire) {
        value = neverExpire.value;
      } else {
        value = options.value[0].value;
      }
    }
    onSelect(value);
  },
  { immediate: true }
);

watch(
  () => state.expirationTimestampInMS,
  () => {
    emit("update:timestampInMs", state.expirationTimestampInMS);
  }
);

defineExpose({
  isValid: computed(() => {
    if (state.expirationTimestampInMS === undefined) {
      return !maximumRoleExpiration.value;
    }
    return state.expirationTimestampInMS > new Date().getTime();
  }),
});
</script>
