## Help file naming rules

Let's name our help files based on routes/domains, and all files should start with `help.` to indicate that it is a help doc. For example, `help.environment.md` is the help file that stands for the whole `/environment` page. If you want to create a more detailed help file to explain _Approval Policy_ under `\environment`, you can name the file to `help.environment.approval-policy.md`.

If the objective of the explanation doesn't belong to any specific route/domain, we can name it with `help.global...`.

## Route map configuration guide

File `routeMapList.json` contains the configuration for the help system.

It maps from the route name to the help document name. It lists all available route names in the whole project. If you want to add a new mapping rule, you can fill in the help name next to the route name.

The format is:

```json
[
  ...,
  {
    "routeName": "write name of route here",
    "helpName": "write markdown's file name without '.md' here"
  },
  ...
]
```

For example, `{ "routeName": "workspace.project", "helpName": "help.project" }` means let's connect the `help.project.md` file with the route `/project`.

Note:

1. Configs here work on all route names that are listed in `routeMapList.json`. If you want to add help in other places, please ask developers for help.
2. Don't forget to provide markdown files in both `/en` and `/zh` directory.
