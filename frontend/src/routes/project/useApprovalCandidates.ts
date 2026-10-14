import { useCallback, useEffect, useMemo, useState } from "react";
import { useCurrentUser } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import { ensureGroupIdentifier } from "@/stores/app/group";
import {
  extractServiceAccountId,
  serviceAccountToUser,
} from "@/stores/app/serviceAccount";
import {
  extractWorkloadIdentityId,
  workloadIdentityToUser,
} from "@/stores/app/workloadIdentity";
import {
  projectNamePrefix,
  serviceAccountNamePrefix,
  userNamePrefix,
  workloadIdentityNamePrefix,
} from "@/stores/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  AccountType,
  getAccountTypeByEmail,
  groupBindingPrefix,
  unknownUser,
} from "@/types/v1/user";
import {
  ensureUserFullName,
  isBindingPolicyExpired,
  memberMapToRolesInProjectIAM,
} from "@/utils";
import {
  type ApprovalIneligibility,
  getApprovalEligibility,
} from "./approvalEligibility";

export interface ApprovalCandidate {
  canApprove: boolean;
  canReview: boolean;
  ineligibilities: ApprovalIneligibility[];
  user: User;
}

export interface ApprovalCandidates {
  // Every active principal holding the role: people, service accounts and
  // workload identities, each as a `users/{email}` User. They stay visible even
  // when a project policy means they cannot approve.
  candidates: ApprovalCandidate[];
  // Role membership remains distinct from review and approval eligibility.
  canCurrentUserApprove: boolean;
  isCurrentUserReviewCandidate: boolean;
  isCurrentUserRoleMember: boolean;
}

export function useApprovalCandidates(
  issue: Issue,
  projectId: string,
  role: string,
  lastPlanEditor: string
): ApprovalCandidates {
  const currentUser = useCurrentUser();
  const currentUserEmail = currentUser?.email ?? "";
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);
  const projectIamPolicy = useAppStore(
    (state) => state.projectPoliciesByName[projectName]
  );
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const fetchServiceAccount = useAppStore((state) => state.fetchServiceAccount);
  const fetchWorkloadIdentity = useAppStore(
    (state) => state.fetchWorkloadIdentity
  );
  const batchGetOrFetchGroups = useAppStore(
    (state) => state.batchGetOrFetchGroups
  );
  const groupsByName = useAppStore((state) => state.groupsByName);
  const getGroupByIdentifier = useCallback(
    (identifier: string) => groupsByName[ensureGroupIdentifier(identifier)],
    [groupsByName]
  );
  const [users, setUsers] = useState<User[]>([]);

  // Prefetch any groups bound to this role so candidate expansion can see
  // their members (same dance as the existing useApprovalStep).
  const groupNamesKey = useMemo(() => {
    if (!projectIamPolicy) return "";
    const names: string[] = [];
    for (const binding of projectIamPolicy.bindings) {
      if (binding.role !== role || isBindingPolicyExpired(binding)) continue;
      for (const member of binding.members) {
        if (member.startsWith(groupBindingPrefix)) {
          names.push(member);
        }
      }
    }
    return JSON.stringify([...new Set(names)].sort());
  }, [projectIamPolicy, role]);

  useEffect(() => {
    const names = groupNamesKey ? (JSON.parse(groupNamesKey) as string[]) : [];
    void batchGetOrFetchGroups(names).catch(() => undefined);
  }, [groupNamesKey, batchGetOrFetchGroups]);

  const holderNamesKey = useMemo(() => {
    // An empty role would make memberMapToRolesInProjectIAM skip its role
    // filter and return every project member — guard so callers that pass a
    // placeholder role (e.g. when no step is current) get no candidates.
    if (!projectIamPolicy || !role) return "";
    const memberMap = memberMapToRolesInProjectIAM(
      projectIamPolicy,
      role,
      getGroupByIdentifier
    );
    return JSON.stringify([...memberMap.keys()].sort());
  }, [projectIamPolicy, role, getGroupByIdentifier]);

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      const names = holderNamesKey
        ? (JSON.parse(holderNamesKey) as string[])
        : [];
      const withPrefix = (prefix: string) =>
        names.filter((name) => name.startsWith(prefix));
      // Each kind of principal is served by its own service. A record that
      // does not resolve keeps a placeholder named from its email.
      const [people, serviceAccounts, workloadIdentities] = await Promise.all([
        batchGetOrFetchUsers(withPrefix(userNamePrefix)),
        Promise.all(
          withPrefix(serviceAccountNamePrefix).map(async (name) => {
            const serviceAccount = await fetchServiceAccount(name, true);
            return serviceAccount
              ? serviceAccountToUser(serviceAccount)
              : unknownUser(ensureUserFullName(extractServiceAccountId(name)));
          })
        ),
        Promise.all(
          withPrefix(workloadIdentityNamePrefix).map(async (name) => {
            const workloadIdentity = await fetchWorkloadIdentity(name, true);
            return workloadIdentity
              ? workloadIdentityToUser(workloadIdentity)
              : unknownUser(
                  ensureUserFullName(extractWorkloadIdentityId(name))
                );
          })
        ),
      ]);
      if (canceled) return;
      const isPerson = (user: User) =>
        getAccountTypeByEmail(user.email) === AccountType.USER;
      setUsers(
        [
          // A machine's email bound as `user:` matches no policy member on the
          // server, so it holds nothing.
          ...people.filter(isPerson),
          ...serviceAccounts,
          ...workloadIdentities,
        ]
          .filter((user) => user.state === State.ACTIVE)
          .sort((left, right) => {
            if (left.email === currentUserEmail) return -1;
            if (right.email === currentUserEmail) return 1;
            if (isPerson(left) !== isPerson(right)) {
              return isPerson(left) ? -1 : 1;
            }
            return left.title.localeCompare(right.title);
          })
      );
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [
    batchGetOrFetchUsers,
    currentUserEmail,
    fetchServiceAccount,
    fetchWorkloadIdentity,
    holderNamesKey,
  ]);

  const candidates = useMemo(
    () =>
      users.map((user) => ({
        ...getApprovalEligibility({
          actor: user.name,
          issueCreator: issue.creator,
          lastPlanEditor,
          project,
        }),
        user,
      })),
    [issue.creator, lastPlanEditor, project, users]
  );
  const currentUserCandidate = candidates.find(
    (candidate) => candidate.user.email === currentUserEmail
  );

  return {
    candidates,
    canCurrentUserApprove: currentUserCandidate?.canApprove ?? false,
    isCurrentUserReviewCandidate: currentUserCandidate?.canReview ?? false,
    isCurrentUserRoleMember: currentUserCandidate !== undefined,
  };
}
