import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto/v1/common";
import { Environment, EnvironmentTier } from "../proto/v1/environment_service";

export const ENVIRONMENT_ALL_NAME = "environments/-";

export const emptyEnvironment = () => {
  return Environment.fromJSON({
    name: `environments/${EMPTY_ID}`,
    uid: String(EMPTY_ID),
    title: "",
    state: State.ACTIVE,
    order: 0,
    tier: EnvironmentTier.UNPROTECTED,
  });
};

export const unknownEnvironment = () => {
  return {
    ...emptyEnvironment(),
    name: `environments/${UNKNOWN_ID}`,
    uid: String(UNKNOWN_ID),
    title: "<<Unknown environment>>",
  };
};
