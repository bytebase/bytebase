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
  matchPresetKey,
  resolveAnnouncementTheme,
} from "@/react/components/announcement-theme";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { ColorInput } from "@/react/components/ui/color-input";
import {
  FormControlGroup,
  FormControlRow,
  FormField,
  FormFieldGroup,
  FormLabel,
  FormSection,
} from "@/react/components/ui/form";
import { Input } from "@/react/components/ui/input";
import { SegmentedControl } from "@/react/components/ui/segmented-control";
import { usePlanFeature } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import {
  Announcement_AnnouncementThemeSchema,
  AnnouncementSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hexToColor } from "@/utils";
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
          theme: create(Announcement_AnnouncementThemeSchema, {
            background: hexToColor(state.theme.background),
            text: hexToColor(state.theme.text),
          }),
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
    <FormSection
      id="announcement"
      title={
        <span className="inline-flex items-center gap-x-2">
          {title}
          <FeatureBadge
            feature={PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT}
            clickable
          />
        </span>
      }
    >
      <PermissionGuard
        permissions={["bb.settings.setWorkspaceProfile"]}
        display="block"
      >
        <FormFieldGroup>
          {/* Theme selector */}
          <FormField
            className="gap-y-3"
            title={t("settings.general.workspace.announcement-theme.self")}
            description={t(
              "settings.general.workspace.announcement-theme.description"
            )}
          >
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
              <FormControlGroup>
                <FormControlRow>
                  <FormLabel
                    className="w-28 text-sm text-control"
                    htmlFor="announcement-theme-background"
                  >
                    {t(
                      "settings.general.workspace.announcement-theme.background"
                    )}
                  </FormLabel>
                  <ColorInput
                    id="announcement-theme-background"
                    value={state.theme.background}
                    disabled={disabled}
                    ariaLabel={t(
                      "settings.general.workspace.announcement-theme.background"
                    )}
                    onChange={(hex) =>
                      setState((s) => ({
                        ...s,
                        theme: { ...s.theme, background: hex },
                      }))
                    }
                  />
                </FormControlRow>
                <FormControlRow>
                  <FormLabel
                    className="w-28 text-sm text-control"
                    htmlFor="announcement-theme-text"
                  >
                    {t("settings.general.workspace.announcement-theme.text")}
                  </FormLabel>
                  <ColorInput
                    id="announcement-theme-text"
                    value={state.theme.text}
                    disabled={disabled}
                    ariaLabel={t(
                      "settings.general.workspace.announcement-theme.text"
                    )}
                    onChange={(hex) =>
                      setState((s) => ({
                        ...s,
                        theme: { ...s.theme, text: hex },
                      }))
                    }
                  />
                </FormControlRow>
              </FormControlGroup>
            )}
          </FormField>

          <FormField className="gap-y-2">
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
          </FormField>

          {/* Announcement text */}
          <FormField
            title={t("settings.general.workspace.announcement-text.self")}
            description={t(
              "settings.general.workspace.announcement-text.description"
            )}
          >
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
          </FormField>

          {/* Extra link */}
          <FormField title={t("settings.general.workspace.extra-link.self")}>
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
          </FormField>
        </FormFieldGroup>
      </PermissionGuard>
    </FormSection>
  );
});
