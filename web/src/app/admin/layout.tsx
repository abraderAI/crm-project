/**
 * Admin layout — simplified wrapper.
 * Navigation is handled by the sidebar sub-menu; this layout provides only
 * the page heading and content area.
 */
export default function AdminLayout({ children }: { children: React.ReactNode }): React.ReactNode {
  return (
    <div data-testid="admin-layout" className="mx-auto max-w-7xl px-6 py-6">
      <h1 className="text-2xl font-bold text-foreground">Admin Dashboard</h1>
      <div className="mt-6">{children}</div>
    </div>
  );
}
