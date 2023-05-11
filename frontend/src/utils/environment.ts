import { Environment } from "../types";
import { Environment as EnvironmentV1 } from "@/types/proto/v1/environment_service";
import { State } from "@/types/proto/v1/common";

export function environmentName(environment: Environment) {
  let name = environment.name;
  if (environment.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}

export function environmentTitleV1(environment: EnvironmentV1) {
  let name = environment.title;
  if (environment.state == State.DELETED) {
    name += " (Archived)";
  }
  return name;
}
