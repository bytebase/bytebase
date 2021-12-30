# Environment Guide

This guide describes the environments used in the Bytebase dev workflow. We use [Render](https://render.com/) to deploy our build for the corresponding environment.

## Environment

| Env     |              | When created or reloaded                      | Destroy       | URL                                           |
| ------- | ------------ | --------------------------------------------- | ------------- | ----------------------------------------------|
| Preview | Read & Write | New **PR** created                            | **PR** closed | internal-preview-pr-**${PR No}**.onrender.com |
| Staging | Read & Write | New **Pre-Release** created                   | **Never**     | [staging.bytebase.com](staging.bytebase.com)  |
| Demo    | Read Only    | New **commits** merged to **the main branch** | **Never**     | [demo.bytebase.com](demo.bytebase.com)        |

### Preview Environment

Whenever a new PR is created, a preview environment for that PR will be automatically deployed on render.
You can use the link commented at your PR to do some testing and share with your reviewers.

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/preview-env1.png)

Any following commits pushed to that PR will trigger an update. However, it may take a while (usually within minutes) for Render to update the environment. We add a 5-digits git commit hash to the version tag at the left bottom, please check this to see whether the preview has loaded your latest commit.

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/preview-env2.png)

### [Staging Environment (staging.bytebase.com)](https://staging.bytebase.com)

Staging Environment is for release preview, and it would be triggered by a prerelease action in Github.

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/staging-env1.png)

### [Demo Environment (demo.bytebase.com)](https://demo.bytebase.com)

Demo Environment always reflects the main branch. Every update at the main branch will trigger an update for the Demo Environment. 

**We intentionally do this because we adopt trunk based development and we want the main branch to always stay in a deployable state.**

## The Dev Workflow with Environment

1. Design your feature, our [API style guide](https://github.com/bytebase/bytebase/blob/main/docs/coding-guide.md).
2. Create a Pull Request, and check the preview.
3. Request a review, our [Review guide](https://github.com/bytebase/bytebase/blob/main/docs/code-review-guide.md).
4. Make changes if necessary.
5. Merge to the main branch.
6. Checkout the [Demo Environment](https://demo.bytebase.com/) to see if your new feature behaves properly.
