import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, test, vi } from "vitest";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { StandardRuleSwitches } from "./StandardRuleSwitches";
import { STANDARD_RULE_TYPES } from "./standardRules";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

const ruleSwitch = (rule: string) =>
  screen.getByRole("switch", {
    name: `sql-review.standard-rules.rule.${rule}.title`,
  });

// Holds the rules the way a page does, so a sequence of clicks can be tested.
function Stateful({
  initial,
  onChange,
}: {
  initial: readonly ReviewRuleType[];
  onChange: (rules: ReviewRuleType[]) => void;
}) {
  const [rules, setRules] = useState<readonly ReviewRuleType[]>(initial);
  return (
    <StandardRuleSwitches
      rules={rules}
      onChange={(next) => {
        setRules(next);
        onChange(next);
      }}
    />
  );
}

const rowOf = (rule: string) =>
  ruleSwitch(rule).closest("[class*='justify-between']") as HTMLElement;

const codesOf = (row: HTMLElement) =>
  Array.from(row.querySelectorAll("[data-testid='review-code']")).map(
    (node) => node.textContent
  );

describe("StandardRuleSwitches", () => {
  test("lists the engines reviewed, each rule's codes, and the engines of a rule fewer engines check", () => {
    render(
      <StandardRuleSwitches rules={STANDARD_RULE_TYPES} onChange={vi.fn()} />
    );

    expect(screen.getByTestId("standard-rule-engines").textContent).toBe(
      "sql-review.standard-rules.supported-engines"
    );
    const where = rowOf("require-where");
    expect(codesOf(where)).toEqual(["WH101"]);
    expect(
      where.querySelector("[data-testid='standard-rule-rule-engines']")
    ).toBeNull();

    const online = rowOf("online-migration");
    expect(codesOf(online)[0]).toBe("OM101");
    expect(
      online.querySelector("[data-testid='standard-rule-rule-engines']")
        ?.textContent
    ).toBe("MySQL, MariaDB, OceanBase (MySQL)");
  });

  test("switches one rule and reports the list in display order", () => {
    const onChange = vi.fn();
    render(
      <StandardRuleSwitches
        rules={[ReviewRuleType.DISALLOW_RENAME, ReviewRuleType.SYNTAX]}
        onChange={onChange}
      />
    );

    expect(screen.getAllByRole("switch")).toHaveLength(
      STANDARD_RULE_TYPES.length
    );
    expect(
      screen.getByText("sql-review.standard-rules.running")
    ).toBeInTheDocument();
    expect(ruleSwitch("require-where")).toHaveAttribute(
      "aria-checked",
      "false"
    );
    fireEvent.click(ruleSwitch("require-where"));
    expect(onChange).toHaveBeenLastCalledWith([
      ReviewRuleType.SYNTAX,
      ReviewRuleType.REQUIRE_WHERE,
      ReviewRuleType.DISALLOW_RENAME,
    ]);

    fireEvent.click(ruleSwitch("disallow-rename"));
    expect(onChange).toHaveBeenLastCalledWith([ReviewRuleType.SYNTAX]);
  });

  test("turning SYNTAX off switches every rule off", () => {
    const onChange = vi.fn();
    render(
      <StandardRuleSwitches rules={STANDARD_RULE_TYPES} onChange={onChange} />
    );

    expect(
      screen.queryByText("sql-review.standard-rules.syntax-off")
    ).not.toBeInTheDocument();
    fireEvent.click(ruleSwitch("syntax"));
    expect(onChange).toHaveBeenLastCalledWith([]);
  });

  test("with nothing on, the status line explains it and the rest are locked", () => {
    const onChange = vi.fn();
    render(<StandardRuleSwitches rules={[]} onChange={onChange} />);

    expect(
      screen.getByText("sql-review.standard-rules.syntax-off")
    ).toBeInTheDocument();
    expect(
      screen.queryByText("sql-review.standard-rules.running")
    ).not.toBeInTheDocument();
    expect(ruleSwitch("syntax")).not.toHaveAttribute("data-disabled");
    expect(ruleSwitch("require-where")).toHaveAttribute("data-disabled");
    expect(ruleSwitch("require-where")).toHaveAttribute(
      "aria-checked",
      "false"
    );

    // Nothing to restore, so SYNTAX brings every rule back.
    fireEvent.click(ruleSwitch("syntax"));
    expect(onChange).toHaveBeenLastCalledWith([...STANDARD_RULE_TYPES]);
  });

  test("turning SYNTAX back on restores the rules from before", () => {
    const onChange = vi.fn();
    render(<Stateful initial={STANDARD_RULE_TYPES} onChange={onChange} />);

    fireEvent.click(ruleSwitch("disallow-truncate"));
    fireEvent.click(ruleSwitch("syntax"));
    expect(onChange).toHaveBeenLastCalledWith([]);
    expect(ruleSwitch("disallow-rename")).toHaveAttribute(
      "aria-checked",
      "false"
    );

    fireEvent.click(ruleSwitch("syntax"));
    expect(onChange).toHaveBeenLastCalledWith(
      STANDARD_RULE_TYPES.filter(
        (rule) => rule !== ReviewRuleType.DISALLOW_TRUNCATE
      )
    );
  });

  test("the read-only view does not replace the rules to restore", () => {
    const own = [ReviewRuleType.SYNTAX, ReviewRuleType.DISALLOW_RENAME];
    const inherited = [ReviewRuleType.SYNTAX, ReviewRuleType.REQUIRE_WHERE];
    const onChange = vi.fn();
    const { rerender } = render(
      <StandardRuleSwitches rules={own} onChange={onChange} />
    );
    rerender(<StandardRuleSwitches rules={[]} onChange={onChange} />);
    rerender(
      <StandardRuleSwitches rules={inherited} onChange={onChange} readOnly />
    );
    rerender(<StandardRuleSwitches rules={[]} onChange={onChange} />);

    fireEvent.click(ruleSwitch("syntax"));
    expect(onChange).toHaveBeenLastCalledWith(own);
  });

  test("disabled locks every switch", () => {
    render(
      <StandardRuleSwitches
        rules={STANDARD_RULE_TYPES}
        onChange={vi.fn()}
        disabled
      />
    );

    for (const control of screen.getAllByRole("switch")) {
      expect(control).toHaveAttribute("data-disabled");
    }
  });

  test("read-only shows each rule's state as text", () => {
    render(
      <StandardRuleSwitches
        rules={[ReviewRuleType.SYNTAX, ReviewRuleType.REQUIRE_WHERE]}
        onChange={vi.fn()}
        readOnly
      />
    );

    expect(screen.queryAllByRole("switch")).toHaveLength(0);
    expect(screen.getAllByText("sql-review.standard-rules.on")).toHaveLength(2);
    expect(screen.getAllByText("sql-review.standard-rules.off")).toHaveLength(
      STANDARD_RULE_TYPES.length - 2
    );
  });
});
