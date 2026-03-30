import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { SparklesIcon } from "lucide-react";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { useSubscriptionV1Store } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  Announcement_AlertLevel,
  AnnouncementSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { SectionHandle } from "./useSettingSection";

interface AnnouncementSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface AnnouncementState {
  level: Announcement_AlertLevel;
  text: string;
  link: string;
}

const ALERT_LEVELS = [
  Announcement_AlertLevel.INFO,
  Announcement_AlertLevel.WARNING,
  Announcement_AlertLevel.CRITICAL,
] as const;

export const AnnouncementSection = forwardRef<
  SectionHandle,
  AnnouncementSectionProps
>(function AnnouncementSection({ title, onDirtyChange }, ref) {
  const { t } = useTranslation();
  const settingV1Store = useSettingV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const hasFeature = useVueState(() =>
    subscriptionStore.hasFeature(PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT)
  );

  const canEdit = hasWorkspacePermissionV2("bb.settings.setWorkspaceProfile");

  const getRawAnnouncement = useCallback((): AnnouncementState => {
    const announcement = settingV1Store.workspaceProfile.announcement;
    if (announcement) {
      return {
        level: announcement.level,
        text: announcement.text,
        link: announcement.link,
      };
    }
    return {
      level: Announcement_AlertLevel.INFO,
      text: "",
      link: "",
    };
  }, [settingV1Store]);

  const [state, setState] = useState<AnnouncementState>(() =>
    cloneDeep(getRawAnnouncement())
  );

  const isDirty = useCallback(
    () => !isEqual(state, getRawAnnouncement()),
    [state, getRawAnnouncement]
  );

  const revert = useCallback(() => {
    setState(cloneDeep(getRawAnnouncement()));
  }, [getRawAnnouncement]);

  const update = useCallback(async () => {
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        announcement: create(AnnouncementSchema, { ...state }),
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.announcement"],
      }),
    });
  }, [state, settingV1Store]);

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  useEffect(() => {
    onDirtyChange();
  }, [state, onDirtyChange]);

  const disabled = !canEdit || !hasFeature;

  const minimumPlan = useVueState(() =>
    subscriptionStore.getMinimumRequiredPlan(
      PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT
    )
  );

  return (
    <div id="announcement" className="py-6 lg:flex">
      <div className="text-left lg:w-1/4">
        <div className="flex items-center gap-x-2">
          <h1 className="text-2xl font-bold">{title}</h1>
          {!hasFeature && (
            <a
              href="/setting/subscription"
              className="text-accent"
              title={t("subscription.require-subscription", {
                requiredPlan: t(
                  `subscription.plan.${PlanType[minimumPlan].toLowerCase()}.title`
                ),
              })}
            >
              <SparklesIcon className="w-5 h-5" />
            </a>
          )}
        </div>
      </div>
      <div className="flex-1 lg:px-5">
        <div className="mt-5 lg:mt-0">
          {/* Alert level radio */}
          <label className="flex items-center gap-x-2">
            <span className="font-medium">
              {t(
                "settings.general.workspace.announcement-alert-level.description"
              )}
            </span>
          </label>
          <div className="flex flex-wrap py-2 gap-4">
            {ALERT_LEVELS.map((level) => (
              <label
                key={level}
                className="flex items-center gap-x-2 cursor-pointer"
              >
                <input
                  type="radio"
                  name="announcementLevel"
                  disabled={disabled}
                  checked={state.level === level}
                  onChange={() => setState((s) => ({ ...s, level }))}
                />
                <span>
                  {level === Announcement_AlertLevel.INFO &&
                    t(
                      "settings.general.workspace.announcement-alert-level.field.info"
                    )}
                  {level === Announcement_AlertLevel.WARNING &&
                    t(
                      "settings.general.workspace.announcement-alert-level.field.warning"
                    )}
                  {level === Announcement_AlertLevel.CRITICAL &&
                    t(
                      "settings.general.workspace.announcement-alert-level.field.critical"
                    )}
                </span>
              </label>
            ))}
          </div>

          {/* Announcement text */}
          <label className="flex items-center mt-2 gap-x-2">
            <span className="font-medium">
              {t("settings.general.workspace.announcement-text.self")}
            </span>
          </label>
          <div className="mb-3 text-sm text-gray-400">
            {t("settings.general.workspace.announcement-text.description")}
          </div>
          <Input
            value={state.text}
            className="mb-3 w-full"
            placeholder={t(
              "settings.general.workspace.announcement-text.placeholder"
            )}
            disabled={disabled}
            onChange={(e) => setState((s) => ({ ...s, text: e.target.value }))}
          />

          {/* Extra link */}
          <label className="flex items-center py-2 gap-x-2">
            <span className="font-medium">
              {t("settings.general.workspace.extra-link.self")}
            </span>
          </label>
          <Input
            value={state.link}
            className="mb-5 w-full"
            placeholder={t("settings.general.workspace.extra-link.placeholder")}
            disabled={disabled}
            onChange={(e) => setState((s) => ({ ...s, link: e.target.value }))}
          />
        </div>
      </div>
    </div>
  );
});
