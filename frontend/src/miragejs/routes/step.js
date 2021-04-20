import { Response } from "miragejs";
import { WORKSPACE_ID } from "./index";
import { IssueBuiltinFieldId } from "../../plugins";

export default function configureStep(route) {
  route.patch(
    "/pipeline/:pipelineId/step/:stepId/status",
    function (schema, request) {
      const attrs = this.normalizedRequestAttrs("step-status-patch");
      const pipeline = schema.pipelines.find(request.params.pipelineId);

      if (!pipeline) {
        return new Response(
          404,
          {},
          { errors: "Pipeline " + request.params.pipelineId + " not found" }
        );
      }

      const step = schema.steps.find(request.params.stepId);
      if (!step) {
        return new Response(
          404,
          {},
          { errors: "Step " + request.params.stepId + " not found" }
        );
      }

      const ts = Date.now();

      const changeList = [];

      if (attrs.status && step.status != attrs.status) {
        changeList.push({
          fieldId: IssueBuiltinFieldId.STEP_STATUS,
          oldValue: step.status,
          newValue: attrs.status,
        });

        const updatedStep = step.update({ ...attrs, updatedTs: ts });

        const payload = {
          changeList,
        };

        schema.activities.create({
          creatorId: attrs.updaterId,
          createdTs: ts,
          updaterId: attrs.updaterId,
          updatedTs: ts,
          actionType: "bytebase.issue.task.step.status.update",
          containerId: updatedStep.id,
          comment: attrs.comment,
          payload,
          workspaceId: WORKSPACE_ID,
        });

        return updatedStep;
      }

      return step;
    }
  );
}
