import { CheckCircle } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { Badge } from "@/react/components/ui/badge";
import { Separator } from "@/react/components/ui/separator";
import { useVueState } from "@/react/hooks/useVueState";
import { rulesToTemplate } from "@/react/lib/sql-review/utils";
import { cn } from "@/react/lib/utils";
import { useProjectV1Store, useSQLReviewStore } from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicyTemplateV2 } from "@/types";
import { TEMPLATE_LIST_V2 as builtInTemplateList } from "@/types";

interface PolicyTemplate extends SQLReviewPolicyTemplateV2 {
  review?: {
    name: string;
    resources: string[];
  };
}

interface TemplateSelectorProps {
  required?: boolean;
  selectedTemplateId?: string;
  onSelectTemplate: (template: SQLReviewPolicyTemplateV2) => void;
}

function ResourceBadge({ resource }: { resource: string }) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();

  useEffect(() => {
    if (resource.startsWith(projectNamePrefix)) {
      projectStore.getOrFetchProjectByName(resource, true);
    }
  }, [projectStore, resource]);

  const projectTitle = useVueState(() =>
    resource.startsWith(projectNamePrefix)
      ? projectStore.getProjectByName(resource)?.title
      : undefined
  );

  if (resource.startsWith(environmentNamePrefix)) {
    return (
      <Badge variant="default" className="gap-x-1">
        <span className="text-control-light text-xs">
          {t("common.environment")}:
        </span>
        <EnvironmentLabel environmentName={resource} />
      </Badge>
    );
  }

  if (resource.startsWith(projectNamePrefix)) {
    return (
      <Badge variant="default" className="gap-x-1">
        <span className="text-control-light text-xs">
          {t("common.project")}:
        </span>
        <span>{projectTitle || resource}</span>
      </Badge>
    );
  }

  return <Badge variant="default">{resource}</Badge>;
}

export function TemplateSelector({
  required,
  selectedTemplateId,
  onSelectTemplate,
}: TemplateSelectorProps) {
  const { t } = useTranslation();
  const sqlReviewStore = useSQLReviewStore();
  const policyList = useVueState(() => [...sqlReviewStore.reviewPolicyList]);

  useEffect(() => {
    sqlReviewStore.fetchReviewPolicyList();
  }, []);

  const reviewPolicyTemplateList = useMemo<PolicyTemplate[]>(() => {
    return policyList.map((r) => rulesToTemplate(r));
  }, [policyList]);

  const isSelected = (template: SQLReviewPolicyTemplateV2) =>
    template.id === selectedTemplateId;

  return (
    <div className="flex flex-col gap-y-2">
      <p className="textlabel">
        {t("sql-review.create.basic-info.choose-template")}
        {required && <span className="text-error"> *</span>}
      </p>

      {reviewPolicyTemplateList.length > 0 && (
        <>
          <div className="flex flex-wrap gap-4">
            {reviewPolicyTemplateList.map((template) => (
              <button
                type="button"
                key={template.id}
                aria-pressed={isSelected(template)}
                className={cn(
                  "relative flex w-full cursor-pointer flex-col rounded-sm border border-control-border px-6 py-4 text-left transition-colors sm:max-w-xs",
                  "hover:bg-control-bg focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
                  isSelected(template) && "bg-control-bg"
                )}
                onClick={() => onSelectTemplate(template)}
              >
                <div className="text-left flex flex-col gap-y-2">
                  <span className="text-base font-medium">
                    {template.review?.name}
                  </span>
                  <div className="flex flex-wrap gap-2">
                    {template.review?.resources.map((resource) => (
                      <ResourceBadge key={resource} resource={resource} />
                    ))}
                  </div>
                  <p className="text-sm">
                    <span className="mr-2">
                      {t("sql-review.enabled-rules")}:
                    </span>
                    <span>{template.ruleList.length}</span>
                  </p>
                </div>
                {isSelected(template) && (
                  <CheckCircle className="absolute right-3 top-3 size-7 text-accent" />
                )}
              </button>
            ))}
          </div>

          <Separator className="my-2" />
        </>
      )}

      <div className="flex flex-wrap gap-4">
        {builtInTemplateList.map((template) => (
          <button
            type="button"
            key={template.id}
            aria-pressed={isSelected(template)}
            className={cn(
              "relative flex w-full cursor-pointer flex-col rounded-sm border border-control-border px-6 py-4 text-left transition-colors sm:max-w-xs",
              "hover:bg-control-bg focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
              isSelected(template) && "bg-control-bg"
            )}
            onClick={() => onSelectTemplate(template)}
          >
            <div className="text-left flex flex-col gap-y-2">
              <span className="text-base font-medium">
                {t(`sql-review.template.${template.id.split(".").join("-")}`)}
              </span>
              <p className="text-sm text-control-light">
                {t(
                  `sql-review.template.${template.id.split(".").join("-")}-desc`
                )}
              </p>
              <p className="text-sm">
                <span className="mr-2">{t("sql-review.enabled-rules")}:</span>
                <span>{template.ruleList.length}</span>
              </p>
            </div>
            {isSelected(template) && (
              <CheckCircle className="absolute right-3 top-3 size-7 text-accent" />
            )}
          </button>
        ))}
      </div>
    </div>
  );
}
