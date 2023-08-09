import { ProjectGitOpsInfo } from "@/types/proto/v1/externalvs_service";
import { Project } from "@/types/proto/v1/project_service";

export interface ComposedRepository extends ProjectGitOpsInfo {
  project: Project;
}
