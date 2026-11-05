import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { engineNameV1 } from "@/utils/v1/instance";
import { FindingCodeBadge } from "./FindingCodeBadge";
import { SwitchRow } from "./SwitchRow";
import {
  RULE_REVIEW_ENGINES,
  standardRuleCodes,
  standardRuleEngines,
} from "./standardRuleCatalog";
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
  // the rules from before it went off in this session, else every rule. The
  // read-only view shows rules set elsewhere, so it does not count.
  const rulesWithSyntax = useRef<readonly ReviewRuleType[] | undefined>(
    undefined
  );
  if (reviewOn && !readOnly) {
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
      <div
        className="border-b border-block-border px-4 py-2 text-xs text-control-light"
        data-testid="standard-rule-engines"
      >
        {t("sql-review.standard-rules.supported-engines", {
          engines: RULE_REVIEW_ENGINES.map(engineNameV1).join(", "),
        })}
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
              details={<StandardRuleFacts rule={rule} />}
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

// A rule's finding codes, and its engines when not every reviewed engine
// checks it.
function StandardRuleFacts({ rule }: { rule: ReviewRuleType }) {
  const { t } = useTranslation();
  const codes = standardRuleCodes(rule);
  const engines = standardRuleEngines(rule);
  const partial = engines.length < RULE_REVIEW_ENGINES.length;
  if (codes.length === 0 && !partial) return null;
  return (
    <dl className="mt-1 flex flex-col gap-y-1 text-xs">
      {codes.length > 0 && (
        <div className="flex items-baseline gap-x-2">
          <dt className="shrink-0 text-control-light">
            {t("sql-review.standard-rules.codes")}
          </dt>
          <dd
            className="flex min-w-0 flex-wrap items-center gap-1"
            data-testid="standard-rule-codes"
          >
            {codes.map(({ code }) => (
              <FindingCodeBadge code={code} key={code} />
            ))}
          </dd>
        </div>
      )}
      {partial && (
        <div className="flex items-baseline gap-x-2">
          <dt className="shrink-0 text-control-light">
            {t("sql-review.standard-rules.engines")}
          </dt>
          <dd className="text-control" data-testid="standard-rule-rule-engines">
            {engines.map(engineNameV1).join(", ")}
          </dd>
        </div>
      )}
    </dl>
  );
}
