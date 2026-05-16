import { ReleaseInfoCard } from "@/react/components/release/ReleaseInfoCard";
import { useReleaseByName } from "@/store/modules/release";

export function DeployReleaseInfoCard({
  className,
  releaseName,
}: {
  className?: string;
  releaseName: string;
}) {
  const { release, ready } = useReleaseByName(releaseName);
  return (
    <ReleaseInfoCard
      className={className}
      isLoading={!ready.value}
      release={release.value ?? undefined}
      releaseName={releaseName}
    />
  );
}
