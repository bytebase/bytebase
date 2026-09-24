import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { SwitchRow } from "./SwitchRow";
import {
  isStandardReviewOn,
  STANDARD_RULE_TYPES,
  standardRuleText,
} from "./standardRules";

interface StandardRuleSwitchesProps {
  // The rules switched on.
  rules: readonly ReviewRuleType[];
  onChange: (rules: ReviewRuleType[]) => void;
  disabled?: boolean;
  // Shows each rule's state as text instead of a switch, for rules that are
  // set elsewhere.
  readOnly?: boolean;
}

export function StandardRuleSwitches({
  rules,
  onChange,
  disabled = false,
  readOnly = false,
}: StandardRuleSwitchesProps) {
  const { t } = useTranslation();
  const reviewOn = isStandardReviewOn(rules);

  // SYNTAX gates the rest, and the backend reads a list without it as empty,
  // so switching it off saves an empty list. Switching it back on restores
  // the rules from before it went off in this session, else every rule.
  const rulesWithSyntax = useRef<readonly ReviewRuleType[] | undefined>(
    undefined
  );
  if (reviewOn) {
    rulesWithSyntax.current = rules;
  }
  const toggle = (rule: ReviewRuleType, on: boolean) => {
    if (rule === ReviewRuleType.SYNTAX) {
      onChange(on ? [...(rulesWithSyntax.current ?? STANDARD_RULE_TYPES)] : []);
      return;
    }
    onChange(
      STANDARD_RULE_TYPES.filter((type) =>
        type === rule ? on : rules.includes(type)
      )
    );
  };

  return (
    <div className="rounded-sm border border-block-border">
      {/* The status line keeps its place whether or not SYNTAX is on, so
          switching it does not move the rules below. */}
      <div
        role="status"
        className={cn(
          "flex items-center gap-x-2 border-b px-4 py-2 text-sm",
          reviewOn
            ? "border-block-border bg-control-bg text-control-light"
            : "border-warning/40 bg-warning/10 text-warning"
        )}
      >
        <span
          className={cn(
            "size-1.5 shrink-0 rounded-full",
            reviewOn ? "bg-accent" : "bg-warning"
          )}
        />
        {reviewOn
          ? t("sql-review.standard-rules.running", {
              on: rules.length,
              total: STANDARD_RULE_TYPES.length,
            })
          : t("sql-review.standard-rules.syntax-off")}
      </div>
      <div className="divide-y divide-block-border">
        {STANDARD_RULE_TYPES.map((rule) => {
          const text = standardRuleText(t, rule);
          return (
            <SwitchRow
              key={rule}
              className="px-4 py-3"
              title={text.title}
              description={text.description}
              checked={rules.includes(rule)}
              onCheckedChange={(on) => toggle(rule, on)}
              disabled={
                disabled || (rule !== ReviewRuleType.SYNTAX && !reviewOn)
              }
              readOnly={readOnly}
            />
          );
        })}
      </div>
    </div>
  );
}
