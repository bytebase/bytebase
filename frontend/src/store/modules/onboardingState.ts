import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";

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
 *
 * If the value in localStorage is gone (e.g., the user cleared the browser
 * data), the onboarding state will also be cleared. This is to ensure that the
 * onboarding actions will never fire more than once.
 */

export const useOnboardingStateStore = defineStore("onboardingState", () => {
  // states
  const state = useLocalStorage<OnboardingState>("bb.onboarding-state", {
    isOnboarding: false,
    consumed: [],
  });

  // actions
  const initialize = () => {
    state.value = {
      isOnboarding: true,
      consumed: [],
    };
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
    initialize,
    getStateByKey,
    consume,
  };
});
