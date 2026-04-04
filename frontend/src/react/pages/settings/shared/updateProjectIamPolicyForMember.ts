import { create } from "@bufbuild/protobuf";
import { useProjectIamPolicyStore } from "@/store/modules/v1/projectIamPolicy";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";

export async function updateProjectIamPolicyForMember(
  projectIamPolicyStore: ReturnType<typeof useProjectIamPolicyStore>,
  projectName: string,
  member: string,
  newRoles: string[]
) {
  const policy = structuredClone(
    projectIamPolicyStore.getProjectIamPolicy(projectName)
  );
  for (const binding of policy.bindings) {
    binding.members = binding.members.filter((m) => m !== member);
  }
  policy.bindings = policy.bindings.filter(
    (binding) => binding.members.length > 0
  );
  for (const role of newRoles) {
    const existing = policy.bindings.find((b) => b.role === role);
    if (existing) {
      if (!existing.members.includes(member)) {
        existing.members.push(member);
      }
    } else {
      policy.bindings.push(create(BindingSchema, { role, members: [member] }));
    }
  }
  await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
}
