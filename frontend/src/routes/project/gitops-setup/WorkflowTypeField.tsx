import { ExternalLink } from "lucide-react";
import { Trans, useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { FormField } from "@/components/ui/form";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

// Migration-based: versioned migration files. State-based (SDL): the folder
// holds the whole schema, and Bytebase generates the DDL.
export type WorkflowType = "migration" | "state";

// The engines state-based works on. The schema export (EngineSupportSDLExport
// on the server) and the deploy (the engines that register an SDL diff) cover
// the same three.
const STATE_BASED_ENGINES = new Set([
  Engine.POSTGRES,
  Engine.COCKROACHDB,
  Engine.MYSQL,
]);

export const supportsStateBased = (database: Database) =>
  STATE_BASED_ENGINES.has(
    database.instanceResource?.engine ?? Engine.ENGINE_UNSPECIFIED
  );

const STATE_BASED_LIMITS_URL =
  "https://docs.bytebase.com/gitops/state-based-workflow/limitations?source=console";

// The first choice, because it decides what every file holds. State-based
// shows the limits that decide whether it fits, before the owner goes on.
export function WorkflowTypeField({
  value,
  onChange,
}: {
  readonly value: WorkflowType;
  readonly onChange: (value: WorkflowType) => void;
}) {
  const { t } = useTranslation();
  const options: { value: WorkflowType; title: string; description: string }[] =
    [
      {
        value: "migration",
        title: t("gitops.setup.workflow-type.migration-title"),
        description: t("gitops.setup.workflow-type.migration-description"),
      },
      {
        value: "state",
        title: t("gitops.setup.workflow-type.state-title"),
        description: t("gitops.setup.workflow-type.state-description"),
      },
    ];
  return (
    <FormField title={t("gitops.setup.workflow-type.label")}>
      <div className="flex flex-col gap-y-2">
        <RadioGroup
          aria-label={t("gitops.setup.workflow-type.label")}
          value={value}
          onValueChange={(next) => onChange(next as WorkflowType)}
          className="grid grid-cols-1 items-stretch gap-2 sm:grid-cols-2"
        >
          {options.map((option) => (
            <RadioGroupItem
              key={option.value}
              value={option.value}
              className="items-start rounded-sm border border-control-border p-3 has-[>[data-checked]]:border-accent has-[>[data-checked]]:bg-accent/5"
              radioClassName="mt-0.5"
              contentClassName="flex min-w-0 flex-col gap-y-1"
            >
              <span className="font-medium text-main">{option.title}</span>
              <span className="text-control-light">{option.description}</span>
            </RadioGroupItem>
          ))}
        </RadioGroup>
        {value === "state" && (
          <Alert
            variant="info"
            title={t("gitops.setup.workflow-type.limits-title")}
          >
            <ul className="mt-1 list-disc pl-5 text-sm">
              <li>
                <Trans
                  i18nKey="gitops.setup.workflow-type.limits-schema-only"
                  components={{ b: <LimitLead /> }}
                />
              </li>
              <li>
                <Trans
                  i18nKey="gitops.setup.workflow-type.limits-engines"
                  components={{ b: <LimitLead /> }}
                />
              </li>
              <li>
                <Trans
                  i18nKey="gitops.setup.workflow-type.limits-statements"
                  components={{ b: <LimitLead /> }}
                />
              </li>
            </ul>
            <a
              href={STATE_BASED_LIMITS_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-1 inline-flex items-center gap-x-1 text-sm font-medium text-accent hover:underline"
            >
              {t("gitops.setup.workflow-type.limits-link")}
              <ExternalLink aria-hidden="true" className="size-3.5 shrink-0" />
            </a>
          </Alert>
        )}
      </div>
    </FormField>
  );
}

// The point a limit line opens with, filled in through <Trans components>.
function LimitLead({ children }: { readonly children?: React.ReactNode }) {
  return <strong className="font-medium">{children}</strong>;
}
