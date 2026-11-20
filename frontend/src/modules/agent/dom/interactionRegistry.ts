import type { PreparedApiCall } from "../logic/tools/callApi";
import type { AgentRunMode } from "../logic/types";
import type { DomActionParams, DomActionResult } from "./actions";
import type { IndexedElement } from "./domTree";
import { availableForActivation } from "./guards";
import { getElementByRef } from "./snapshotIndex";

export type AgentDomVerb = "click" | "input" | "select";
const verbs = ["click", "input", "select"] as const;
interface DomScope {
  readonly actor: string;
  readonly workspace: string;
  readonly chatId: string;
  readonly runId: string;
  readonly runMode: AgentRunMode;
  readonly pageKey: string;
  readonly pageGeneration: number;
  readonly projectName?: string;
}
export interface AgentDomContext extends DomScope {
  /** Checks the active stage's principal, route, foreground chat and execution ownership. */
  readonly isCurrent: () => boolean;
  readonly signal?: AbortSignal;
}
export function captureDomScope(context: AgentDomContext): DomScope {
  const {
    actor,
    workspace,
    chatId,
    runId,
    runMode,
    pageKey,
    pageGeneration,
    projectName,
  } = context;
  return Object.freeze({
    actor,
    workspace,
    chatId,
    runId,
    runMode,
    pageKey,
    pageGeneration,
    projectName,
  });
}
export function isDomContextCurrent(context: AgentDomContext): boolean {
  return !!(
    context.actor &&
    context.workspace &&
    context.chatId &&
    context.runId &&
    context.pageKey &&
    Number.isSafeInteger(context.pageGeneration) &&
    !context.signal?.aborted &&
    context.isCurrent()
  );
}
export function matchesDomScope(
  scope: DomScope | undefined,
  context: AgentDomContext
): boolean {
  return (
    !!scope &&
    scope.actor === context.actor &&
    scope.workspace === context.workspace &&
    scope.chatId === context.chatId &&
    scope.runId === context.runId &&
    scope.runMode === context.runMode &&
    scope.pageKey === context.pageKey &&
    scope.pageGeneration === context.pageGeneration &&
    scope.projectName === context.projectName &&
    isDomContextCurrent(context)
  );
}
export interface AgentDomMutationPreview {
  readonly kind: "database-labels" | "database-environment";
  readonly changes: readonly {
    readonly field: string;
    readonly before: string | null;
    readonly after: string | null;
  }[];
}
export type AgentDomBinding = {
  id: string;
  verb: AgentDomVerb;
  eligible: () => boolean;
} & (
  | {
      effect: "inspect" | "local-draft";
      run(value?: string): DomActionResult;
    }
  | {
      effect: "persist";
      prepare(
        value: string | undefined,
        context: AgentDomContext
      ): Promise<
        | { call: PreparedApiCall; preview: AgentDomMutationPreview }
        | { result: DomActionResult }
      >;
      validateFresh(
        context: AgentDomContext,
        call: Readonly<PreparedApiCall>
      ): Promise<boolean>;
    }
);
interface OwnerSnapshot {
  readonly pageKey: string;
  readonly resourceName: string;
  readonly projectName?: string;
  readonly revision: string;
}
export interface CommittedDomMutation {
  readonly call: Readonly<PreparedApiCall>;
  readonly revision: string;
}
export interface AgentDomOwner {
  ownerId: string;
  actor: string;
  workspace: string;
  getSnapshot(): OwnerSnapshot;
  /** Resolve only an exact element; neither ancestry nor shared component type grants authority. */
  getBinding(element: Element, verb: AgentDomVerb): AgentDomBinding | undefined;
  refresh?(
    context: AgentDomContext,
    committed: CommittedDomMutation
  ): Promise<void>;
}
type BindingEvidence = Readonly<{
  id: string;
  verb: AgentDomVerb;
  effect: AgentDomBinding["effect"];
}>;
export interface AgentDomBindingSnapshot extends OwnerSnapshot {
  readonly ownerId: string;
  readonly mountId: number;
  readonly epoch: number;
  readonly actor: string;
  readonly workspace: string;
  readonly bindings: readonly BindingEvidence[];
}
export interface CapturedDomMutation extends OwnerSnapshot {
  readonly actor: string;
  readonly workspace: string;
  readonly chatId: string;
  readonly runId: string;
  readonly runMode: AgentRunMode;
  readonly pageGeneration: number;
  readonly ownerId: string;
  readonly mountId: number;
  readonly bindingId: string;
  readonly verb: AgentDomVerb;
  readonly generation: number;
  readonly elementId: string;
  readonly preview: AgentDomMutationPreview;
}
export type AgentDomPreparation =
  | { kind: "result"; result: DomActionResult }
  | { kind: "confirm-api"; call: PreparedApiCall; origin: CapturedDomMutation };
interface Mount {
  owner: AgentDomOwner;
  identity: OwnerIdentity;
  id: number;
  epoch: number;
}
const owners = new Set<Mount>();
let nextMount = 0;
export function hasAgentDomOwners(): boolean {
  return owners.size > 0;
}
type OwnerIdentity = Pick<AgentDomOwner, "ownerId" | "actor" | "workspace">;
function sameOwner(left: OwnerIdentity, right: OwnerIdentity): boolean {
  return (
    left.ownerId === right.ownerId &&
    left.actor === right.actor &&
    left.workspace === right.workspace
  );
}
export function registerAgentDomOwner(owner: AgentDomOwner): () => void {
  const { ownerId, actor, workspace } = owner;
  const mount = {
    owner,
    identity: { ownerId, actor, workspace },
    id: ++nextMount,
    epoch: 0,
  };
  for (const existing of owners) {
    if (sameOwner(existing.identity, mount.identity)) existing.epoch++;
  }
  owners.add(mount);
  return () => {
    owners.delete(mount);
  };
}
function evidence(binding: AgentDomBinding): BindingEvidence | undefined {
  if (
    !binding.id ||
    !binding.eligible() ||
    !["inspect", "local-draft", "persist"].includes(binding.effect)
  )
    return;
  return {
    id: binding.id,
    verb: binding.verb,
    effect: binding.effect,
  };
}
function resolve(element: Element) {
  let match: { mount: Mount; bindings: AgentDomBinding[] } | undefined;
  for (const mount of owners) {
    if (!sameOwner(mount.owner, mount.identity)) continue;
    const bindings = verbs.flatMap((verb) => {
      const binding = mount.owner.getBinding(element, verb);
      return binding && binding.verb === verb ? [binding] : [];
    });
    if (!bindings.length) continue;
    if (
      match ||
      Array.from(owners).some(
        (other) => other !== mount && sameOwner(other.identity, mount.identity)
      )
    )
      return;
    match = { mount, bindings };
  }
  return match;
}
export function captureAgentDomBinding(
  element: Element
): AgentDomBindingSnapshot | undefined {
  const match = resolve(element);
  if (!match || !availableForActivation(element)) return;
  const { mount, bindings } = match;
  const entries = bindings.flatMap((binding) => evidence(binding) ?? []);
  const { owner } = mount;
  const { revision, resourceName, pageKey, projectName } = owner.getSnapshot();
  if (
    !entries.length ||
    !owner.ownerId ||
    !owner.actor ||
    !owner.workspace ||
    !revision ||
    !resourceName ||
    !pageKey
  )
    return;
  return {
    actor: owner.actor,
    workspace: owner.workspace,
    pageKey,
    resourceName,
    projectName,
    revision,
    ownerId: owner.ownerId,
    mountId: mount.id,
    epoch: mount.epoch,
    bindings: entries,
  };
}
function current(
  entry: IndexedElement,
  verb: AgentDomVerb,
  context: AgentDomContext
) {
  const captured = entry.binding;
  if (
    context.runMode !== "default" ||
    !matchesDomScope(entry.scope, context) ||
    !captured ||
    captured.actor !== context.actor ||
    captured.workspace !== context.workspace ||
    captured.pageKey !== context.pageKey ||
    (context.projectName && captured.projectName !== context.projectName) ||
    !availableForActivation(entry.element)
  )
    return;
  const match = resolve(entry.element);
  if (
    !match ||
    match.mount.id !== captured.mountId ||
    match.mount.epoch !== captured.epoch
  )
    return;
  const { owner } = match.mount;
  const live = owner.getSnapshot();
  if (
    owner.actor !== captured.actor ||
    owner.workspace !== captured.workspace ||
    owner.ownerId !== captured.ownerId ||
    !sameResource(live, captured) ||
    live.revision !== captured.revision
  )
    return;
  const binding = match.bindings.find((b) => b.verb === verb);
  const expected = captured.bindings.find((b) => b.verb === verb);
  const actual = binding && evidence(binding);
  if (
    !binding ||
    !expected ||
    !actual ||
    actual.id !== expected.id ||
    actual.effect !== expected.effect
  )
    return;
  return { mount: match.mount, binding };
}
function sameResource(left: OwnerSnapshot, right: OwnerSnapshot): boolean {
  return (
    left.pageKey === right.pageKey &&
    left.projectName === right.projectName &&
    left.resourceName === right.resourceName
  );
}
// The next snapshot retires eN refs. A prepared request instead retains its
// original target in memory, without putting a node or callback in its origin.
const permits = new WeakMap<
  CapturedDomMutation,
  { entry: IndexedElement; mount: Mount; committed: CommittedDomMutation }
>();
export const manualResult = (): DomActionResult => ({
  success: false,
  code: "manual-required",
  message:
    "Manual action required. This control has no current registered action for this request.",
});
export async function prepareCapturedInteraction(
  params: DomActionParams,
  entry: IndexedElement,
  context: AgentDomContext
): Promise<AgentDomPreparation> {
  const verb = params.type;
  const rejected: AgentDomPreparation = {
    kind: "result",
    result: manualResult(),
  };
  if (verb !== "click" && verb !== "input" && verb !== "select")
    return rejected;
  const get = () =>
    getElementByRef(params.ref) === entry
      ? current(entry, verb, context)
      : undefined;
  const matched = get();
  if (!matched) return rejected;
  const { binding } = matched;
  if (binding.effect !== "persist") {
    const next = get();
    if (!next || next.binding.effect === "persist") return rejected;
    // React may recycle the same node for another row before setters commit.
    if (next.binding.effect === "local-draft") next.mount.epoch++;
    return { kind: "result", result: next.binding.run(params.value) };
  }
  const prepared = await binding
    .prepare(params.value, context)
    .catch(() => undefined);
  if (!prepared || !get() || !entry.binding) return rejected;
  if ("result" in prepared) return { kind: "result", result: prepared.result };
  const { call, preview } = prepared;
  const { pageKey, resourceName, projectName, revision, ownerId, mountId } =
    entry.binding;
  const origin: CapturedDomMutation = Object.freeze({
    actor: context.actor,
    workspace: context.workspace,
    chatId: context.chatId,
    runId: context.runId,
    runMode: context.runMode,
    pageGeneration: context.pageGeneration,
    pageKey,
    resourceName,
    projectName,
    revision,
    ownerId,
    mountId,
    bindingId: binding.id,
    verb,
    generation: entry.generation,
    elementId: entry.ref,
    preview: Object.freeze({
      kind: preview.kind,
      changes: Object.freeze(
        preview.changes.map(({ field, before, after }) =>
          Object.freeze({ field, before, after })
        )
      ),
    }),
  });
  const frozen = Object.freeze({
    operationId: call.operationId,
    path: call.path,
    body: call.body,
  });
  permits.set(origin, {
    entry,
    mount: matched.mount,
    committed: Object.freeze({ call: frozen, revision }),
  });
  return { kind: "confirm-api", call: frozen, origin };
}
function resolveOrigin(origin: CapturedDomMutation, context: AgentDomContext) {
  const entry = permits.get(origin)?.entry;
  try {
    return entry && current(entry, origin.verb, context);
  } catch {
    return undefined;
  }
}
export function isCapturedDomMutationCurrent(
  origin: CapturedDomMutation,
  context: AgentDomContext
): boolean {
  return resolveOrigin(origin, context)?.binding.effect === "persist";
}

export function matchesCapturedDomMutationCall(
  origin: CapturedDomMutation,
  call: Readonly<PreparedApiCall>
): boolean {
  const captured = permits.get(origin)?.committed.call;
  return (
    !!captured &&
    captured.operationId === call.operationId &&
    captured.path === call.path &&
    captured.body === call.body
  );
}
export async function validateCapturedDomMutation(
  origin: CapturedDomMutation,
  context: AgentDomContext
): Promise<boolean> {
  try {
    const live = resolveOrigin(origin, context);
    const call = permits.get(origin)?.committed.call;
    return (
      live?.binding.effect === "persist" &&
      !!call &&
      (await live.binding.validateFresh(context, call)) &&
      isCapturedDomMutationCurrent(origin, context)
    );
  } catch {
    return false;
  }
}
export async function refreshCommittedDomMutation(
  origin: CapturedDomMutation,
  context: AgentDomContext
): Promise<void> {
  const permit = permits.get(origin);
  if (!permit) return;
  const { entry, mount, committed } = permit;
  // A successful save can disable its trigger or leave a newer draft. Only
  // scope and owner/resource identity govern read-only synchronization.
  const isCurrent = () =>
    matchesDomScope(entry.scope, context) &&
    owners.has(mount) &&
    sameOwner(mount.identity, mount.owner) &&
    !Array.from(owners).some(
      (other) => other !== mount && sameOwner(other.identity, mount.identity)
    ) &&
    sameResource(mount.owner.getSnapshot(), origin);
  if (isCurrent())
    await mount.owner.refresh?.({ ...context, isCurrent }, committed);
}
