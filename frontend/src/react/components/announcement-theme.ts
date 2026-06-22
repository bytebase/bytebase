import type { Announcement } from "@/types/proto-es/v1/setting_service_pb";
import { hexToRgb, rgbToHex } from "@/utils/css";

export interface AnnouncementTheme {
  background: string;
  text: string;
}

export type AnnouncementPresetKey = "info" | "warning" | "critical";

/**
 * Built-in presets are a frontend-only concept that seed the two banner colors.
 * Each value is an `"r g b"` triple. The backgrounds match
 * `--color-info` / `--color-warning` / `--color-error`, all with white text.
 */
export const ANNOUNCEMENT_PRESETS: Record<
  AnnouncementPresetKey,
  AnnouncementTheme
> = {
  info: { background: "37 99 235", text: "255 255 255" },
  warning: { background: "245 158 11", text: "255 255 255" },
  critical: { background: "220 38 38", text: "255 255 255" },
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
      background: announcement.theme.background,
      text: announcement.theme.text,
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

/** Converts an `"r g b"` triple to a `#rrggbb` hex string for color inputs. */
export function tripleToHex(triple: string): string {
  const parts = triple.trim().split(/\s+/).map(Number);
  const [r, g, b] = [parts[0] ?? 0, parts[1] ?? 0, parts[2] ?? 0];
  return rgbToHex(r, g, b);
}

/** Converts a `#rrggbb` hex string to an `"r g b"` triple for storage. */
export function hexToTriple(hex: string): string {
  return hexToRgb(hex).join(" ");
}
