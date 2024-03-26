import type { Project } from "@/types/proto/v1/project_service";
import type { ProjectGitOpsInfo } from "@/types/proto/v1/vcs_provider_service";

export interface ComposedRepository extends ProjectGitOpsInfo {
  project: Project;
}
