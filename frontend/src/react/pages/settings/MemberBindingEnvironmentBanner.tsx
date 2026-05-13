import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { DDLWarningCallout } from "@/react/components/role-grant/DDLWarningCallout";
import type { EnvLimitationKind } from "@/react/lib/project-member/utils";
import type { ProjectRoleBindingEnvironmentLimitationState } from "./membersPageEnvironment";

export function MemberBindingEnvironmentBanner({
  envLimitation,
  bindingKind,
}: Readonly<{
  envLimitation: ProjectRoleBindingEnvironmentLimitationState;
  bindingKind: EnvLimitationKind;
}>) {
  if (envLimitation.type === "unrestricted") {
    return <DDLWarningCallout type="binding-all" kind={bindingKind} />;
  }
  if (envLimitation.environments.length === 0) {
    return <DDLWarningCallout type="binding-none" kind={bindingKind} />;
  }
  return (
    <div className="flex flex-col gap-y-2">
      <DDLWarningCallout type="binding-some" kind={bindingKind} />
      <div className="flex flex-wrap gap-1">
        {envLimitation.environments.map((env) => (
          <EnvironmentLabel
            key={env}
            environmentName={env}
            className="text-xs"
          />
        ))}
      </div>
    </div>
  );
}
