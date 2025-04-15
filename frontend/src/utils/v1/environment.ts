import type { Environment } from "@/types/v1/environment";

export const extractEnvironmentResourceName = (name: string) => {
  const pattern = /(?:^|\/)environments\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export function environmentV1Name(environment: Environment) {
  const parts = [environment.title];
  return parts.join(" ");
}
