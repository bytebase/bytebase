import { BackupSetting } from "@/types";
import { BackupPlanSchedule } from "@/types/proto/v1/org_policy_service";

export type BackupSettingEdit = Pick<
  BackupSetting,
  "enabled" | "dayOfWeek" | "hour" | "retentionPeriodTs"
>;

export const PLAN_SCHEDULES: BackupPlanSchedule[] = [
  BackupPlanSchedule.UNSET,
  BackupPlanSchedule.WEEKLY,
  BackupPlanSchedule.DAILY,
];

export const AVAILABLE_DAYS_OF_WEEK = [...Array(7).keys()]; // [0...6]
export const AVAILABLE_HOURS_OF_DAY = [...Array(24).keys()]; // [0...23]

export const DEFAULT_BACKUP_RETENTION_PERIOD_DAYS = 7;
export const DEFAULT_BACKUP_RETENTION_PERIOD_TS =
  DEFAULT_BACKUP_RETENTION_PERIOD_DAYS * 3600 * 24; // 7 days

export function parseScheduleFromBackupSetting(
  backupSetting: BackupSettingEdit
): BackupPlanSchedule {
  if (!backupSetting.enabled) return BackupPlanSchedule.UNSET;
  if (backupSetting.dayOfWeek === -1) return BackupPlanSchedule.DAILY;
  return BackupPlanSchedule.WEEKLY;
}

export function levelOfSchedule(schedule: BackupPlanSchedule) {
  return PLAN_SCHEDULES.indexOf(schedule) || 0;
}

export function localToUTC(hour: number, dayOfWeek: number) {
  return alignUTC(hour, dayOfWeek, new Date().getTimezoneOffset() * 60);
}

export function localFromUTC(hour: number, dayOfWeek: number) {
  return alignUTC(hour, dayOfWeek, -new Date().getTimezoneOffset() * 60);
}

export function alignUTC(
  hour: number,
  dayOfWeek: number,
  offsetInSecond: number
) {
  if (hour != -1) {
    hour = hour + offsetInSecond / 60 / 60;
    let dayOffset = 0;
    if (hour > 23) {
      hour = hour - 24;
      dayOffset = 1;
    }
    if (hour < 0) {
      hour = hour + 24;
      dayOffset = -1;
    }
    if (dayOfWeek != -1) {
      dayOfWeek = (7 + dayOfWeek + dayOffset) % 7;
    }
  }
  return { hour, dayOfWeek };
}
