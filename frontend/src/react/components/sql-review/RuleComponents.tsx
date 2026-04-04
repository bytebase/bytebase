import { CircleHelpIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { PayloadValueType } from "@/components/SQLReview/components/RuleConfigComponents/types";
import { getRulePayload } from "@/components/SQLReview/components/RuleConfigComponents/utils";
import { payloadValueListToComponentList } from "@/components/SQLReview/components/utils";
import { t as vueT, te as vueTe } from "@/plugins/i18n";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
import type {
  RuleConfigComponent,
  RuleTemplateV2,
  TemplatePayload,
} from "@/types/sqlReview";
import {
  getRuleLocalization,
  getRuleLocalizationKey,
  ruleTypeToString,
} from "@/types/sqlReview";

// ---- Types ----

export interface RuleListWithCategory {
  value: string;
  label: string;
  ruleList: RuleTemplateV2[];
}

// ---- RuleLevelSwitch ----

interface RuleLevelSwitchProps {
  level: SQLReviewRule_Level;
  disabled?: boolean;
  editable?: boolean;
  onLevelChange: (level: SQLReviewRule_Level) => void;
}

export function RuleLevelSwitch({
  level,
  disabled,
  editable = true,
  onLevelChange,
}: RuleLevelSwitchProps) {
  const levelOptions = [
    { level: SQLReviewRule_Level.ERROR, label: vueT("sql-review.level.error") },
    {
      level: SQLReviewRule_Level.WARNING,
      label: vueT("sql-review.level.warning"),
    },
  ];

  const base =
    "py-1 w-[4.5rem] whitespace-nowrap border border-control-border text-control font-medium text-sm";

  const activeClass = (opt: SQLReviewRule_Level) => {
    if (opt === SQLReviewRule_Level.ERROR) {
      return "bg-red-100 text-red-800 border-red-800";
    }
    return "bg-yellow-100 text-yellow-800 border-yellow-800";
  };

  const filtered = editable
    ? levelOptions
    : levelOptions.filter((o) => o.level === level);

  return (
    <div className="inline-flex">
      {filtered.map((opt, i) => (
        <button
          key={opt.level}
          type="button"
          disabled={disabled}
          className={`${base} ${i === 0 ? "rounded-l" : "-ml-px"} ${i === filtered.length - 1 ? "rounded-r" : ""} ${
            level === opt.level ? activeClass(opt.level) : ""
          } ${disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"}`}
          onClick={() => onLevelChange(opt.level)}
        >
          {opt.label}
        </button>
      ))}
    </div>
  );
}

// ---- RuleLevelBadge ----

interface RuleLevelBadgeProps {
  level: SQLReviewRule_Level;
  suffix?: string;
}

export function RuleLevelBadge({ level, suffix }: RuleLevelBadgeProps) {
  const variant =
    level === SQLReviewRule_Level.ERROR ? "destructive" : "warning";
  const label =
    level === SQLReviewRule_Level.ERROR
      ? vueT("sql-review.level.error")
      : vueT("sql-review.level.warning");

  return (
    <Badge variant={variant}>
      {label}
      {suffix && ` ${suffix}`}
    </Badge>
  );
}

// ---- RuleConfig ----

interface RuleConfigProps {
  rule: RuleTemplateV2;
  disabled: boolean;
  size: "small" | "medium";
  payloadRef?: React.MutableRefObject<PayloadValueType[]>;
}

function configTitle(
  rule: RuleTemplateV2,
  config: RuleConfigComponent
): string {
  const key = `sql-review.rule.${getRuleLocalizationKey(ruleTypeToString(rule.type))}.component.${config.key}.title`;
  return vueT(key);
}

function configTooltip(
  rule: RuleTemplateV2,
  config: RuleConfigComponent
): string {
  const key = `sql-review.rule.${getRuleLocalizationKey(ruleTypeToString(rule.type))}.component.${config.key}.tooltip`;
  return vueTe(key) ? vueT(key) : "";
}

export function RuleConfig({
  rule,
  disabled,
  size,
  payloadRef,
}: RuleConfigProps) {
  const [payload, setPayload] = useState<PayloadValueType[]>(() =>
    getRulePayload(rule)
  );

  useEffect(() => {
    setPayload(getRulePayload(rule));
  }, [rule.componentList]);

  // Expose payload to parent via ref
  useEffect(() => {
    if (payloadRef) {
      payloadRef.current = payload;
    }
  }, [payload, payloadRef]);

  const updatePayload = (index: number, value: PayloadValueType) => {
    setPayload((prev) => {
      const next = [...prev];
      next[index] = value;
      return next;
    });
  };

  return (
    <div className="flex flex-col gap-y-4">
      {rule.componentList.map((config, index) => (
        <div key={index} className="flex flex-col gap-y-1">
          {config.payload.type !== "BOOLEAN" && (
            <div className="flex items-center gap-x-1">
              <p
                className={`font-medium ${size !== "small" ? "text-lg text-control mb-2" : ""}`}
              >
                {configTitle(rule, config)}
              </p>
              {configTooltip(rule, config) && (
                <Tooltip content={configTooltip(rule, config)}>
                  <CircleHelpIcon className="w-4 h-4" />
                </Tooltip>
              )}
            </div>
          )}

          {config.payload.type === "STRING" && (
            <Input
              value={(payload[index] as string) ?? ""}
              disabled={disabled}
              placeholder={`${config.payload.default}`}
              onChange={(e) => updatePayload(index, e.target.value)}
            />
          )}

          {config.payload.type === "NUMBER" && (
            <input
              type="number"
              className="flex h-9 w-full rounded-xs border border-control-border bg-transparent px-3 py-1 text-sm text-main placeholder:text-control-placeholder focus:outline-hidden focus:ring-2 focus:ring-accent focus:border-accent disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50"
              value={(payload[index] as number) ?? 0}
              disabled={disabled}
              placeholder={`${config.payload.default}`}
              onChange={(e) => updatePayload(index, Number(e.target.value))}
            />
          )}

          {config.payload.type === "BOOLEAN" && (
            <label className="flex items-center gap-x-2">
              <input
                type="checkbox"
                checked={(payload[index] as boolean) ?? false}
                disabled={disabled}
                onChange={(e) => updatePayload(index, e.target.checked)}
              />
              <span>{configTitle(rule, config)}</span>
              {configTooltip(rule, config) && (
                <Tooltip content={configTooltip(rule, config)}>
                  <CircleHelpIcon className="w-4 h-4" />
                </Tooltip>
              )}
            </label>
          )}

          {config.payload.type === "STRING_ARRAY" &&
            Array.isArray(payload[index]) && (
              <StringArrayInput
                value={payload[index] as string[]}
                disabled={disabled}
                onChange={(val) => updatePayload(index, val)}
              />
            )}

          {config.payload.type === "TEMPLATE" && (
            <TemplateSelect
              ruleType={ruleTypeToString(rule.type)}
              config={config}
              value={(payload[index] as string) ?? ""}
              disabled={disabled}
              onChange={(val) => updatePayload(index, val)}
            />
          )}
        </div>
      ))}
    </div>
  );
}

// ---- StringArrayInput (internal) ----

function StringArrayInput({
  value,
  disabled,
  onChange,
}: {
  value: string[];
  disabled: boolean;
  onChange: (value: string[]) => void;
}) {
  const [inputValue, setInputValue] = useState("");

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      const trimmed = inputValue.trim();
      if (trimmed) {
        onChange([...value, trimmed]);
        setInputValue("");
      }
    }
  };

  const removeTag = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };

  return (
    <div>
      {!disabled && (
        <p className="text-sm text-control-placeholder mb-1">
          {vueT("sql-review.input-then-press-enter")}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-1">
        {value.map((tag, i) => (
          <span
            key={i}
            className="inline-flex items-center gap-1 rounded bg-control-bg px-2 py-0.5 text-sm"
          >
            {tag}
            {!disabled && (
              <button
                type="button"
                className="cursor-pointer hover:text-error"
                onClick={() => removeTag(i)}
              >
                <XIcon className="w-3 h-3" />
              </button>
            )}
          </span>
        ))}
        {!disabled && (
          <input
            className="min-w-[20rem] flex-1 border border-control-border rounded-xs px-2 py-1 text-sm focus:outline-hidden focus:ring-2 focus:ring-accent"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
          />
        )}
      </div>
    </div>
  );
}

// ---- TemplateSelect (internal) ----

function TemplateSelect({
  ruleType,
  config,
  value,
  disabled,
  onChange,
}: {
  ruleType: string;
  config: RuleConfigComponent;
  value: string;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  const payload = config.payload as TemplatePayload;
  const options = payload.templateList.map((id) => ({
    id,
    description: vueT(
      `sql-review.rule.${getRuleLocalizationKey(ruleType)}.component.${config.key}.template.${id}`
    ),
  }));

  return (
    <select
      className="flex h-9 w-full rounded-xs border border-control-border bg-transparent px-3 py-1 text-sm text-main focus:outline-hidden focus:ring-2 focus:ring-accent disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50"
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value)}
    >
      {options.map((opt) => (
        <option key={opt.id} value={opt.id}>
          {opt.description}
        </option>
      ))}
    </select>
  );
}

// ---- RuleEditDialog ----

interface RuleEditDialogProps {
  rule: RuleTemplateV2;
  disabled: boolean;
  onUpdateRule: (update: Partial<RuleTemplateV2>) => void;
  onCancel: () => void;
}

export function RuleEditDialog({
  rule,
  disabled,
  onUpdateRule,
  onCancel,
}: RuleEditDialogProps) {
  const { t } = useTranslation();
  const [level, setLevel] = useState(rule.level);
  const payloadRef = useRef<PayloadValueType[]>(getRulePayload(rule));
  const localization = useMemo(
    () => getRuleLocalization(ruleTypeToString(rule.type), rule.engine),
    [rule.type, rule.engine]
  );

  const handleConfirm = () => {
    const componentList = payloadValueListToComponentList(
      rule,
      payloadRef.current
    );
    onUpdateRule({ level, componentList });
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent>
        <div className="p-6 flex flex-col gap-y-4">
          <div className="flex items-center justify-between">
            <DialogTitle>{localization.title}</DialogTitle>
            <span className="text-sm text-control-placeholder">
              {Engine[rule.engine]}
            </span>
          </div>

          <RuleLevelSwitch
            level={level}
            disabled={disabled}
            editable={!disabled}
            onLevelChange={setLevel}
          />

          {localization.description && (
            <p className="text-sm text-control-placeholder">
              {localization.description}
            </p>
          )}

          {rule.componentList.length > 0 && (
            <RuleConfig
              rule={rule}
              disabled={disabled}
              size="medium"
              payloadRef={payloadRef}
            />
          )}

          <div className="flex justify-end gap-x-2">
            <Button variant="outline" onClick={onCancel}>
              {t("common.cancel")}
            </Button>
            <Button disabled={disabled} onClick={handleConfirm}>
              {t("common.confirm")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ---- RuleLevelFilter ----

interface RuleLevelFilterProps {
  ruleList: RuleListWithCategory[];
  isCheckedLevel: (level: SQLReviewRule_Level) => boolean;
  onToggleCheckedLevel: (level: SQLReviewRule_Level) => void;
}

export function RuleLevelFilter({
  ruleList,
  isCheckedLevel,
  onToggleCheckedLevel,
}: RuleLevelFilterProps) {
  const countByLevel = useMemo(() => {
    const counts = new Map<SQLReviewRule_Level, number>();
    for (const category of ruleList) {
      for (const rule of category.ruleList) {
        counts.set(rule.level, (counts.get(rule.level) ?? 0) + 1);
      }
    }
    return counts;
  }, [ruleList]);

  const levels = [
    SQLReviewRule_Level.ERROR,
    SQLReviewRule_Level.WARNING,
  ] as const;

  return (
    <div className="flex items-center gap-x-4">
      {levels.map((level) => {
        const count = countByLevel.get(level) ?? 0;
        if (count === 0) return null;
        return (
          <label
            key={level}
            className="flex items-center gap-x-2 cursor-pointer"
          >
            <input
              type="checkbox"
              checked={isCheckedLevel(level)}
              onChange={() => onToggleCheckedLevel(level)}
            />
            <RuleLevelBadge level={level} suffix={`(${count})`} />
          </label>
        );
      })}
    </div>
  );
}
