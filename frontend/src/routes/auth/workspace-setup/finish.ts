import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { projectServiceClientConnect } from "@/api";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { useAppStore } from "@/stores/app";
import { callOptions, isConnectAlreadyExists } from "@/stores/app/utils";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { PresetRoleType } from "@/types/iam/role";
import {
  type Binding,
  BindingSchema,
  GetIamPolicyRequestSchema,
  type IamPolicy,
} from "@/types/proto-es/v1/iam_policy_pb";
import {
  GetProjectRequestSchema,
  type Project,
} from "@/types/proto-es/v1/project_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { getUserEmailInBinding } from "@/types/v1/user";

export type SetupFinishStage = "names" | "project" | "sample" | "invitees";

// Why the server will not create a project with the chosen ID: its reason, or
// none when another project holds the ID. Only screen 3 can change the ID, so
// a refusal sends the wizard back there.
export type ProjectIdRefusal = { reason?: string };

// For each invitee role, the members it holds because a Finish attempt added
// them.
export type ProjectGrants = Readonly<Record<string, readonly string[]>>;

// What the server holds for a project ID whose create got no answer.
type LostCreate =
  | { kind: "own"; project: string }
  // No such project in this workspace.
  | { kind: "absent" }
  | { kind: "other" };

export type SetupNames = {
  userName: string;
  displayName: string;
  // Set only when this account may name the workspace.
  workspaceName?: string;
  workspaceTitle: string;
};

// True once this tab no longer holds setup (D11). No write is sent after that.
type Superseded = () => boolean;

type SetupFinishInput = SetupNames & {
  ownerEmail: string;
  // Self-hosted may leave the project for later (D12): nothing past the names
  // is written then.
  withoutProject: boolean;
  projectTitle: string;
  projectId: string;
  // The project an earlier attempt created.
  createdProject?: string;
  // An earlier attempt from this draft already sent a create for `projectId`.
  // Only then can a project that holds the ID be the one this draft made.
  createAttempted: boolean;
  prepareSample: boolean;
  invitees: readonly string[];
  projectGrants: ProjectGrants;
  superseded: Superseded;
};

type SetupFinishResult =
  | { ok: true; project: string | undefined }
  | {
      ok: false;
      stage: SetupFinishStage;
      error: unknown;
      refusal?: ProjectIdRefusal;
    }
  // The attempt stopped because this tab no longer holds setup.
  | { ok: false; superseded: true };

const SUPERSEDED = { ok: false, superseded: true } as const;

// The wizard shows each failure beside what failed, so the global error toast
// would report it a second time.
const REPORT = { byCaller: true };

// Together these let an invitee propose changes, run read-only queries, and
// export their results.
const INVITEE_PROJECT_ROLES: string[] = [
  PresetRoleType.PROJECT_DEVELOPER,
  PresetRoleType.SQL_EDITOR_EXPORTER,
];

// Screen 1's two names. The name is optional, and an empty one is not saved
// over the account's.
export const saveSetupNames = async (
  names: SetupNames,
  superseded: Superseded
) => {
  const store = useAppStore.getState();
  if (names.displayName) {
    await store.updateUser(
      create(UpdateUserRequestSchema, {
        user: { name: names.userName, title: names.displayName },
        updateMask: create(FieldMaskSchema, { paths: ["title"] }),
      }),
      REPORT
    );
  }
  if (names.workspaceName && !superseded()) {
    await store.updateWorkspace(
      create(WorkspaceSchema, {
        name: names.workspaceName,
        title: names.workspaceTitle,
      }),
      ["title"],
      REPORT
    );
  }
};

// Looks up what the server holds now for an ID whose create got no answer.
// The project is the draft's own only when it carries the draft's name and its
// access is still what the server gives a new project: its creator as owner,
// and nobody else. A project anyone else has been given access to is never
// taken over. Rejects when the server cannot be read.
export const findLostCreate = async (
  projectId: string,
  projectTitle: string,
  ownerEmail: string
): Promise<LostCreate> => {
  const name = `${projectNamePrefix}${projectId}`;
  let existing: Project;
  try {
    existing = await projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, { name }),
      callOptions(REPORT)
    );
  } catch (error) {
    if (error instanceof ConnectError && error.code === Code.NotFound) {
      return { kind: "absent" };
    }
    throw error;
  }
  if (existing.title !== projectTitle) return { kind: "other" };
  const { bindings } = await projectServiceClientConnect.getIamPolicy(
    create(GetIamPolicyRequestSchema, { resource: name }),
    callOptions(REPORT)
  );
  const untouched =
    bindings.length === 1 &&
    bindings[0].role === PresetRoleType.PROJECT_OWNER &&
    bindings[0].members.length === 1 &&
    bindings[0].members[0] === getUserEmailInBinding(ownerEmail);
  return untouched
    ? { kind: "own", project: existing.name }
    : { kind: "other" };
};

// Returns the project's resource name, or why it cannot be created.
const createOrAdoptProject = async (
  input: SetupFinishInput
): Promise<string | { error: unknown; refusal?: ProjectIdRefusal }> => {
  const store = useAppStore.getState();
  try {
    const created = await store.createProject(
      input.projectTitle,
      input.projectId,
      REPORT
    );
    return created.name;
  } catch (error) {
    if (error instanceof ConnectError && error.code === Code.InvalidArgument) {
      return { error, refusal: { reason: error.rawMessage } };
    }
    if (!isConnectAlreadyExists(error)) return { error };
    if (!input.createAttempted) return { error, refusal: {} };
    // The earlier attempt may have created the project although its answer
    // never arrived. An ID that exists and is not in this workspace is held
    // by another one: IDs are shared by all.
    try {
      const found = await findLostCreate(
        input.projectId,
        input.projectTitle,
        input.ownerEmail
      );
      return found.kind === "own" ? found.project : { error, refusal: {} };
    } catch (readError) {
      return { error: readError };
    }
  }
};

// Failures that may come after the server applied the write: the request was
// cut off, or the server failed while answering. Any other answer is the
// server's refusal, and a refused write changed nothing.
const MAY_HAVE_APPLIED = new Set([
  Code.Canceled,
  Code.Unknown,
  Code.DeadlineExceeded,
  Code.Unavailable,
  Code.Internal,
  Code.DataLoss,
]);
const refused = (error: unknown) =>
  error instanceof ConnectError && !MAY_HAVE_APPLIED.has(error.code);

// Gives the invitee roles to `invitees` and takes away the ones in `earlier`
// whose address is no longer listed: the wizard undoes only what it added, so
// a role someone else gave keeps its members. The write replaces every
// binding, the owner's included, so `policy` must be the one the server holds
// now, with its etag: the store's copy would miss the write of an earlier
// attempt whose answer was lost. It is sent even when it changes nothing, as
// such a write may still be on its way: of two writes built on the same etag,
// the server refuses whichever lands second.
const writeProjectInvitees = async (
  project: string,
  policy: IamPolicy,
  invitees: readonly string[],
  earlier: ProjectGrants,
  record: (grants: Record<string, string[]>) => void
) => {
  // The wizard's grants carry no condition; a conditional one is someone
  // else's and stays as it is.
  const isInviteeBinding = (binding: Binding, role: string) =>
    binding.role === role && !binding.condition?.expression;
  const listed = invitees.map(getUserEmailInBinding);
  const roles = INVITEE_PROJECT_ROLES.map((role) => {
    const current = policy.bindings
      .filter((binding) => isInviteeBinding(binding, role))
      .flatMap((binding) => binding.members);
    const mine = earlier[role] ?? [];
    // In the server's order, so a write that changes no member reorders none.
    const members = [
      ...new Set([
        ...current.filter(
          (member) => !mine.includes(member) || listed.includes(member)
        ),
        ...listed,
      ]),
    ];
    // The wizard's after this write: what it added before and still lists,
    // and what it adds now. A member who held the role already is not.
    const added = listed.filter(
      (member) => mine.includes(member) || !current.includes(member)
    );
    return { role, members, mine, added };
  });
  // Kept with the earlier additions until the write is known to have taken
  // them back: its answer may never arrive.
  record(
    Object.fromEntries(
      roles.map(({ role, mine, added }) => [
        role,
        [...new Set([...mine, ...added])],
      ])
    )
  );
  try {
    await useAppStore.getState().updateProjectIamPolicy(
      project,
      {
        ...policy,
        bindings: [
          ...policy.bindings.filter(
            (binding) =>
              !INVITEE_PROJECT_ROLES.some((role) =>
                isInviteeBinding(binding, role)
              )
          ),
          ...roles
            .filter(({ members }) => members.length > 0)
            .map(({ role, members }) =>
              create(BindingSchema, { role, members })
            ),
        ],
      },
      REPORT
    );
  } catch (error) {
    if (refused(error)) {
      record(
        Object.fromEntries(roles.map(({ role, mine }) => [role, [...mine]]))
      );
    }
    throw error;
  }
  record(Object.fromEntries(roles.map(({ role, added }) => [role, added])));
};

// Adds Workspace Member to each invitee. The store's patch replaces a member's
// whole role set, so a role the address already holds is sent along with it,
// taken from the policy the store has just fetched from the server, for the
// same reason as the project's.
const addWorkspaceMembers = (invitees: readonly string[]) => {
  const store = useAppStore.getState();
  return store.patchWorkspaceIamPolicy(
    invitees.map((email) => {
      const member = getUserEmailInBinding(email);
      return {
        member,
        roles: [
          ...new Set([
            ...store.findWorkspaceRolesByMember(member),
            PresetRoleType.WORKSPACE_MEMBER,
          ]),
        ],
      };
    }),
    REPORT
  );
};

type SetupFinishEvents = {
  // Called before the create is sent: its answer may never arrive.
  projectCreateSent: () => void;
  projectCreated: (project: string) => void;
  samplePrepared: (instanceName: string) => void;
  // The project's roles the wizard has given, as they stand.
  projectGrants: (grants: Record<string, string[]>) => void;
};

const commitSetup = async (
  input: SetupFinishInput,
  on: SetupFinishEvents
): Promise<SetupFinishResult> => {
  const store = useAppStore.getState();

  try {
    await saveSetupNames(input, input.superseded);
  } catch (error) {
    return { ok: false, stage: "names", error };
  }
  if (input.withoutProject) return { ok: true, project: undefined };

  let project = input.createdProject;
  if (!project) {
    if (input.superseded()) return SUPERSEDED;
    on.projectCreateSent();
    const created = await createOrAdoptProject(input);
    if (typeof created !== "string") {
      return { ok: false, stage: "project", ...created };
    }
    project = created;
    on.projectCreated(project);
  }

  if (input.prepareSample) {
    if (input.superseded()) return SUPERSEDED;
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("sample instance requested", {
        properties: { source: "workspace_setup" },
      })
    );
    try {
      const instance = await store.prepareSampleProjectInstance(project);
      on.samplePrepared(instance.name);
    } catch (error) {
      return { ok: false, stage: "sample", error };
    }
  }

  try {
    const earlier = input.projectGrants;
    if (
      input.invitees.length > 0 ||
      Object.values(earlier).some((members) => members.length > 0)
    ) {
      const policy = await projectServiceClientConnect.getIamPolicy(
        create(GetIamPolicyRequestSchema, { resource: project }),
        callOptions(REPORT)
      );
      if (input.superseded()) return SUPERSEDED;
      await writeProjectInvitees(
        project,
        policy,
        input.invitees,
        earlier,
        on.projectGrants
      );
    }
    // Last, because this write is what sends the invitations: nobody is
    // invited to a project they cannot open yet.
    if (input.invitees.length > 0) {
      await store.fetchWorkspaceIamPolicy(false, REPORT);
      if (input.superseded()) return SUPERSEDED;
      await addWorkspaceMembers(input.invitees);
    }
  } catch (error) {
    return { ok: false, stage: "invitees", error };
  }

  return { ok: true, project };
};

// The wizard's one commit. Creating the project is the only step that cannot
// be repeated, so a retry passes `createdProject`; the others are computed
// from the current answers and from what the server holds.
export const runSetupFinish = async (
  input: SetupFinishInput,
  on: SetupFinishEvents
): Promise<SetupFinishResult> => {
  const result = await commitSetup(input, on);
  // Losing setup while the last request was out leaves its outcome, a failure
  // too, to whoever holds setup next.
  return input.superseded() ? SUPERSEDED : result;
};
