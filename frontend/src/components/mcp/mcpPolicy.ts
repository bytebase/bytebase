import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { MCPSetting } from "@/types/proto-es/v1/setting_service_pb";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import type {
  MCPCapabilityMode,
  MCPEngineEnforcement,
  MCPInfo,
  MCPMethod,
} from "@/types/proto-es/v1/workspace_service_pb";
import {
  MCPEngineEnforcement_Masking,
  MCPEngineEnforcement_ReadOnlyDepth,
} from "@/types/proto-es/v1/workspace_service_pb";

/** The ceilings an admin picks between, least to most capable. */
export const MCP_CAPABILITY_CHOICES: readonly MCPSetting_Capability[] = [
  MCPSetting_Capability.DISABLED,
  MCPSetting_Capability.READ_ONLY,
  MCPSetting_Capability.READ_WRITE,
];

/**
 * What the stored row says, which is not always a mode.
 *
 * `unconfigured` and `unreadable` both arrive as CAPABILITY_UNSPECIFIED and
 * mean opposite things: the first resolves READ_WRITE server-side, the second
 * is refused every connection. Collapsing them is the defect BOT-100 records.
 */
export type StoredCeiling =
  | { kind: "mode"; capability: MCPSetting_Capability }
  | { kind: "unconfigured" }
  | { kind: "unreadable" }
  | { kind: "unserved"; stored: string };

const isChoice = (capability: MCPSetting_Capability): boolean =>
  MCP_CAPABILITY_CHOICES.includes(capability);

/** Reads the stored row into the state the page renders. */
export const readStoredCeiling = (setting: MCPSetting): StoredCeiling => {
  // The server does not return the stored token, only that it could not
  // resolve one. Unserved is different: that value parsed, so the number is
  // already here and costs nothing to show.
  if (setting.capabilityUnreadable) {
    return { kind: "unreadable" };
  }
  if (isChoice(setting.capability)) {
    return { kind: "mode", capability: setting.capability };
  }
  if (setting.capability === MCPSetting_Capability.CAPABILITY_UNSPECIFIED) {
    return { kind: "unconfigured" };
  }
  // A number that parsed but no mode serves: the reserved 2, or a ceiling a
  // newer release wrote. Enforcement refuses it, so it is not a choice either.
  return { kind: "unserved", stored: String(setting.capability) };
};

/** A stored row the server refuses, which carries the value nobody could read. */
export type BrokenCeiling = Extract<
  StoredCeiling,
  { kind: "unreadable" } | { kind: "unserved" }
>;

/** Whether the stored row is one the server can act on. */
export const ceilingIsBroken = (
  stored: StoredCeiling
): stored is BrokenCeiling =>
  stored.kind === "unreadable" || stored.kind === "unserved";

/**
 * The card the form starts on, or undefined when the row is broken.
 *
 * A broken row starts on no card so that every pick is a change the footer can
 * save — including the most permissive one. Preselecting the ceiling such a row
 * resolves to is what left an admin unable to repair it in one save (BOT-100).
 */
export const initialCapabilityPick = (
  stored: StoredCeiling
): MCPSetting_Capability | undefined => {
  switch (stored.kind) {
    case "mode":
      return stored.capability;
    case "unconfigured":
      // No row resolves READ_WRITE server-side, so this is the ceiling in
      // force, not a guess.
      return MCPSetting_Capability.READ_WRITE;
    default:
      return undefined;
  }
};

/**
 * The methods a mode serves, by the same rule the gate evaluates: a method is
 * served when its class is one of the mode's served classes.
 */
export const methodsServedBy = (
  mode: MCPCapabilityMode | undefined,
  methods: MCPMethod[]
): MCPMethod[] => {
  if (!mode) {
    return [];
  }
  const served = new Set(mode.servedClasses);
  return methods.filter((method) => served.has(method.class));
};

/** The service a method belongs to: `/bytebase.v1.SQLService/Query` -> `SQLService`. */
export const serviceOfMethod = (method: MCPMethod): string => {
  const parts = method.method.split("/");
  const service = parts.length > 1 ? parts[1] : method.method;
  const short = service.split(".").pop();
  return short || service;
};

export interface MethodGroup {
  service: string;
  methods: MCPMethod[];
}

/** Groups served methods by service, both the groups and their rows sorted. */
export const groupMethodsByService = (methods: MCPMethod[]): MethodGroup[] => {
  const groups = new Map<string, MCPMethod[]>();
  for (const method of methods) {
    const service = serviceOfMethod(method);
    const existing = groups.get(service);
    if (existing) {
      existing.push(method);
    } else {
      groups.set(service, [method]);
    }
  }
  return [...groups.entries()]
    .map(([service, rows]) => ({
      service,
      methods: [...rows].sort((a, b) => a.method.localeCompare(b.method)),
    }))
    .sort((a, b) => a.service.localeCompare(b.service));
};

export interface EngineGroup<T> {
  value: T;
  engines: Engine[];
}

const groupEnginesBy = <T>(
  engines: MCPEngineEnforcement[],
  order: readonly T[],
  pick: (engine: MCPEngineEnforcement) => T
): EngineGroup<T>[] => {
  const groups = new Map<T, Engine[]>();
  for (const enforcement of engines) {
    const value = pick(enforcement);
    const existing = groups.get(value);
    if (existing) {
      existing.push(enforcement.engine);
    } else {
      groups.set(value, [enforcement.engine]);
    }
  }
  // Rank rather than filter. The server answers with the depth or masking mode
  // it computed, which a bundle in an open tab across a backend upgrade may not
  // have been compiled against; filtering the known order would take that
  // engine off the page entirely rather than describe it imprecisely. Both
  // renderers already have a default arm for a value they cannot name.
  const rank = (value: T): number => {
    const at = order.indexOf(value);
    return at === -1 ? order.length : at;
  };
  return [...groups.keys()]
    .sort((a, b) => rank(a) - rank(b))
    .map((value) => ({ value, engines: groups.get(value) ?? [] }));
};

/**
 * Read-only depth groups, deepest first. Grouping rather than listing every
 * engine is the point: an admin choosing Read-only is asking which engines the
 * ceiling reaches all the way down, and thirty rows do not answer that.
 */
export const READ_ONLY_DEPTH_ORDER = [
  MCPEngineEnforcement_ReadOnlyDepth.STATEMENT_AND_SESSION,
  MCPEngineEnforcement_ReadOnlyDepth.STATEMENT,
  MCPEngineEnforcement_ReadOnlyDepth.UNSUPPORTED,
  MCPEngineEnforcement_ReadOnlyDepth.READ_ONLY_DEPTH_UNSPECIFIED,
] as const;

export const groupEnginesByReadOnlyDepth = (
  engines: MCPEngineEnforcement[]
): EngineGroup<MCPEngineEnforcement_ReadOnlyDepth>[] =>
  groupEnginesBy(engines, READ_ONLY_DEPTH_ORDER, (e) => e.readOnlyDepth);

/**
 * Masking groups. Three states, never two: an engine Bytebase does not mask
 * and an engine whose masking never consults exemptions are different answers
 * to "does ignoring exemptions change anything here".
 */
export const MASKING_ORDER = [
  MCPEngineEnforcement_Masking.COLUMN,
  MCPEngineEnforcement_Masking.DOCUMENT,
  MCPEngineEnforcement_Masking.NONE,
  MCPEngineEnforcement_Masking.MASKING_UNSPECIFIED,
] as const;

export const groupEnginesByMasking = (
  engines: MCPEngineEnforcement[]
): EngineGroup<MCPEngineEnforcement_Masking>[] =>
  groupEnginesBy(engines, MASKING_ORDER, (e) => e.masking);

/** The engines whose per-engine answer carries a note. */
export const engineNotes = (
  engines: MCPEngineEnforcement[]
): MCPEngineEnforcement[] => engines.filter((engine) => engine.note !== "");

/** The mode row for one capability, or undefined when this build serves none. */
export const modeFor = (
  info: MCPInfo | undefined,
  capability: MCPSetting_Capability
): MCPCapabilityMode | undefined =>
  info?.modes.find((mode) => mode.capability === capability);

/**
 * What the consent page can truthfully tell someone about to approve a client.
 *
 * Only `mode` is a policy it can disclose. The other four are the ways it can
 * fail to hold one, kept apart because the remedy differs: retry, reload, or
 * ask an admin (BOT-106).
 */
export type ConsentCeiling =
  /** Carries the response, so the disclosure cannot be rendered without it. */
  | { kind: "mode"; info: MCPInfo }
  /** GetMCPInfo failed or timed out. The policy is not known to be anything. */
  | { kind: "unknown" }
  /** The workspace stores a value nothing can resolve to a ceiling. */
  | { kind: "unreadable" }
  /** It resolved, and the server serves no mode for it. */
  | { kind: "unserved" }
  /** The server serves it and this bundle has no name for it. */
  | { kind: "outdated" };

/**
 * Reads a GetMCPInfo response — or its absence — into that state.
 *
 * `modes` is the serving table the gate evaluates; MCP_CAPABILITY_CHOICES is
 * the set this bundle has copy for. A ceiling a newer release added is in the
 * first and not the second, which is the one case where reloading helps rather
 * than finding an admin.
 */
export const readConsentCeiling = (
  info: MCPInfo | undefined
): ConsentCeiling => {
  if (!info) {
    return { kind: "unknown" };
  }
  // Unspecified is not a ceiling: the server resolves one before answering, and
  // a workspace that never configured MCP resolves to READ_WRITE. So the only
  // way to see it here is a stored row nothing could resolve.
  if (info.capability === MCPSetting_Capability.CAPABILITY_UNSPECIFIED) {
    return { kind: "unreadable" };
  }
  if (!modeFor(info, info.capability)) {
    return { kind: "unserved" };
  }
  if (!isChoice(info.capability)) {
    return { kind: "outdated" };
  }
  return { kind: "mode", info };
};
