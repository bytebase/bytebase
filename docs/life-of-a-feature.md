# Life of a Feature

## Do's

1. Understand the goal by answering "what's the problem we're solving?".
1. Write technical design docs.
   - Do some research and provide background information and overview.
   - Find a simple and sustainable solution. Technical decisions usually come along with trade-offs among several options. We need a simple solution for a complex problem. The solution should also be scalable to future changes or growth if possible.
   - Split a large design into smaller ones, e.g., backend, UI, UX, etc.
   - **(IMPORTANT)** Define data model and design database schema. According to our [Version Management](version-management.md), any schema change could disrupt existing customers. The uncareful design will also lead to software scalability issues.
   - Design API by following [API style guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md)
   - Put thoughts on naming because it's hard to change the names in database schemas and APIs. We can have different names for technical pieces and products.
   - Collaborate with peers and tech leads.
   - Write docs including comments in English. We are doing open-source with contributors globally.
2. Coding.
   - Follow [code review guide](code-review-guide.md) (small changes, effective communication, collaboration, and ```respect```).
   - Split changes to database schema, API, backend, frontend if possible, because you will get different reviewers looking at different parts. For example, we can make the backend do dual writes for any API changes, switch the reads on UI, and clean up the dual writes in the backend.
   - **(IMPORTANT)** Think about compatibility and don't break existing users. This usually happens if we change database schemas or APIs.
   - Guard new features behind a release flag, especially for frontend using [`isDev()`](https://github.com/bytebase/bytebase/blob/4fd7ea41a716dbd72c85b0bc02f04fff5e08370f/frontend/src/main.ts#L41). We should release the feature only when it's mature.
   - Testing is the key to product quality. This includes unit tests, [backend integration tests](https://github.com/bytebase/bytebase/tree/main/tests), frontend manual tests. Tests should cover critical user journeys. While writing backend integration tests, you will have an even better idea of how users will use the product from end to end.
   - Golang code follows [Go Wiki](https://github.com/golang/go/wiki/CodeReviewComments) and [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).
   - Collaborate if a feature requires multiple developers to work on.
   - (Optional) After code is submitted, you can wait for a while and use a docker image with tag `bytebase/bytebase:dev-ci` for testing release build. See [Docker Deployment Guide](https://docs.bytebase.com/install/docker).
1. Documentation.
   - We should update [public documentation](https://docs.bytebase.com/) for new features.
2. Testing and feedback.
   - Before a feature is released, get some peers to try out these new features by following public documentation. Receive feedback and iterate.
3. Release and announcement. Cheers!


## Branch Management

Since git utilizes branches as a primary development pattern, we usually face the problem of branch management. We suggest naming your fork of the code, i.e. `${YourGithubID}/bytebase` as `origin`, and the repo `bytebase/bytebase` as `upstream`. Here's a guide for following this branch development pattern.

### Remote Tracking

After forking the `bytebase/bytebase` repository, set up the git remote tracking.

```bash
# clone your bytebase fork
git clone git@github.com:${YourGithubID}/bytebase.git
cd bytebase
# setup upstream pointing to bytebase/bytebase
git remote add upstream git@github.com:bytebase/bytebase.git
# check the result
git remote -v
# expected outputs:
#   origin     git@github.com:${YourGithubID}/bytebase.git (fetch)
#   origin     git@github.com:${YourGithubID}/bytebase.git (push)
#   upstream   git@github.com:bytebase/bytebase.git (fetch)
#   upstream   git@github.com:bytebase/bytebase.git (push)
```

Now you have set up two tracked repositories: `upstream` for `bytebase/bytebase` and `origin` for your fork.

### Development

We usually create a new branch when we start developing a new feature. Here's a typical workflow.

```bash
# checkout to the main branch
git checkout main
# sync with the upstream
git pull upstream main
# create and checkout to your new feature branch
git checkout -b feat/xxx
# coding & commit
# push to origin
git push
# then git will prompt you with a complete command to push and track to the origin, copy & paste
git push --set-upstream origin feat/xxx
```

### Branch Naming Convention

The recommended branch naming convention is using `/` as a namespace separator, e.g., feat/xxx, chore/xxx, docs/xxx, which works nicely with 3rd party git tools like GitLens.
