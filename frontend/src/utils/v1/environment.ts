import slug from "slug";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";

export const extractEnvironmentResourceName = (name: string) => {
  const pattern = /(?:^|\/)environments\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export function environmentV1Slug(environment: Environment): string {
  return [slug(environment.title), environment.uid].join("-");
}

export function environmentV1Name(environment: Environment) {
  const parts = [environment.title];
  if (environment.state === State.DELETED) {
    parts.push("(Archived)");
  }
  return parts.join(" ");
}
