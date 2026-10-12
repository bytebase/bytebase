import type { ConnectionFailureCategory } from "./ConnectionRecovery";

export type RecoveryActionKind = "ssh-tunnel" | "sample" | "skip-tls-verify";

// The failures where Bytebase Cloud never got an answer from the database.
const unreachableCategories: ConnectionFailureCategory[] = [
  "private_address",
  "host_not_found",
  "network_unreachable",
  "timeout",
];

// Any SSH field turns the tunnel switch on, before there is a host to dial.
export const sshTunnelIsOn = (dataSource: {
  sshHost: string;
  sshPort: string;
  sshUser: string;
}): boolean =>
  !!(dataSource.sshHost || dataSource.sshPort || dataSource.sshUser);

// Decides which fixes the recovery panel offers as buttons, in display order.
export function recoveryActionKinds({
  category,
  message,
  isSaaSMode,
  canUseSshTunnel,
  hasSampleInstance,
  verifiesServerCertificate,
}: {
  category: ConnectionFailureCategory;
  message: string;
  isSaaSMode: boolean;
  // The engine supports a tunnel and the data source has it off.
  canUseSshTunnel: boolean;
  hasSampleInstance: boolean;
  verifiesServerCertificate: boolean;
}): RecoveryActionKind[] {
  const kinds: RecoveryActionKind[] = [];
  if (isSaaSMode && unreachableCategories.includes(category)) {
    if (canUseSshTunnel) kinds.push("ssh-tunnel");
    if (hasSampleInstance) kinds.push("sample");
  }
  // Only a certificate Bytebase refused is fixed by not checking it; a server
  // that does not speak TLS is not.
  if (
    category === "ssl_tls_failed" &&
    verifiesServerCertificate &&
    /x509|certificate/i.test(message)
  ) {
    kinds.push("skip-tls-verify");
  }
  return kinds;
}
