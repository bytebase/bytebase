import { ProfileMenuTrigger } from "@/components/header/ProfileMenuTrigger";

export function HeaderProfileMenuMount({
  size = "small",
}: {
  size?: "small" | "medium";
}) {
  return <ProfileMenuTrigger size={size} link />;
}
