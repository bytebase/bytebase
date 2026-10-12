import { CopyButton } from "@/components/ui/copy-button";
import { useAppStore } from "@/stores/app";

const NO_ADDRESSES: string[] = [];

// The addresses Bytebase Cloud connects from, when the deployment publishes
// them. Empty on self-hosted deployments.
export const useCloudEgressAddresses = (): string[] =>
  useAppStore((s) =>
    s.isSaaSMode()
      ? (s.serverInfo?.egressIpAddresses ?? NO_ADDRESSES)
      : NO_ADDRESSES
  );

export function CloudEgressAddresses({
  addresses,
}: Readonly<{ addresses: string[] }>) {
  return (
    <span className="inline-flex flex-wrap items-center gap-x-2 gap-y-1">
      {addresses.map((address) => (
        <span key={address} className="inline-flex items-center gap-x-1">
          <code className="font-mono font-medium">{address}</code>
          <CopyButton content={address} />
        </span>
      ))}
    </span>
  );
}
