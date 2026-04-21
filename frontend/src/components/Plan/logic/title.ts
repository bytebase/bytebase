import type { Project } from "@/types/proto-es/v1/project_service_pb";

/**
 * Governance gate for auto-generated plan/issue titles.
 *
 * Returns the auto-generated title for the plan-create route's `query.name`,
 * or `undefined` when the project has `enforceIssueTitle=true` ("Require manual
 * title"). Leaving `query.name` undefined forces the plan-create page to open
 * with an empty title; `CreateButton.vue` then blocks submit until the user
 * types a deliberate title that will appear in the audit log.
 *
 * DO NOT inline this gate away. It is the load-bearing check that makes an
 * audited project's setting mean what it says. Removing it silently reintroduces
 * auto-generated titles in the audit trail.
 *
 * The generator is invoked lazily so callers that build the auto-title via
 * store lookups or string formatting don't pay that cost when the gate returns.
 */
export const planQueryNameForProject = (
  project: Pick<Project, "enforceIssueTitle">,
  generate: () => string
): string | undefined => (project.enforceIssueTitle ? undefined : generate());

/**
 * Writes the auto-generated plan title into `query.name` unless the project
 * enforces manual titles. When `enforceIssueTitle` is true, leaves `query.name`
 * unset so the plan-create page opens with an empty title — the user must type
 * a deliberate title before `CreateButton.vue` allows submit.
 *
 * If `query.name` is already set by the caller, this helper clears it when
 * enforcement is on, to prevent a stale pre-fill from reaching an enforced
 * project.
 *
 * Use this at every plan-route launcher. It centralizes the governance contract
 * in one callable so there's exactly one place to audit.
 */
export const applyPlanTitleToQuery = (
  query: Record<string, string>,
  project: Pick<Project, "enforceIssueTitle">,
  generate: () => string
): void => {
  const name = planQueryNameForProject(project, generate);
  if (name === undefined) {
    delete query.name;
  } else {
    query.name = name;
  }
};
