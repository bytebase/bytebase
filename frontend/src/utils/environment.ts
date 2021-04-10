import { Environment } from "../types";

export function environmentName(environment: Environment) {
  let name = environment.name;
  if (environment.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}
