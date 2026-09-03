import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";
import {
  isGuideScenarioId,
  isGuideWorkspaceUsage,
} from "@/modules/workspace-setup-guide/selection";
import type {
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "@/modules/workspace-setup-guide/types";

type WorkspaceSetupQuestionnaireStepProps = {
  scenarioValue?: GuideScenarioId;
  workspaceUsageValue?: GuideWorkspaceUsage;
  onScenarioChange: (value: GuideScenarioId) => void;
  onWorkspaceUsageChange: (value: GuideWorkspaceUsage) => void;
  onContinue: () => void;
};

const SCENARIO_OPTIONS: Array<{
  id: GuideScenarioId;
  titleKey: string;
  descriptionKey: string;
}> = [
  {
    id: "create-database-change",
    titleKey: "settings.profile.setup-scenario.create-database-change.title",
    descriptionKey:
      "settings.profile.setup-scenario.create-database-change.description",
  },
  {
    id: "query-data",
    titleKey: "settings.profile.setup-scenario.query-data.title",
    descriptionKey: "settings.profile.setup-scenario.query-data.description",
  },
];

const WORKSPACE_USAGE_OPTIONS: Array<{
  id: GuideWorkspaceUsage;
  titleKey: string;
  descriptionKey: string;
}> = [
  {
    id: "team",
    titleKey: "settings.profile.setup-scenario.workspace-usage.team.title",
    descriptionKey:
      "settings.profile.setup-scenario.workspace-usage.team.description",
  },
  {
    id: "solo",
    titleKey: "settings.profile.setup-scenario.workspace-usage.solo.title",
    descriptionKey:
      "settings.profile.setup-scenario.workspace-usage.solo.description",
  },
];

const optionClassName = (selected: boolean) =>
  cn(
    "items-start rounded-sm border px-4 py-3",
    selected
      ? "border-accent bg-accent/5"
      : "border-control-border hover:bg-control-bg"
  );

export function WorkspaceSetupQuestionnaireStep({
  scenarioValue,
  workspaceUsageValue,
  onScenarioChange,
  onWorkspaceUsageChange,
  onContinue,
}: WorkspaceSetupQuestionnaireStepProps) {
  const { t } = useTranslation();

  return (
    <div className="flex w-full max-w-lg flex-col gap-y-6">
      <div className="flex flex-col gap-y-3">
        <h1 className="font-medium text-main">
          {t("settings.profile.setup-scenario.outcome-title")}
        </h1>
        <RadioGroup
          aria-label={t("settings.profile.setup-scenario.outcome-title")}
          value={scenarioValue ?? ""}
          onValueChange={(nextValue) => {
            if (isGuideScenarioId(nextValue)) onScenarioChange(nextValue);
          }}
          className="flex-col items-stretch gap-y-2"
        >
          {SCENARIO_OPTIONS.map((option) => (
            <RadioGroupItem
              key={option.id}
              value={option.id}
              className={optionClassName(scenarioValue === option.id)}
              radioClassName="mt-0.5"
              contentClassName="min-w-0"
            >
              <div className="font-medium text-main">{t(option.titleKey)}</div>
              <div className="mt-1 text-sm text-control-light">
                {t(option.descriptionKey)}
              </div>
            </RadioGroupItem>
          ))}
        </RadioGroup>
      </div>

      <div className="flex flex-col gap-y-3">
        <h2 className="font-medium text-main">
          {t("settings.profile.setup-scenario.workspace-usage.title")}
        </h2>
        <RadioGroup
          aria-label={t(
            "settings.profile.setup-scenario.workspace-usage.title"
          )}
          value={workspaceUsageValue ?? ""}
          onValueChange={(nextValue) => {
            if (isGuideWorkspaceUsage(nextValue)) {
              onWorkspaceUsageChange(nextValue);
            }
          }}
          className="flex-col items-stretch gap-y-2"
        >
          {WORKSPACE_USAGE_OPTIONS.map((option) => (
            <RadioGroupItem
              key={option.id}
              value={option.id}
              className={optionClassName(workspaceUsageValue === option.id)}
              radioClassName="mt-0.5"
              contentClassName="min-w-0"
            >
              <div className="font-medium text-main">{t(option.titleKey)}</div>
              <div className="mt-1 text-sm text-control-light">
                {t(option.descriptionKey)}
              </div>
            </RadioGroupItem>
          ))}
        </RadioGroup>
      </div>

      <div className="flex items-center justify-end gap-x-2 border-t border-block-border pt-4">
        <Button onClick={onContinue}>
          {t("settings.profile.setup-scenario.continue")}
        </Button>
      </div>
    </div>
  );
}
