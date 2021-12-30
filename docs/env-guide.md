# Environment Guide

This guide describes the environments for our dev workflow.

## Summary

Sometime it is really pain in the neck that reviewers would read your changes line by line without running it.
To ease this, we adopt [Render](https://render.com/) for feature preview.
Now, **each PR** to the main branch will be deployed on Render automatically, and a link to your preview environment would be commented by Render right at your PR.
![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/preview-env1.png)

## Environment

For now, we have three environments configured at Render:
| Env     | Creation                                      | Destroy       | URL                                               |
| ------- | --------------------------------------------- | ------------- | ------------------------------------------------- |
| Preview | New **PR** created                            | **PR** closed | [internal-preview-pr-**${PR No}**.onrender.com]() |
| Staging | New **Pre-Release** created                   | **Never**     | staging.bytebase.com                              |
| Demo    | New **commits** merged to **the main branch** | **Never**     | demo.bytebase.com                                 |

### Preview Environment

Preview environment will be deploy on a PR basis.
Once a new pull request is created, a preview environment for your PR would automatically be deployed.
You can use the link commented at your PR to do some testing and share with your reviewers.
Also, any new commit pushed to your PR branch will trigger a update. However, it may take a while (usually within minutes) for Render to update the environment. We added a 5-digits commit hash to the version tag at the left bottom, please check this to see whether your newest commit has been updated.
![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/preview-env2.png)

### [Staging Environment](https://staging.bytebase.com/)

Staging Environment is for release preview, and it would be triggered by a prerelease action in Github.
You can access this environment by clicking [here](https://staging.bytebase.com/).
![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/staging-env1.png)

### [Demo Environment](https://demo.bytebase.com/)

Demo Environment always reflect the main branch. Every updates at the main branch will trigger a update for Demo Environment.
You can access this environment by clicking [here](https://demo.bytebase.com/).

## The Dev Workflow with Environment

1. Design your feature, our [API style guide](https://github.com/bytebase/bytebase/blob/main/docs/coding-guide.md).
2. Create a Pull Request, and checkout the environment.
3. Request a review, our [Review guide](https://github.com/bytebase/bytebase/blob/main/docs/code-review-guide.md)
4. Make changes if necessary.
5. Merge to the main branch
6. Checkout the [Demo Environment](https://demo.bytebase.com/) to see if your new feature is well function.
