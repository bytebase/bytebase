import { useLocalStorage } from "@vueuse/core";

const STORAGE_KEY = "bb.issue.layout";

// When releasing a new version as default, increment this number.
// This helps to reset user preferences to the new default layout.
const CURRENT_DEFAULT_VERSION = 1;

export function useIssueLayoutVersion() {
  // Store whether user prefers the new layout (true) or old layout (false).
  const enabled = useLocalStorage<boolean>(`${STORAGE_KEY}.enabled`, true);

  // Store the version number when user last made a choice
  const lastVersion = useLocalStorage<number>(
    `${STORAGE_KEY}.version`,
    CURRENT_DEFAULT_VERSION
  );

  // If a new version is released and user hasn't made a choice for it,
  // reset to the new default (true)
  if (lastVersion.value < CURRENT_DEFAULT_VERSION) {
    enabled.value = true;
    lastVersion.value = CURRENT_DEFAULT_VERSION;
  }

  // Toggle between layouts
  const toggleLayout = () => {
    enabled.value = !enabled.value;
    lastVersion.value = CURRENT_DEFAULT_VERSION;

    // Reload the window to ensure clean state transition.
    window.location.reload();
  };

  return {
    enabledNewLayout: enabled,
    toggleLayout,
  };
}
