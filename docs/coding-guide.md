# General

1. Put more thought on data modeling and naming.
1. Write comment and use English.
1. If the change is beyond trivial, write informative PR title and description.


# How to build a feature

Split into smaller changes. This makes reviewer life easy and you will get faster feedback.

If you are working on an end-to-end feature including both backend and frontend, the usual steps to follow are:

1. If it requires schema change, design the schema first (you may need to discuss with peers).
1. Design the API, our [API style guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md).
1. Finish the rest.

Ideally, you will split the schema change, API change and the rest into separate PRs. If you put them together, then if the schema requires a change after the review, it will end up with a lot of code changes. For obvious schema changes, you can still choose to put them in a single PR.

All in all, figure out the schema/data model first before moving forward and use your judgement to decide whether to split the change into separate PRs.

## Example commits

1. An end-to-end example showing you the code touched when adding a field to the schema and populate it all the way to UI: [Add path field to backup setting](https://github.com/bytebase/bytebase/commit/a7c28a4fefb2c2cff0c1ed9bb7fc043a47f535cd#diff-e547f2c710d4d67f2887ee13f4361d35537404829114e9c10d6aa5f48b3179dc)


# Code review workflow

We prefer the async review workflow since we can't always IM the other side.

The review workflow is like ping-pong between the author and reviewer. To streamline the process, please leave explicit comment to indicate you have finished your part and request the other side to take action.

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/codereview1.png)

You can install [Neat](https://neat.run/) to subscribe GitHub notification, make sure to watch the bytebase repository.

## Common reviewer term
* LGTM (Looks Good To Me): short note while approving the PR.

* Needs more work: require the reviewer to refine the PR.

* Request more info: require the reviewer to provide more info.

## Common author term

* PTAL (Please Take Another Look): request the reviewer to take another look after addressing the review comments.