import { DatabaseGroupForm } from "@/react/components/DatabaseGroupForm";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

export function ProjectDatabaseGroupCreatePage({
  projectId,
}: {
  projectId: string;
}) {
  const projectStore = useProjectV1Store();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  if (!project) return null;

  return (
    <DatabaseGroupForm
      className="py-4 h-full"
      readonly={false}
      project={project}
      onDismiss={() => router.back()}
      onCreated={(databaseGroupName) => {
        router.push({
          name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
          params: { databaseGroupName },
        });
      }}
    />
  );
}
