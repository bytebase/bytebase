import { redirect, replace } from "react-router";
import { getAppRouterState } from "./navigation";

// A direct or POP/REPLACE legacy URL must replace itself or Back loops through
// the redirect. Only a genuine in-app PUSH should preserve the referring entry;
// React Router commits the redirect destination without committing the alias.
export const canonicalRedirect = (location: string): Response => {
  const state = getAppRouterState();
  return state?.initialized === true &&
    state.navigation?.historyAction === "PUSH"
    ? redirect(location)
    : replace(location);
};
