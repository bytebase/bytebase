# Code Review Guide

_Note: this guide also applies to [bytebase.com](https://github.com/bytebase/bytebase.com) repository for producing content._

Please read and follow [Google's Code Review Guideline](https://google.github.io/eng-practices/). There is additional guide below because this is an open-source project and we'd like to have effective communications on GitHub.

# Install [Neat](https://neat.run/)

Neat subscribes you to the GitHub notifications:

1. Make sure to watch the Bytebase repository.
1. Go to your [notification setting](https://github.com/settings/notifications), make sure to check `Web and Mobile`
   ![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/codereview2.png)

# Additional Guide

## Who's next?

- Because it's an open-source project with developers coming from different timezones, we recommend asynchronous communication between reviewers and authors except for emergency such as change rollback or high-priority bug fixes. Each side should expect responses for **up to one business day**.
- However, if the reviewer doesn't respond in a business day, it's OK to ping the other side by commenting again on the PR, IM messaging, or offline chats. If there is a back and forth discussion on the same topic or you would think so, it will be more efficient to discuss offline.
- Each side should make very explicit comments to indicate whether the other side can take the turn, e.g. when all previous review comments are addressed, author should leave a PTAL comment and ask reviewer to review again.
- The goal is to complete the work nicely and quickly so that the authors should have more incentives to drive the discussions, resolve the comments, and merge the PR. However, the comment can be resolved by anyone whoever feels the end of a discussion or the comment is addressed. PRs can be merged by either authors or approvers whenever the PR is ready to be merged. If the comments are not perfectly addressed for a merged PR, leave a comment and the change be addressed by a follow-up PR.

### For reviewers

- If the change is good to go, please approve the PR with short note LGTM (Looks Good To Me).
- Otherwise, make comments to the related lines and finish the review with [Comment status](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/about-pull-request-reviews#about-pull-request-reviews). If the inline comment is nice to improve (not required), comment should start with nit(picking), e.g. "nit: use word hello instead of hi".
- If you cannot review e.g. during busy time, please leave comments to let authors know and unassign youself.

### For authors

- Start the review with only one reviewer. Once the change is approved, you can add additional reviewers if needed such as owner review. If you want each reviewer looking at different parts of the code, please leave explicit comments.
- Commit updates to the PR.
- Respond to the comments.
- [Re-requesting a review](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/about-pull-request-reviews#about-pull-request-reviews) with a short note PTAL (Please Take Another Look)
