// One-time URL rewrite for renamed routes, run before the router mounts.
// Deep links from before the worksheet → saved query rename used
// `/sql-editor/projects/{project}/sheets/{sheet}`; bookmarks and shared
// links land as full page loads, so rewriting here (history.replaceState)
// covers every real entry point and lets the SQL Editor bootstrap read the
// current route shape only.
const LEGACY_SHEET_PATH =
  /^(\/sql-editor\/projects\/[^/]+)\/sheets\/([^/]+)\/?$/;

export function rewriteLegacyPath() {
  const match = LEGACY_SHEET_PATH.exec(window.location.pathname);
  if (!match) return;
  window.history.replaceState(
    window.history.state,
    "",
    `${match[1]}/savedQueries/${match[2]}${window.location.search}${window.location.hash}`
  );
}
