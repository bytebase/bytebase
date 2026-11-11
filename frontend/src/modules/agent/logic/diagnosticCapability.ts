import { normalizeAgentResourceReference } from "./resourceMention";
import type { AgentDiagnosticCapability } from "./types";

const RESOURCE_ID_PATTERN = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

const isConcreteScope = (value: string) => {
  let decoded = value;
  try {
    decoded = decodeURIComponent(value);
  } catch {
    return false;
  }
  return (
    !decoded
      .split("/")
      .some(
        (token) => token === "-" || token === "create" || token.includes("*")
      ) &&
    !/[a-z][a-z0-9+.-]*:\/\//i.test(decoded) &&
    !/(?:password|passwd|pwd|secret|token|api[_-]?key|authorization)\s*[:=]/i.test(
      decoded
    ) &&
    !/-----BEGIN [A-Z ]+-----/i.test(decoded)
  );
};

export const normalizeAgentDiagnosticCapability = (
  raw: unknown
): AgentDiagnosticCapability | undefined => {
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    return undefined;
  }
  const candidate = raw as Record<string, unknown>;
  if (candidate.kind !== "diagnostic") return undefined;
  const resource = normalizeAgentResourceReference(candidate.resource);
  if (!resource || resource.kind === "project") return undefined;
  if (
    !isConcreteScope(resource.name) ||
    !isConcreteScope(resource.project) ||
    !RESOURCE_ID_PATTERN.test(resource.project.split("/")[1] ?? "") ||
    (resource.kind === "database" &&
      !RESOURCE_ID_PATTERN.test(resource.name.split("/").at(-3) ?? "")) ||
    ((resource.kind === "plan" || resource.kind === "issue") &&
      !/^[1-9]\d*$/.test(resource.name.split("/").at(-1) ?? ""))
  ) {
    return undefined;
  }
  return {
    kind: "diagnostic",
    resource: { ...resource, label: resource.name },
  };
};
