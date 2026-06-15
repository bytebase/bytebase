import { useTranslation } from "react-i18next";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import { PRESETS } from "./theme/presets";

// NOTE: preset names below are rendered as literal strings, NOT i18n keys, by
// design. This switcher is dev-only (mounted behind `isDev()`), so end users
// never see it; the names are largely proper nouns (e.g. "Solarized Dark") that
// aren't translated anyway; and future user-defined custom themes will carry
// arbitrary user-provided names. Only the switcher's own chrome label is
// localized (`sql-editor.theme.self`). See the theme design doc's as-shipped
// notes. (Intentional exception to the AGENTS.md "all UI text in locales" rule.)
export function ThemeSelect() {
  const { t } = useTranslation();
  const themeId = useSQLEditorEditorState((s) => s.themeId);

  return (
    <Select
      value={themeId}
      onValueChange={(value) => {
        if (typeof value === "string") {
          getSQLEditorEditorState().setThemeId(value);
        }
      }}
    >
      <SelectTrigger
        size="sm"
        className="h-8"
        aria-label={t("sql-editor.theme.self")}
      >
        {/* Render the preset's display name, not the raw id ("light"). */}
        <SelectValue>
          {(value) => {
            const preset = PRESETS.find((p) => p.id === value);
            return preset ? preset.name : String(value ?? "");
          }}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {PRESETS.map((preset) => (
          <SelectItem key={preset.id} value={preset.id}>
            {preset.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
