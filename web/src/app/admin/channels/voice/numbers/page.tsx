import Link from "next/link";
import { PhoneNumberManager } from "@/components/admin/phone-number-manager";

const DEFAULT_ORG = "default";

/** Admin page for managing phone numbers (LiveKit provisioning). */
export default function PhoneNumbersPage(): React.ReactNode {
  return (
    <div data-testid="phone-numbers-page" className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <nav className="text-sm text-muted-foreground" data-testid="phone-numbers-breadcrumb">
        <Link href="/admin" className="hover:text-foreground">
          Admin
        </Link>
        <span className="mx-1">→</span>
        <Link href="/admin/channels" className="hover:text-foreground">
          Channels
        </Link>
        <span className="mx-1">→</span>
        <Link href="/admin/channels/voice" className="hover:text-foreground">
          Voice
        </Link>
        <span className="mx-1">→</span>
        <span className="font-medium text-foreground">Numbers</span>
      </nav>

      <PhoneNumberManager org={DEFAULT_ORG} />
    </div>
  );
}
