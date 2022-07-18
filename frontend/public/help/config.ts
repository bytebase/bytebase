// This file contains the configuration for the help system.

// Below is a mapping object from the route name to the help document name.
// If you're going to add a new mapping rule, you can add a new line between the curly brackets {}.
// The format is:
//
//     "name of route": "markdown's file name without '.md'",
//
// For example, "workspace.project": "help.project" means let's connect the help.project.md file with the route "/project".
//
// Note: 1. Configs here only works on /issue, /project, /db, /instance, /environment and /setting. If you
//          want to add help in other places, please ask developers for help.
//       2. Make sure that the markdown file is in both /en and /zh directory.

export const routeHelpNameMap = {
  "workspace.project": "help.project",
  "workspace.instance": "help.instance",
  "workspace.database": "help.database",
  "workspace.environment": "help.environment",
};
