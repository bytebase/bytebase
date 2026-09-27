// Single BroadcastChannel instance for cross-tab "workspace switched"
// notifications.
//
// Sharing one instance matters for BroadcastChannel's source-object exclusion
// rule: a message posted via `channel.postMessage(...)` is NOT delivered to
// listeners attached to that same channel object — but IS delivered to any
// other channel objects with the same name (even in the same tab). A second
// channel object in this tab would receive the tab's own switch and could
// navigate the consent flow out from underneath itself.
//
// With a single shared instance, all in-tab listeners attached to it are
// correctly excluded when the consent page or any other caller broadcasts.

export const workspaceSwitchChannel = new BroadcastChannel(
  "bb-workspace-switch"
);

export const broadcastWorkspaceSwitch = (workspaceName: string) => {
  workspaceSwitchChannel.postMessage(workspaceName);
};
