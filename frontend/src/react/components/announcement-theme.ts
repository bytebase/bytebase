import type { Announcement } from "@/types/proto-es/v1/setting_service_pb";
import { colorToHex } from "@/utils";

export interface AnnouncementTheme {
  background: string;
  text: string;
}

export type AnnouncementPresetKey = "info" | "warning" | "critical";

/**
 * Built-in presets are a frontend-only concept that seed the two banner colors.
 * Each value is a #rrggbb hex color. The backgrounds match `--color-info` /
 * `--color-warning` / `--color-error`, all with white text.
 */
export const ANNOUNCEMENT_PRESETS: Record<
  AnnouncementPresetKey,
  AnnouncementTheme
> = {
  info: { background: "#2563eb", text: "#ffffff" },
  warning: { background: "#f59e0b", text: "#ffffff" },
  critical: { background: "#dc2626", text: "#ffffff" },
};

export const ANNOUNCEMENT_PRESET_KEYS: AnnouncementPresetKey[] = [
  "info",
  "warning",
  "critical",
];

/**
 * Resolves the banner theme for an announcement.
 *
 * Prefers the stored `theme`, defaulting to the `info` preset.
 */
export function resolveAnnouncementTheme(
  announcement: Pick<Announcement, "theme"> | undefined
): AnnouncementTheme {
  if (announcement?.theme?.background) {
    return {
      background: colorToHex(announcement.theme.background),
      text: announcement.theme.text
        ? colorToHex(announcement.theme.text)
        : ANNOUNCEMENT_PRESETS.info.text,
    };
  }
  return ANNOUNCEMENT_PRESETS.info;
}

/**
 * Returns the preset key whose theme exactly match the given theme, or
 * `"custom"` when no preset matches. Used by the admin UI to highlight the
 * selected option.
 */
export function matchPresetKey(
  theme: AnnouncementTheme
): AnnouncementPresetKey | "custom" {
  for (const key of ANNOUNCEMENT_PRESET_KEYS) {
    const preset = ANNOUNCEMENT_PRESETS[key];
    if (preset.background === theme.background && preset.text === theme.text) {
      return key;
    }
  }
  return "custom";
}
