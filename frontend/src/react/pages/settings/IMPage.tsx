import { Plus, Trash2 } from "lucide-react";
import { type ChangeEvent, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
// Static image imports — Vite cannot resolve dynamic src in <img>
import dingtalkIcon from "@/assets/im/dingtalk.png";
import feishuIcon from "@/assets/im/feishu.webp";
import slackIcon from "@/assets/im/slack.png";
import teamsIcon from "@/assets/im/teams.svg";
import wecomIcon from "@/assets/im/wecom.png";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";

const IM_ICON_MAP: Record<string, string> = {
  SLACK: slackIcon,
  FEISHU: feishuIcon,
  LARK: feishuIcon,
  WECOM: wecomIcon,
  DINGTALK: dingtalkIcon,
  TEAMS: teamsIcon,
};

export interface IMFieldDef {
  key: string;
  label: string;
}

export interface IMSettingItem {
  type: string;
  typeLabel: string;
  isConfigured: boolean;
  fields: IMFieldDef[];
  values: Record<string, string>;
}

export interface IMPageProps {
  settings: IMSettingItem[];
  availableTypes: { type: string; label: string }[];
  allowEdit: boolean;
  pendingSaveType: string | null;
  onAdd: (type: string) => void;
  onSave: (
    index: number,
    type: string,
    values: Record<string, string>
  ) => Promise<void>;
  onDelete: (index: number, type: string) => Promise<void>;
  onChangeType: (index: number, newType: string) => void;
}

export function IMPage({
  settings,
  availableTypes,
  allowEdit,
  pendingSaveType,
  onAdd,
  onSave,
  onDelete,
  onChangeType,
}: IMPageProps) {
  const { t } = useTranslation();

  // Local form state — tracks edits independently from props
  const [localValues, setLocalValues] = useState<Record<string, string>[]>([]);

  // Sync local state when settings change from outside (initial load, after save/delete)
  useEffect(() => {
    setLocalValues(settings.map((s) => ({ ...s.values })));
  }, [settings]);

  const updateField = useCallback(
    (index: number, key: string, value: string) => {
      setLocalValues((prev) => {
        const next = [...prev];
        next[index] = { ...next[index], [key]: value };
        return next;
      });
    },
    []
  );

  const isDirty = useCallback(
    (index: number) => {
      const local = localValues[index];
      const original = settings[index]?.values;
      if (!local || !original) return false;
      return Object.keys(original).some((k) => local[k] !== original[k]);
    },
    [localValues, settings]
  );

  const handleDiscard = useCallback(
    (index: number) => {
      setLocalValues((prev) => {
        const next = [...prev];
        const original = settings[index];
        if (original) {
          next[index] = { ...original.values };
        }
        return next;
      });
    },
    [settings]
  );

  const handleDelete = useCallback(
    async (index: number, type: string) => {
      if (!window.confirm(t("bbkit.confirm-button.sure-to-delete"))) return;
      await onDelete(index, type);
    },
    [onDelete, t]
  );

  if (settings.length === 0) {
    return (
      <div className="w-full px-4 flex flex-col gap-y-4 py-4">
        <Description />
        <div className="py-12 border rounded-sm flex flex-col items-center justify-center gap-y-4 text-control-light">
          <span>{t("common.no-data")}</span>
          {availableTypes.length > 0 && allowEdit && (
            <Button onClick={() => onAdd(availableTypes[0].type)}>
              <Plus className="h-4 w-4 mr-1" />
              {t("settings.im.add-im-integration")}
            </Button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="w-full px-4 flex flex-col gap-y-4 py-4">
      <Description />

      {settings.map((item, i) => (
        <div key={item.type} className="border rounded-sm p-4">
          {/* Header: IM type */}
          {item.isConfigured ? (
            <IMTypeLabel type={item.type} label={item.typeLabel} />
          ) : (
            <select
              value={item.type}
              className="flex h-9 w-full max-w-xs rounded-md border border-control-border bg-transparent px-3 py-1 text-sm cursor-pointer"
              onChange={(e) => onChangeType(i, e.target.value)}
            >
              <option value={item.type}>{item.typeLabel}</option>
              {availableTypes
                .filter((opt) => opt.type !== item.type)
                .map((opt) => (
                  <option key={opt.type} value={opt.type}>
                    {opt.label}
                  </option>
                ))}
            </select>
          )}

          {/* Fields */}
          <div className="mt-4 flex flex-col gap-y-4">
            {item.fields.map((field) => (
              <div key={field.key}>
                <label className="textlabel">{field.label}</label>
                <Input
                  className="mt-2"
                  disabled={!allowEdit}
                  placeholder={t("common.sensitive-placeholder")}
                  value={localValues[i]?.[field.key] ?? ""}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    updateField(i, field.key, e.target.value)
                  }
                />
              </div>
            ))}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between mt-4 gap-x-2">
            <div>
              {item.isConfigured && allowEdit && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-error hover:text-error hover:bg-error/10"
                  onClick={() => handleDelete(i, item.type)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              )}
            </div>
            {isDirty(i) && (
              <div className="flex items-center gap-x-2">
                <Button
                  variant="outline"
                  disabled={!!pendingSaveType}
                  onClick={() => handleDiscard(i)}
                >
                  {t("common.discard-changes")}
                </Button>
                <Button
                  disabled={!!pendingSaveType || !allowEdit}
                  onClick={() =>
                    onSave(i, item.type, localValues[i] ?? item.values)
                  }
                >
                  {t("common.save")}
                </Button>
              </div>
            )}
          </div>
        </div>
      ))}

      {availableTypes.length > 0 && allowEdit && (
        <div className="flex justify-end">
          <Button
            variant="outline"
            onClick={() => onAdd(availableTypes[0].type)}
          >
            <Plus className="h-4 w-4 mr-1" />
            {t("settings.im.add-another-im")}
          </Button>
        </div>
      )}
    </div>
  );
}

function Description() {
  const { t } = useTranslation();
  return (
    <div className="textinfolabel">
      {t("settings.im-integration.description")}{" "}
      <a
        href="https://docs.bytebase.com/change-database/webhook?source=console"
        target="_blank"
        rel="noopener noreferrer"
        className="text-accent hover:underline"
      >
        {t("common.learn-more")} &gt;
      </a>
    </div>
  );
}

function IMTypeLabel({ type, label }: { type: string; label: string }) {
  const icon = IM_ICON_MAP[type];
  return (
    <div className="flex items-center gap-x-2">
      {icon && <img src={icon} alt={label} className="h-5 w-5" />}
      <span className="text-sm font-medium">{label}</span>
    </div>
  );
}
