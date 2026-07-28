import { useEffect, useState } from "react";
import { ReleaseInfoCard } from "@/components/release/ReleaseInfoCard";
import { useReleaseByName } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";

export function DeployReleaseInfoCard({
  className,
  releaseName,
}: {
  className?: string;
  releaseName: string;
}) {
  const release = useReleaseByName(releaseName);
  const fetchRelease = useAppStore((state) => state.fetchRelease);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let canceled = false;
    setReady(false);
    void fetchRelease(releaseName, true).finally(() => {
      if (!canceled) {
        setReady(true);
      }
    });
    return () => {
      canceled = true;
    };
  }, [fetchRelease, releaseName]);

  return (
    <ReleaseInfoCard
      className={className}
      isLoading={!ready}
      release={release}
      releaseName={releaseName}
    />
  );
}
