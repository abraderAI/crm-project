import { fetchAdminSettings } from "@/lib/admin-api";
import { SystemSettings } from "@/components/admin/system-settings";

/** Admin system settings page — platform-wide configuration editor. */
export default async function AdminSettingsPage(): Promise<React.ReactNode> {
  const settings = await fetchAdminSettings();

  return <SystemSettings initialSettings={settings} />;
}
