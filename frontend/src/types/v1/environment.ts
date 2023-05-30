import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { State } from "../proto/v1/common";
import { Environment, EnvironmentTier } from "../proto/v1/environment_service";

export const ENVIRONMENT_ALL_NAME = "environments/-";
export const EMPTY_ENVIRONMENT_NAME = `environments/${EMPTY_ID}`;
export const UNKNOWN_ENVIRONMENT_NAME = `environments/${UNKNOWN_ID}`;

export const emptyEnvironment = () => {
  return Environment.fromJSON({
    name: EMPTY_ENVIRONMENT_NAME,
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
    name: UNKNOWN_ENVIRONMENT_NAME,
    uid: String(UNKNOWN_ID),
    title: "<<Unknown environment>>",
  };
};
