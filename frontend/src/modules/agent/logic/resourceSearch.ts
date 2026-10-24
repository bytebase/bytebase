import { useAppStore } from "@/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export async function searchAgentProjects(
  query: string,
  pageToken: string,
  signal: AbortSignal
): Promise<{ items: Project[]; nextPageToken: string }> {
  const { projects, nextPageToken } = await useAppStore
    .getState()
    .fetchProjectList({
      filter: {
        query: query.trim(),
        state: State.ACTIVE,
        excludeDefault: true,
      },
      pageSize: 20,
      pageToken,
      signal,
      silent: true,
      orderBy: "title",
      cache: true,
    });
  return { items: projects, nextPageToken: nextPageToken ?? "" };
}
