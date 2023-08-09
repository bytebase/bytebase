<template>
  <DrawerContent :title="$t('database.automatic-backup')">
    <div class="max-w-full flex flex-col items-center">
      <div class="py-0.5 space-y-6">
        <div class="flex flex-col gap-y-2">
          <label class="textlabel">
            {{ $t("database.backup-setting.form.schedule") }}
          </label>
          <NPopover
            trigger="manual"
            :show="state.showBackupPolicyEnforcement"
            :show-arrow="false"
            placement="bottom-start"
          >
            <template #trigger>
              <div class="flex items-center gap-x-2 text-sm">
                <label
                  v-for="schedule in PLAN_SCHEDULES"
                  :key="schedule"
                  class="flex items-center gap-x-2"
                  :class="
                    !isAllowedScheduleByPolicy(schedule) &&
                    'opacity-50 cursor-not-allowed'
                  "
                  @click="setSchedule(schedule)"
                >
                  <input
                    type="radio"
                    :value="schedule"
                    :checked="schedule === checkedSchedule"
                    :disabled="!isAllowedScheduleByPolicy(schedule)"
                  />
                  <span>{{ nameOfSchedule(schedule) }}</span>
                </label>
              </div>
            </template>

            <router-link
              class="normal-link text-sm"
              :to="`/environment/${database.instanceEntity.environmentEntity.uid}`"
            >
              {{
                $t(
                  "database.backuppolicy-backup-enforced-and-cant-be-disabled",
                  [
                    $t(
                      `database.backup-policy.${backupPlanScheduleToJSON(
                        backupPolicy
                      )}`
                    ),
                  ]
                )
              }}
            </router-link>
          </NPopover>
        </div>

        <div
          v-if="checkedSchedule === BackupPlanSchedule.WEEKLY"
          class="flex flex-col gap-y-2"
        >
          <label class="textlabel">
            {{ $t("database.backup-setting.form.day-of-week") }}
          </label>
          <div class="w-[16rem]">
            <BBSelect
              :selected-item="
                localFromUTC(state.setting.hour, state.setting.dayOfWeek)
                  .dayOfWeek
              "
              :item-list="AVAILABLE_DAYS_OF_WEEK"
              @select-item="setDayOfWeek"
            >
              <template #menuItem="{ item: day }">
                {{ nameOfDay(day) }}
              </template>
            </BBSelect>
          </div>
        </div>

        <div
          v-if="
            checkedSchedule === BackupPlanSchedule.WEEKLY ||
            checkedSchedule === BackupPlanSchedule.DAILY
          "
          class="flex flex-col gap-y-2"
        >
          <label class="textlabel">
            <span>
              {{ $t("database.backup-setting.form.time-of-day") }}
            </span>
            <span class="ml-1 textinfolabel">
              ({{ Intl.DateTimeFormat().resolvedOptions().timeZone }})
            </span>
          </label>
          <div class="w-[16rem]">
            <BBSelect
              :selected-item="
                localFromUTC(state.setting.hour, state.setting.dayOfWeek).hour
              "
              :item-list="AVAILABLE_HOURS_OF_DAY"
              @select-item="setHour"
            >
              <template #menuItem="{ item: hour }">
                {{ nameOfHour(hour) }}
              </template>
            </BBSelect>
          </div>
        </div>

        <div
          v-if="
            checkedSchedule === BackupPlanSchedule.WEEKLY ||
            checkedSchedule === BackupPlanSchedule.DAILY
          "
          class="flex flex-col gap-y-2"
        >
          <label class="textlabel">
            {{ $t("database.backup-setting.form.retention-period") }}
          </label>
          <div class="w-[16rem]">
            <input
              type="number"
              class="textfield w-full hide-ticker"
              :placeholder="String(DEFAULT_BACKUP_RETENTION_PERIOD_DAYS)"
              :value="retentionPeriodDaysInputValue"
              @input="(e: any) => setRetentionPeriodDays(e.target.value)"
            />
          </div>
        </div>
      </div>

      <div
        class="w-full mt-5 pt-4 flex justify-end border-t border-block-border"
      >
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!isValid"
          @click.prevent="handleSave"
        >
          {{ $t("common.save") }}
        </button>
      </div>

      <div
        v-if="state.loading"
        class="absolute inset-0 z-10 bg-white/70 flex items-center justify-center rounded-lg"
      >
        <BBSpin />
      </div>
    </div>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { NPopover } from "naive-ui";
import { computed, PropType, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { DrawerContent } from "@/components/v2";
import { pushNotification, useBackupV1Store } from "@/store";
import { ComposedDatabase, unknown } from "@/types";
import { BackupSetting } from "@/types/proto/v1/database_service";
import {
  BackupPlanSchedule,
  backupPlanScheduleToJSON,
} from "@/types/proto/v1/org_policy_service";
import {
  AVAILABLE_DAYS_OF_WEEK,
  AVAILABLE_HOURS_OF_DAY,
  PLAN_SCHEDULES,
  DEFAULT_BACKUP_RETENTION_PERIOD_TS,
  DEFAULT_BACKUP_RETENTION_PERIOD_DAYS,
  localFromUTC,
  localToUTC,
  parseScheduleFromBackupSetting,
} from "./utils";

interface BackupSettingEdit {
  enabled: boolean;
  dayOfWeek: number;
  hour: number;
  retentionPeriodTs: number;
}

type LocalState = {
  setting: BackupSettingEdit;
  showBackupPolicyEnforcement: boolean;
  loading: boolean;
};

const BACKUP_POLICY_ENFORCEMENT_POPUP_DURATION = 5000;

const props = defineProps({
  database: {
    required: true,
    type: Object as PropType<ComposedDatabase>,
  },
  allowAdmin: {
    required: true,
    type: Boolean,
  },
  backupPolicy: {
    type: Number as PropType<BackupPlanSchedule>,
    default: BackupPlanSchedule.UNSET,
  },
  backupSetting: {
    type: Object as PropType<BackupSetting>,
    default: () => unknown("BACKUP_SETTING"),
  },
});

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "update", setting: BackupSetting): void;
}>();

const backupStore = useBackupV1Store();

const state = reactive<LocalState>({
  setting: extractEditValue(props.backupSetting),
  showBackupPolicyEnforcement: false,
  loading: false,
});

const { t } = useI18n();

const allowDisableAutoBackup = computed(() => {
  return props.allowAdmin && props.backupPolicy == BackupPlanSchedule.UNSET;
});

const daysOfWeek = computed(() => [
  t("database.week.Sunday"),
  t("database.week.Monday"),
  t("database.week.Tuesday"),
  t("database.week.Wednesday"),
  t("database.week.Thursday"),
  t("database.week.Friday"),
  t("database.week.Saturday"),
]);

watch(
  () => props.backupSetting,
  () => {
    state.setting = extractEditValue(props.backupSetting);
  },
  { deep: true }
);

const checkedSchedule = computed((): BackupPlanSchedule => {
  return parseScheduleFromBackupSetting(
    backupStore.buildSimpleSchedule({
      enabled: state.setting.enabled,
      hourOfDay: state.setting.hour,
      dayOfWeek: state.setting.dayOfWeek,
    })
  );
});

const retentionPeriodDaysInputValue = computed((): string => {
  const seconds = state.setting.retentionPeriodTs;
  if (!seconds || seconds <= 0) return "";
  return String(Math.floor(seconds / 3600 / 24));
});

const isValid = computed((): boolean => {
  const schedule = checkedSchedule.value;
  if (!isAllowedScheduleByPolicy(schedule)) {
    return false;
  }

  const { setting } = state;
  if (!setting.enabled) {
    return true;
  }

  return true;
});

const handleSave = async () => {
  if (!isValid.value) {
    return;
  }

  const { setting } = state;
  if (setting.enabled && setting.retentionPeriodTs <= 0) {
    // Set default value to retentionPeriodTs if needed.
    setting.retentionPeriodTs = DEFAULT_BACKUP_RETENTION_PERIOD_TS;
  }

  try {
    state.loading = true;
    const updatedBackupSetting = await backupStore.upsertBackupSetting({
      name: `${props.database.name}/backupSetting`,
      cronSchedule: backupStore.buildSimpleSchedule({
        enabled: setting.enabled,
        hourOfDay: setting.hour,
        dayOfWeek: setting.dayOfWeek,
      }),
      hookUrl: props.backupSetting.hookUrl,
      backupRetainDuration: {
        seconds: setting.retentionPeriodTs,
        nanos: 0,
      },
    });

    const action = setting.enabled
      ? t("database.enabled")
      : t("database.disabled");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t(
        "database.action-automatic-backup-for-database-props-database-name",
        [action, props.database.databaseName]
      ),
    });

    emit("update", updatedBackupSetting);
  } finally {
    state.loading = false;
  }
};

function setSchedule(schedule: BackupPlanSchedule) {
  if (!isAllowedScheduleByPolicy(schedule)) {
    // show a popup and automatically disappear after several seconds
    state.showBackupPolicyEnforcement = true;
    setTimeout(
      () => (state.showBackupPolicyEnforcement = false),
      BACKUP_POLICY_ENFORCEMENT_POPUP_DURATION
    );
    return;
  }

  state.showBackupPolicyEnforcement = false;

  const { setting } = state;

  const normalizeHour = () => {
    const local = localFromUTC(setting.hour, setting.dayOfWeek);
    const minHour = AVAILABLE_HOURS_OF_DAY[0];
    const maxHour = AVAILABLE_HOURS_OF_DAY[AVAILABLE_HOURS_OF_DAY.length - 1];
    if (local.hour < minHour) local.hour = minHour;
    if (local.hour > maxHour) local.hour = maxHour;
    const utc = localToUTC(local.hour, local.dayOfWeek);
    setting.hour = utc.hour;
    setting.dayOfWeek = utc.dayOfWeek;
  };

  switch (schedule) {
    case BackupPlanSchedule.UNSET:
      setting.enabled = false;
      break;
    case BackupPlanSchedule.WEEKLY:
      setting.enabled = true;
      if (setting.dayOfWeek < 0) {
        setting.dayOfWeek = 0;
      }
      if (!setting.retentionPeriodTs || setting.retentionPeriodTs <= 0) {
        setting.retentionPeriodTs = DEFAULT_BACKUP_RETENTION_PERIOD_TS;
      }
      normalizeHour();
      break;
    case BackupPlanSchedule.DAILY:
      setting.enabled = true;
      setting.dayOfWeek = -1;
      if (!setting.retentionPeriodTs || setting.retentionPeriodTs <= 0) {
        setting.retentionPeriodTs = DEFAULT_BACKUP_RETENTION_PERIOD_TS;
      }
      normalizeHour();
      break;
  }
}

function setDayOfWeek(dayOfWeek: number) {
  // Combine the old local hour with the newly selected local dayOfWeek
  // and convert them to UTC.
  const local = localFromUTC(state.setting.hour, state.setting.dayOfWeek);
  const utc = localToUTC(local.hour, dayOfWeek);
  state.setting.hour = utc.hour;
  state.setting.dayOfWeek = utc.dayOfWeek;
}

function setHour(hour: number) {
  // Combine the old local dayOfWeek with the newly selected local hour
  // and convert them to UTC.
  const local = localFromUTC(state.setting.hour, state.setting.dayOfWeek);
  const utc = localToUTC(hour, local.dayOfWeek);
  state.setting.hour = utc.hour;
  state.setting.dayOfWeek = utc.dayOfWeek;
}

function setRetentionPeriodDays(input: string) {
  const days = parseInt(input, 10);
  if (days <= 0 || Number.isNaN(days)) {
    state.setting.retentionPeriodTs = -1;
  } else {
    state.setting.retentionPeriodTs = days * 3600 * 24;
  }
}

function extractEditValue(backupSetting: BackupSetting): BackupSettingEdit {
  const schedule = backupStore.parseBackupSchedule(backupSetting.cronSchedule);
  return {
    enabled: backupSetting.cronSchedule !== "",
    dayOfWeek: schedule.dayOfWeek,
    hour: schedule.hourOfDay,
    retentionPeriodTs: backupSetting.backupRetainDuration?.seconds ?? 0,
  };
}

function isAllowedScheduleByPolicy(schedule: BackupPlanSchedule): boolean {
  if (schedule === BackupPlanSchedule.UNSET) {
    return allowDisableAutoBackup.value;
  }

  // In the future, a db's backup setting and its environment's backup policy
  // can be set separately.
  // Now, the database backup policy can be configured when its environment
  // backup policy is "Not enforced"
  return (
    props.backupPolicy === schedule ||
    props.backupPolicy === BackupPlanSchedule.UNSET
  );
}

function nameOfSchedule(schedule: BackupPlanSchedule): string {
  switch (schedule) {
    case BackupPlanSchedule.WEEKLY:
      return t("database.backup-setting.schedule.weekly");
    case BackupPlanSchedule.DAILY:
      return t("database.backup-setting.schedule.daily");
    default:
      return t("database.backup-setting.schedule.disabled");
  }
}

function nameOfDay(day: number): string {
  if (day >= 0 && day < daysOfWeek.value.length) return daysOfWeek.value[day];
  return `Invalid day of week: ${day}`;
}

function nameOfHour(hour: number): string {
  return `${String(hour).padStart(2, "0")}:00`;
}
</script>
