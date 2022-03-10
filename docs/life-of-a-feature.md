# Life of a Feature

## Do's

1. Understand the goal by answering "what's the problem we're solving?".
1. Write technical design docs.
    - Do some researches and provide background information and overview.
    - Find a simple and sustainable solution. Technical decisions usually come along with trade-offs among several options. We need a simple solution for a difficult problem. The solution should also be scalable to future changes or growth if possible.
    - Split a large design into smaller ones, e.g. backend, UI, UX, etc.
    - ```(IMPORTANT)``` Define data model and design database schema. According to our [Version Management](version-management.md), any schema change could cause disruption to existing customers. Uncareful design will also lead to software scalability issues.
    - Design API by following [API style guide](https://github.com/bytebase/bytebase/blob/main/docs/api-style-guide.md)
    - Put thoughts on naming, because it's hard to change the names in database schemas and APIs. We can have different names for technical pieces and products.
    - Collabrate with peers and tech leads.
    - Write doc including comments in English. We are doing open-source with contributors globally.
2. Coding.
    - Follow [code review guide](code-review-guide.md) (small changes, effective communication, collabrate, and ```respect```).
    - Split changes to database schema, API, backend, frontend if possible, because you will get different reviewers looking at different parts. For example, we can make backend do dual writes for any API changes, switch the reads on UI, and clean up the dual writes in bacnkend.
    - ```(IMPORTANT)``` Think about compatability and don't break existing users. This usually happen if we change database schemas or APIs.
    - Guard new features behind a release flag especially for frontend using [`isDev()`](https://github.com/bytebase/bytebase/blob/4fd7ea41a716dbd72c85b0bc02f04fff5e08370f/frontend/src/main.ts#L41). We should release the feature only when it's mature.
    - Testing, key for product quality. This includes unit tests, [backend integration tests](https://github.com/bytebase/bytebase/tree/main/tests), frontend manual tests. Tests should cover critical user journeys. While writing backend integration tests, you will have even better idea on how users will use the product from end to end.
    - Golang code follows [Go Wiki](https://github.com/golang/go/wiki/CodeReviewComments) and [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).
    - Collabrate if a feature requires multiple developers to work on.
    - (Optional) After code is submitted, you can wait for a while and use a docker image with tag `bytebase/bytebase:dev-ci` for testing release build. See [Docker Deployment Guide](https://docs.bytebase.com/install/docker).
3. Documentation.
    - We should update [public documentation](https://docs.bytebase.com/) for new features.
4. Testing and feedback.
   - Before a feature is released, get some peers to try out this new features by following public documentation. Receive feedback and iterate.
5. Release and announcement. Cheers!

