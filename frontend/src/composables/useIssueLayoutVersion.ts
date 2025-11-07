import { useLocalStorage } from "@vueuse/core";
import { useActuatorV1Store, useCurrentUserV1 } from "@/store";
import { isDev } from "@/utils";

const STORAGE_KEY = "bb.issue.layout";

// When releasing a new version as default, increment this number.
// This helps to reset user preferences to the new default layout.
const CURRENT_DEFAULT_VERSION = 1;

// Send event to hub.bytebase.com when switching to legacy layout.
async function trackLayoutSwitch() {
  try {
    const actuatorStore = useActuatorV1Store();
    const currentUser = useCurrentUserV1();

    const workspaceId = actuatorStore.info?.workspaceId;
    const email = currentUser.value.email;
    const version = actuatorStore.version;
    const commit = actuatorStore.gitCommitBE;

    // Only track when switching to legacy layout
    if (!workspaceId || !email) {
      return;
    }

    // Send event to hub.bytebase.com
    await fetch("https://hub.bytebase.com/v1/events", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        workspaceId,
        email,
        version,
        commit,
        consoleCicdLayoutSwitchToLegacy: {
          reason: "",
        },
      }),
    });
  } catch {
    // Silently fail if tracking fails - don't block user action
  }
}

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
    const switchingToLegacy = enabled.value === true;
    enabled.value = !enabled.value;
    lastVersion.value = CURRENT_DEFAULT_VERSION;

    // Track the switch event when switching to legacy layout in production.
    if (switchingToLegacy && !isDev()) {
      trackLayoutSwitch();
    }

    // Reload the window to ensure clean state transition.
    window.location.reload();
  };

  return {
    enabledNewLayout: enabled,
    toggleLayout,
  };
}
