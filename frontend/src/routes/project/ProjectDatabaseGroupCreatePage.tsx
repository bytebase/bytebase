import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/app/router/handles";
import { DatabaseGroupForm } from "@/components/DatabaseGroupForm";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";

export function ProjectDatabaseGroupCreatePage({
  projectId,
}: {
  projectId: string;
}) {
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const project = useProjectByName(projectName);

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
