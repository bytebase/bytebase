import { defineStore } from "pinia";
import { useLocalStorage } from "@vueuse/core";

type OnboardingState = {
  isOnboarding: boolean;
  consumed: string[];
};
/**
 * OnboardingState will tell if we are onboarding the workspace, so we can do
 * some welcomes, tutorials, demonstrations if needed.
 *
 * example:
 * - { isOnboarding: false }
 *   (the default and fallback value) means we are not onboarding the workspace
 * - { isOnboarding: true, consumed: ["a", "b"] }
 *   means we are onboarding the workspace, and module "a" and "b" already
 *   consumed the onboarding state.
 */

export const useOnboardingStateStore = defineStore("onboardingState", () => {
  // states
  const state = useLocalStorage<OnboardingState>("bb.onboarding-state", {
    isOnboarding: false,
    consumed: [],
  });

  // actions
  const setState = (newState: OnboardingState) => {
    state.value = newState;
  };
  const getStateByKey = (key: string) => {
    if (!state.value.isOnboarding) return false;
    return !state.value.consumed.includes(key);
  };
  const consume = (key: string) => {
    state.value.consumed.push(key);
  };

  // exposure
  return {
    setState,
    getStateByKey,
    consume,
  };
});
