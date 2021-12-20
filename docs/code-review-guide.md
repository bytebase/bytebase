# Code review workflow

The code review workflow is like ping-pong between the author and reviewer. To streamline the process and avoid conflicts, both sides need to know which side should take the turn at the moment.

There are 2 approaches to inform this transition, the sync way and the async way.

## The sync way

Right after one finishes her part, she tells the other side immediately, usually via IM or face-to-face. We do NOT recommend this approach except for emergency because:

1. It's interruptive to your peers.
2. We are a distributed team, also working on open source means we have contributors in different timezones.

## The async way

This is the preferred way. On the other hand, it requires some discipline to make this approach efficient, thus we define some review guidelines.

# Guideline

## Terms

Below list some common review terms

### For the reviewer

- LGTM (Looks Good To Me): short note while approving the PR.

- Needs more work: require the reviewer to refine the PR.

- Request more info: require the reviewer to provide more info.

- nit: nit(picking), nice to improve.

### For the author

- PTAL (Please Take Another Look): request the reviewer to take another look after addressing the review comments.

# An example flow

1. Author creates the PR and request 1 or more reviewers.

1. The reviewer spots couple issues, leaves corresponding comments and sends the PR back by choosing **Request changes** when submitting the review.

1. The author addresses the comments and requests the reviewer to review again. We require author to **explicilty leave a PTAL comment** to indicate this.

1. The above 2 steps could take couple rounds until the reviewer chooses **Approve** when submitting the review and **explicitly leaves a LGTM comment**.

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/codereview1.png)

## Notes

1. GitHub does a poor job indicating which side should take the turn. Thus, we strongly encourage each side to explicitly leave comment to indicate the other side can take the turn. e.g. Author should leave a PTAL comment when she addresses all previous review comments and ask reviwer to review again.
1. For the author, only include multiple reviewers if really needed. Multiple reviewers would increase review time and the reviewers would also be confused about who should review the code. If you want each reviewer review different parts of the code, please leave explicit comments.
1. The assigned reviewer is expected to give review feedback in 1 business day. If the reviewer is busy, please let the author know. The reviewer can unassign herself, just leave a comment to explain.

Though we don't recommend sync commnunication, sometimes it's more appropriate.

1. The reviewer is not responding in a business day. It's OK to ping the other side to check the status.
2. If there is back and forth on a particular topic, it's usaully more efficient to discuss offline.

# Tools

1. You can install [Neat](https://neat.run/) to subscribe GitHub notification, make sure to watch the bytebase repository.

1. You can also try [graphite](https://graphite.dev/) which is a new code review tool for GitHub. One nice feature is it displays who should take the turn for each ongoing PR.

![Screenshot](https://raw.githubusercontent.com/bytebase/bytebase/main/docs/assets/codereview2.png)
