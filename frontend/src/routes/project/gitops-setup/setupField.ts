// The fields the files section can send the owner back to.
export type SetupFieldKey = "deploy-to" | "sign-in";

export const setupFieldId = (key: SetupFieldKey) => `gitops-field-${key}`;

// Brings a field into view and opens its picker. A field with no picker is
// only scrolled to.
export function goToSetupField(key: SetupFieldKey) {
  const field = document.getElementById(setupFieldId(key));
  if (!field) return;
  const reducedMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)"
  ).matches;
  field.scrollIntoView({
    block: "nearest",
    behavior: reducedMotion ? "auto" : "smooth",
  });
  field.querySelector<HTMLElement>('[data-slot="combobox-trigger"]')?.click();
}
