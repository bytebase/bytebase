Pipeline, Stage, Task are the backbones of execution.

A PIPELINE consists of multiple STAGES. A STAGE consists of multiple TASKS.

Comparison with Tekton
PIPELINE = Tekton Pipeline
STAGE = N/A
TASK = Tekton Task
N/A = Tekton Step

Comparison with GitLab:
PIPELINE = GitLab Pipeline
STAGE = GitLab Stage
TASK = GitLab Job
N/A = GitLab Script Step

Comparison with GitHub:
PIPELINE = GitHub Workflow
STAGE = N/A
TASK = GitHub Job
N/A = GitHub Step

Comparison with Octopus:
PIPELINE = Octopus Lifecycle
STAGE = Octopus Phase + Task
TASK = Octopus Step

Comparsion with Jenkins:
PIPELINE = Jenkins Pipeline
STAGE = Jenkins Stage
TASK = Jenkins Step (but it's also called task from its doc)

Comparsion with Spinnaker:
PIPELINE = Spinnaker Pipeline
STAGE = Spinnaker Stage
TASK = Spinnaker Task

Design consideration

- Other mainstream products either have 3 or 4 layers.
  We choose 3 layers omitting the most granular layer - Step. For now only GitLab has 4 layer systems
  and its step is mostly used to model a lightweight step like shell script step. This seems like
  an overkill for our case. BTW, Octopus employes 3 layer design which seems to be sufficient.

- All products agree on the smallest querable execution unit (having a dedicated API resource endpoint):
  Tekton Task/GitLab Job/GitHub Job/Octopus Step.
  Thus, we also choose Task as our smallest querable execution unit.

- We also have a Stage concept which is similar to GitLab Stage/Octopus Phase, in that it's a
  container to group mulitple tasks. Stage is usually used to model a stage in the development
  lifecycle (dev, testing, staging, prod).

- Only Pipeline and Task have status, while Stage doesn't. Stage's status derives from its
  containing Tasks.

- Pipeline status is 1-to-1 mapping to the Issue status. We introduce Pipepline for decoupling
  pipeline logic (workflow orchestration etc) from issue logic (collabration etc). And it
  helps testing (we can mock the entire pipeline implemenation) and also allows Pipeline to
  be reused in other situation. On the other hand, we want to reduce the complexity of
  introducing this extra layer, thus we always try to make a fixed 1-to-1 mapping for their
  respective fields. Client code could combine Pipeline status and its running step status (substatus)
  to achieve more granular behavior.

So we finally arrive the same conclusion as spinnaker

We require a stage to associate with a database. Since database belongs to an instance, which
in turns belongs to an environment, thus the stage is also associated with an instance and environment.
The environment has tiers which defines rules like whether requires manual approval.
