import { useTranslation } from "react-i18next";
import type { EnvLimitationKind } from "@/components/ProjectMember/utils";
import { Alert } from "@/react/components/ui/alert";

type DDLWarningProps =
  | { type: "drawer"; kind: EnvLimitationKind }
  | { type: "issue"; kind: EnvLimitationKind; environments: string[] }
  | { type: "binding-some"; kind: EnvLimitationKind }
  | { type: "binding-all"; kind: EnvLimitationKind }
  | { type: "binding-none"; kind: EnvLimitationKind };

export function DDLWarningCallout(props: DDLWarningProps) {
  const { t } = useTranslation();
  switch (props.type) {
    case "drawer":
      return (
        <Alert variant="warning">
          {t("project.members.ddl-warning", { kind: props.kind })}
        </Alert>
      );
    case "issue":
      return (
        <Alert variant="warning">
          {t("issue.role-grant.ddl-warning", {
            kind: props.kind,
            environments: props.environments.join(", "),
          })}
        </Alert>
      );
    case "binding-some":
      return (
        <Alert variant="warning">
          {t("project.members.ddl-current-some", { kind: props.kind })}
        </Alert>
      );
    case "binding-all":
      return (
        <Alert variant="warning">
          {t("project.members.ddl-current-all", { kind: props.kind })}
        </Alert>
      );
    case "binding-none":
      return (
        <Alert variant="warning">
          {t("project.members.ddl-current-none", { kind: props.kind })}
        </Alert>
      );
  }
}
