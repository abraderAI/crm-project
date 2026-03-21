"use client";

import { useUserDirectory } from "@/lib/use-user-directory";
import { AuditLogViewer, type AuditLogViewerProps } from "./audit-log-viewer";

/** Client wrapper that injects useUserDirectory into AuditLogViewer. */
export function AuditLogViewerWithDirectory(
  props: Omit<AuditLogViewerProps, "formatUser">,
): React.ReactNode {
  const userDir = useUserDirectory();
  return <AuditLogViewer {...props} formatUser={userDir.format} />;
}
