import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { AnnouncementBanner } from "@/react/components/AnnouncementBanner";
import {
  ANNOUNCEMENT_PRESET_KEYS,
  ANNOUNCEMENT_PRESETS,
  type AnnouncementTheme,
  hexToTriple,
  matchPresetKey,
  resolveAnnouncementTheme,
  tripleToHex,
} from "@/react/components/announcement-theme";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { ColorInput } from "@/react/components/ui/color-input";
import { Input } from "@/react/components/ui/input";
import { SegmentedControl } from "@/react/components/ui/segmented-control";
import { usePlanFeature } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import {
  Announcement_AnnouncementThemeSchema,
  AnnouncementSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SectionHandle } from "./useSettingSection";

interface AnnouncementSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface AnnouncementState {
  theme: AnnouncementTheme;
  text: string;
  link: string;
}

const CUSTOM_THEME_OPTION = "custom";

const THEME_OPTIONS = [
  ...ANNOUNCEMENT_PRESET_KEYS,
  CUSTOM_THEME_OPTION,
] as const;

type ThemeOption = (typeof THEME_OPTIONS)[number];

export const AnnouncementSection = forwardRef<
  SectionHandle,
  AnnouncementSectionProps
>(function AnnouncementSection({ title, onDirtyChange }, ref) {
  const { t } = useTranslation();

  const hasFeature = usePlanFeature(PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT);

  const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

  const getRawAnnouncement = useCallback((): AnnouncementState => {
    const announcement = useAppStore
      .getState()
      .getWorkspaceProfile().announcement;
    return {
      theme: resolveAnnouncementTheme(announcement),
      text: announcement?.text ?? "",
      link: announcement?.link ?? "",
    };
  }, []);

  const [state, setState] = useState<AnnouncementState>(() =>
    cloneDeep(getRawAnnouncement())
  );

  // UI-only: keep "Custom" sticky even when the edited theme happens to match a
  // preset. Not persisted — the store only holds the resolved theme.
  const [customSelected, setCustomSelected] = useState(
    () => matchPresetKey(getRawAnnouncement().theme) === "custom"
  );

  const isDirty = useCallback(
    () => !isEqual(state, getRawAnnouncement()),
    [state, getRawAnnouncement]
  );

  const revert = useCallback(() => {
    const raw = getRawAnnouncement();
    setState(cloneDeep(raw));
    setCustomSelected(matchPresetKey(raw.theme) === "custom");
  }, [getRawAnnouncement]);

  const update = useCallback(async () => {
    await useAppStore.getState().updateWorkspaceProfile({
      payload: {
        announcement: create(AnnouncementSchema, {
          text: state.text,
          link: state.link,
          theme: create(Announcement_AnnouncementThemeSchema, state.theme),
        }),
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.announcement"],
      }),
    });
  }, [state]);

  useImperativeHandle(ref, () => ({ isDirty, revert, update }));

  const selectedTheme: ThemeOption =
    customSelected || matchPresetKey(state.theme) === "custom"
      ? "custom"
      : matchPresetKey(state.theme);

  const onSelectTheme = useCallback((option: ThemeOption) => {
    if (option === "custom") {
      setCustomSelected(true);
      return;
    }
    setCustomSelected(false);
    setState((s) => ({ ...s, theme: { ...ANNOUNCEMENT_PRESETS[option] } }));
  }, []);

  useEffect(() => {
    onDirtyChange();
  }, [state, onDirtyChange]);

  const disabled = !canEdit || !hasFeature;

  const themeOptionLabel = (option: ThemeOption): string => {
    switch (option) {
      case "info":
        return t(
          "settings.general.workspace.announcement-alert-level.field.info"
        );
      case "warning":
        return t(
          "settings.general.workspace.announcement-alert-level.field.warning"
        );
      case "critical":
        return t(
          "settings.general.workspace.announcement-alert-level.field.critical"
        );
      default:
        return t("settings.general.workspace.announcement-theme.custom");
    }
  };

  const previewText =
    state.text || t("settings.general.workspace.announcement-text.placeholder");

  return (
    <div id="announcement" className="py-6 lg:flex">
      <div className="text-left lg:w-1/4">
        <div className="flex items-center gap-x-2">
          <h1 className="text-2xl font-bold">{title}</h1>
          <FeatureBadge
            feature={PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT}
            clickable
          />
        </div>
      </div>
      <PermissionGuard
        permissions={["bb.settings.setWorkspaceProfile"]}
        display="block"
      >
        <div className="flex-1 lg:px-5">
          <div className="mt-5 flex flex-col gap-y-6 lg:mt-0">
            {/* Theme selector */}
            <div className="flex flex-col gap-y-3">
              <div>
                <p className="text-base font-semibold">
                  {t("settings.general.workspace.announcement-theme.self")}
                </p>
                <p className="mt-1 text-sm text-gray-400">
                  {t(
                    "settings.general.workspace.announcement-theme.description"
                  )}
                </p>
              </div>

              <SegmentedControl
                ariaLabel={t(
                  "settings.general.workspace.announcement-theme.self"
                )}
                disabled={disabled}
                value={selectedTheme}
                onValueChange={onSelectTheme}
                options={THEME_OPTIONS.map((option) => ({
                  value: option,
                  label: themeOptionLabel(option),
                }))}
              />

              {selectedTheme === "custom" && (
                <div className="flex flex-col gap-y-3">
                  <div className="flex items-center gap-x-3">
                    <label
                      className="w-28 text-sm text-control"
                      htmlFor="announcement-theme-background"
                    >
                      {t(
                        "settings.general.workspace.announcement-theme.background"
                      )}
                    </label>
                    <ColorInput
                      id="announcement-theme-background"
                      value={tripleToHex(state.theme.background)}
                      disabled={disabled}
                      ariaLabel={t(
                        "settings.general.workspace.announcement-theme.background"
                      )}
                      onChange={(hex) =>
                        setState((s) => ({
                          ...s,
                          theme: { ...s.theme, background: hexToTriple(hex) },
                        }))
                      }
                    />
                  </div>
                  <div className="flex items-center gap-x-3">
                    <label
                      className="w-28 text-sm text-control"
                      htmlFor="announcement-theme-text"
                    >
                      {t("settings.general.workspace.announcement-theme.text")}
                    </label>
                    <ColorInput
                      id="announcement-theme-text"
                      value={tripleToHex(state.theme.text)}
                      disabled={disabled}
                      ariaLabel={t(
                        "settings.general.workspace.announcement-theme.text"
                      )}
                      onChange={(hex) =>
                        setState((s) => ({
                          ...s,
                          theme: { ...s.theme, text: hexToTriple(hex) },
                        }))
                      }
                    />
                  </div>
                </div>
              )}

              <div className="flex flex-col gap-y-2">
                <p className="text-sm font-medium text-control">
                  {t("common.preview")}
                </p>
                <AnnouncementBanner
                  text={previewText}
                  link={state.link}
                  background={state.theme.background}
                  textColor={state.theme.text}
                  interactive={false}
                  className="rounded-xs"
                />
              </div>
            </div>

            {/* Announcement text */}
            <div>
              <p className="text-base font-semibold">
                {t("settings.general.workspace.announcement-text.self")}
              </p>
              <p className="mb-3 text-sm text-gray-400">
                {t("settings.general.workspace.announcement-text.description")}
              </p>
              <Input
                value={state.text}
                className="w-full"
                placeholder={t(
                  "settings.general.workspace.announcement-text.placeholder"
                )}
                disabled={disabled}
                onChange={(e) =>
                  setState((s) => ({ ...s, text: e.target.value }))
                }
              />
            </div>

            {/* Extra link */}
            <div>
              <p className="mb-2 text-base font-semibold">
                {t("settings.general.workspace.extra-link.self")}
              </p>
              <Input
                value={state.link}
                className="w-full"
                placeholder={t(
                  "settings.general.workspace.extra-link.placeholder"
                )}
                disabled={disabled}
                onChange={(e) =>
                  setState((s) => ({ ...s, link: e.target.value }))
                }
              />
            </div>
          </div>
        </div>
      </PermissionGuard>
    </div>
  );
});
