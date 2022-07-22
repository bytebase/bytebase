## Help file naming rules

Let's name our help files based on routes/domains, and all files should start with `help.` to indicate that it is a help doc. For example, `help.environment.md` is the help file that stands for the whole `/environment` page. If you want to create a more detailed help file to explain *Approval Policy* under `\environment`, you can name the file to `help.environment.approval-policy.md`.

If the objective of the explanation doesn't belong to any specific route/domain, we can name it with `help.global...`.

## Route map configuration guide

File `routeMap.json` contains the configuration for the help system.

It maps from the route name to the help document name. If you're going to add a new mapping rule, you can add a new line in it.

The format is:

```json
    "name of route": "markdown's file name without '.md'"
```

For example, `"workspace.project": "help.project"` means let's connect the `help.project.md` file with the route `/project`.

Note:   

1. Configs here only works on `/issue`, `/project`, `/db`, `/instance`, `/environment` and `/setting`. If you want to add help in other places, please ask developers for help.
        
2. Make sure that the markdown file is available in both `/en` and `/zh` directory.