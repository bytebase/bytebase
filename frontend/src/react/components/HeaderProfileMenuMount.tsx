import { ProfileMenuTrigger } from "@/react/components/header/ProfileMenuTrigger";

export function HeaderProfileMenuMount({
  size = "small",
}: {
  size?: "small" | "medium";
}) {
  return <ProfileMenuTrigger size={size} link />;
}
