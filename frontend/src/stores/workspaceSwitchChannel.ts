// Single BroadcastChannel instance shared by both the Vue and React workspace
// stores for cross-tab "workspace switched" notifications.
//
// Centralizing this matters for BroadcastChannel's source-object exclusion
// rule: a message posted via `channel.postMessage(...)` is NOT delivered to
// listeners attached to that same channel object — but IS delivered to any
// other channel objects with the same name (even in the same tab). Before
// this module existed, the Vue store and the React store each created their
// own channel object, so a switch posted from one store reached the other's
// listener in the same tab and could navigate the consent flow out from
// underneath itself.
//
// With a single shared instance, all in-tab listeners attached to it are
// correctly excluded when the consent page or any other caller broadcasts.

export const workspaceSwitchChannel = new BroadcastChannel(
  "bb-workspace-switch"
);

export const broadcastWorkspaceSwitch = (workspaceName: string) => {
  workspaceSwitchChannel.postMessage(workspaceName);
};
