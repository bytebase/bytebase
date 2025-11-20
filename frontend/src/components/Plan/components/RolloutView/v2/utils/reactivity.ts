import type { Ref } from "vue";

/**
 * Trigger reactivity for a Set ref by creating a new Set instance.
 * This is necessary because Vue's reactivity system doesn't automatically track Set mutations.
 * By creating a new Set instance, we trigger Vue's reactive system to detect the change.
 */
export const triggerSetReactivity = <T>(setRef: Ref<Set<T>>): void => {
  setRef.value = new Set(setRef.value);
};

/**
 * Add an item to a reactive Set and trigger reactivity
 */
export const addToSet = <T>(setRef: Ref<Set<T>>, item: T): void => {
  setRef.value.add(item);
  triggerSetReactivity(setRef);
};

/**
 * Delete an item from a reactive Set and trigger reactivity
 */
export const deleteFromSet = <T>(setRef: Ref<Set<T>>, item: T): void => {
  setRef.value.delete(item);
  triggerSetReactivity(setRef);
};

/**
 * Clear a reactive Set and trigger reactivity
 */
export const clearSet = <T>(setRef: Ref<Set<T>>): void => {
  setRef.value.clear();
  triggerSetReactivity(setRef);
};

/**
 * Toggle an item in a reactive Set and trigger reactivity
 */
export const toggleInSet = <T>(setRef: Ref<Set<T>>, item: T): void => {
  if (setRef.value.has(item)) {
    deleteFromSet(setRef, item);
  } else {
    addToSet(setRef, item);
  }
};
