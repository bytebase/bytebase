import type { Engine } from "@/types/proto-es/v1/common_pb";
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

export type MCPMode =
  | MCPSetting_Capability.DISABLED
  | MCPSetting_Capability.READ_ONLY
  | MCPSetting_Capability.READ_WRITE;

/** The ceilings an admin picks between, least to most capable. */
export const MCP_CAPABILITY_CHOICES: readonly MCPMode[] = [
  MCPSetting_Capability.DISABLED,
  MCPSetting_Capability.READ_ONLY,
  MCPSetting_Capability.READ_WRITE,
];

export const isMCPMode = (
  capability: MCPSetting_Capability
): capability is MCPMode =>
  MCP_CAPABILITY_CHOICES.some((choice) => choice === capability);

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
