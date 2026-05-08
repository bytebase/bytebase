import { useTranslation } from "react-i18next";
import type { EnvLimitationKind } from "@/components/ProjectMember/utils";
import { Alert } from "@/react/components/ui/alert";

type DDLWarningProps =
  | { type: "drawer"; kind: EnvLimitationKind }
  | { type: "issue"; kind: EnvLimitationKind; environments: string[] }
  | { type: "binding-some"; kind: EnvLimitationKind }
  | { type: "binding-all"; kind: EnvLimitationKind }
  | { type: "binding-none"; kind: EnvLimitationKind };

const typeToKey: Record<DDLWarningProps["type"], string> = {
  drawer: "project.members.ddl-warning",
  issue: "issue.role-grant.ddl-warning",
  "binding-some": "project.members.ddl-current-some",
  "binding-all": "project.members.ddl-current-all",
  "binding-none": "project.members.ddl-current-none",
};

export function DDLWarningCallout(props: DDLWarningProps) {
  const { t } = useTranslation();
  const key = typeToKey[props.type];
  const interpolated =
    props.type === "issue"
      ? t(key, {
          kind: props.kind,
          environments: props.environments.join(", "),
        })
      : t(key, { kind: props.kind });
  return <Alert variant="warning">{interpolated}</Alert>;
}
