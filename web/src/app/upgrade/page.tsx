import { UpgradePage } from "@/components/home/upgrade-page";

/**
 * /upgrade route — Self-service tier upgrade page.
 * Authentication is enforced by middleware (non-public route).
 * Renders the UpgradePage client component.
 */
export default function UpgradeRoute(): React.ReactNode {
  return <UpgradePage />;
}
