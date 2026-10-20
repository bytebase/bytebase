import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { isSetupGoal, type SetupGoal } from "./draft";
import {
  choiceCardClassName,
  SetupNote,
  SetupScreenFrame,
} from "./SetupScreenFrame";

export function GoalScreen({
  goal,
  onGoalChange,
  onBack,
  onContinue,
}: {
  readonly goal: SetupGoal | undefined;
  readonly onGoalChange: (goal: SetupGoal) => void;
  readonly onBack: () => void;
  readonly onContinue: () => void;
}) {
  const { t } = useTranslation();

  return (
    <SetupScreenFrame
      screen="goal"
      title={t("workspace-setup.goal.title")}
      description={t("workspace-setup.goal.description")}
      onSubmit={() => {
        if (goal) onContinue();
      }}
      footer={
        <>
          <Button type="button" appearance="secondary" onClick={onBack}>
            {t("common.back")}
          </Button>
          <Button type="submit" disabled={!goal}>
            {t("workspace-setup.continue")}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-y-2">
        <RadioGroup
          aria-label={t("workspace-setup.goal.title")}
          aria-describedby={goal ? undefined : "workspace-setup-goal-note"}
          value={goal ?? ""}
          onValueChange={(value) => {
            if (isSetupGoal(value)) onGoalChange(value);
          }}
          className="flex-col items-stretch gap-y-2"
        >
          <RadioGroupItem
            value="create-database-change"
            className={choiceCardClassName(goal === "create-database-change")}
            radioClassName="mt-0.5"
            contentClassName="min-w-0"
          >
            <div className="font-medium text-main">
              {t("workspace-setup.goal.change.title")}
            </div>
            <div className="mt-1 text-control-light">
              {t("workspace-setup.goal.change.description")}
            </div>
            <div className="mt-2 text-xs text-control-light">
              {t("workspace-setup.goal.change.finish-line")}
            </div>
          </RadioGroupItem>
          <RadioGroupItem
            value="query-data"
            className={choiceCardClassName(goal === "query-data")}
            radioClassName="mt-0.5"
            contentClassName="min-w-0"
          >
            <div className="font-medium text-main">
              {t("workspace-setup.goal.query.title")}
            </div>
            <div className="mt-1 text-control-light">
              {t("workspace-setup.goal.query.description")}
            </div>
            <div className="mt-2 text-xs text-control-light">
              {t("workspace-setup.goal.query.finish-line")}
            </div>
          </RadioGroupItem>
          {/* Drawn without a card so it does not read as a third goal. */}
          <RadioGroupItem
            value="just-exploring"
            className="items-start px-4 py-2"
            radioClassName="mt-0.5"
            contentClassName="min-w-0"
          >
            <div className="text-control">
              {t("workspace-setup.goal.explore.title")}
            </div>
            <div className="mt-1 text-xs text-control-light">
              {t("workspace-setup.goal.explore.description")}
            </div>
          </RadioGroupItem>
        </RadioGroup>
        {!goal && (
          <SetupNote id="workspace-setup-goal-note">
            {t("workspace-setup.goal.required")}
          </SetupNote>
        )}
      </div>
    </SetupScreenFrame>
  );
}
