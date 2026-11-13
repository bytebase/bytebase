import githubIcon from "@/assets/git/github.svg";
import gitlabIcon from "@/assets/git/gitlab.svg";
import type { WorkflowProvider } from "../gitopsWorkflowYaml";

const LOGOS: Record<WorkflowProvider, string> = {
  github: githubIcon,
  gitlab: gitlabIcon,
};

// The git provider's logo, placed before text that names the provider or one
// of its repositories. The text carries the meaning, so the image has no name.
export function ProviderLogo({
  provider,
}: {
  readonly provider: WorkflowProvider;
}) {
  return <img src={LOGOS[provider]} alt="" className="size-4 shrink-0" />;
}
