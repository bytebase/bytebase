import { useTranslation } from "react-i18next";
import type { EditorThemeOption } from "@/react/components/monaco/editorThemes";
import type { ThemeAnchors } from "@/react/components/sql-editor/theme/derive";
import { ColorInput } from "@/react/components/ui/color-input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";

interface ThemeAnchorEditorProps {
  value: ThemeAnchors;
  /** The chosen editor theme id (`monacoBase`). */
  editorTheme: string;
  /** Editor color themes registered with the VSCode theme service. */
  editorThemes: EditorThemeOption[];
  disabled?: boolean;
  onChange: (next: ThemeAnchors) => void;
  onEditorThemeChange: (id: string) => void;
}

const ANCHOR_KEYS: (keyof ThemeAnchors)[] = [
  "background",
  "surface",
  "text",
  "accent",
  "border",
];

// Friendly labels for the standalone-alias ids, in case a stored theme still
// references one (they aren't part of the enumerated list).
const ALIAS_LABELS: Record<string, string> = { vs: "Light", "vs-dark": "Dark" };

export function ThemeAnchorEditor({
  value,
  editorTheme,
  editorThemes,
  disabled,
  onChange,
  onEditorThemeChange,
}: ThemeAnchorEditorProps) {
  const { t } = useTranslation();

  // Ensure the current value is always representable so the trigger never shows
  // a bare id (e.g. a legacy "vs" that isn't in the enumerated list).
  const themeOptions = editorThemes.some((option) => option.id === editorTheme)
    ? editorThemes
    : [
        {
          id: editorTheme,
          label: ALIAS_LABELS[editorTheme] ?? editorTheme,
          type: "light" as const,
        },
        ...editorThemes,
      ];

  return (
    <div className="flex flex-col gap-y-3">
      {ANCHOR_KEYS.map((key) => (
        <div key={key} className="flex items-center gap-x-3">
          <label
            className="w-28 text-sm text-control"
            htmlFor={`anchor-${key}`}
          >
            {t(`settings.general.workspace.sql-editor-theme.anchor.${key}`)}
          </label>
          <ColorInput
            id={`anchor-${key}`}
            value={value[key]}
            disabled={disabled}
            ariaLabel={t(
              `settings.general.workspace.sql-editor-theme.anchor.${key}`
            )}
            onChange={(hex) => onChange({ ...value, [key]: hex })}
          />
        </div>
      ))}

      {/* Editor syntax theme — the VSCode color theme applied to the code
          editor. Data-driven from the themes the theme service has registered;
          the chrome background still shows through the transparent canvas. */}
      <div className="flex items-center gap-x-3">
        <span className="w-28 text-sm text-control">
          {t("settings.general.workspace.sql-editor-theme.editor-theme")}
        </span>
        <Select
          value={editorTheme}
          onValueChange={(id) => {
            if (typeof id === "string") onEditorThemeChange(id);
          }}
        >
          <SelectTrigger size="sm" className="w-60" disabled={disabled}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {themeOptions.map((option) => (
              <SelectItem key={option.id} value={option.id}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
