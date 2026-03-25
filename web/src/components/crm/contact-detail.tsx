"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { Users, Building2, TrendingUp, AlertTriangle, PenLine } from "lucide-react";

import type { Thread, Message } from "@/lib/api-types";
import { parseLeadData, STAGE_LABELS, type PipelineStage } from "@/lib/crm-types";
import { fetchContact, fetchLinkedOpportunities, fetchEntityMessages } from "@/lib/crm-api";
import { ApiError } from "@/lib/api-client";
import { useTier } from "@/hooks/use-tier";
import { MessageTimeline } from "@/components/thread/message-timeline";
import { Breadcrumbs, type BreadcrumbItem } from "@/components/layout/breadcrumbs";

export interface ContactDetailProps {
  contactId: string;
}

/** Contact detail view: attributes, parent company link, opps, activity timeline. */
export function ContactDetail({ contactId }: ContactDetailProps): React.ReactNode {
  const { orgId } = useTier();
  const { getToken, userId } = useAuth();
  const router = useRouter();

  const [contact, setContact] = useState<Thread | null>(null);
  const [opportunities, setOpportunities] = useState<Thread[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  const loadData = useCallback(async (): Promise<void> => {
    if (!orgId) return;
    setIsLoading(true);
    setError(null);
    try {
      const token = await getToken();
      if (!token || !mountedRef.current) return;

      const contactData = await fetchContact(token, orgId, contactId);
      if (!mountedRef.current) return;

      const [oppsData, msgsData] = await Promise.all([
        fetchLinkedOpportunities(token, orgId, "contacts", contactId),
        fetchEntityMessages(token, orgId, "contacts", contactId),
      ]);

      if (!mountedRef.current) return;
      setContact(contactData);
      setOpportunities(oppsData.data);
      setMessages(msgsData.data);
    } catch (err) {
      if (mountedRef.current) {
        // API returns 404 for invisible contacts → redirect to list
        if (err instanceof ApiError && err.status === 404) {
          router.push("/crm/contacts");
          return;
        }
        setError(err instanceof Error ? err.message : "Failed to load contact");
      }
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [getToken, orgId, contactId, router]);

  useEffect(() => {
    mountedRef.current = true;
    void loadData();
    return () => {
      mountedRef.current = false;
    };
  }, [loadData]);

  if (isLoading) {
    return (
      <div
        data-testid="contact-detail-loading"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        Loading contact...
      </div>
    );
  }

  if (error || !contact) {
    return (
      <div data-testid="contact-detail-error" className="flex flex-col items-center gap-3 py-12">
        <AlertTriangle className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm text-foreground">{error ?? "Contact not found"}</p>
        <Link href="/crm/contacts" className="text-sm text-primary hover:underline">
          Back to contacts
        </Link>
      </div>
    );
  }

  const meta = parseLeadData(contact.metadata);
  const breadcrumbs: BreadcrumbItem[] = [
    { label: "CRM", href: "/crm" },
    { label: "Contacts", href: "/crm/contacts" },
    { label: contact.title },
  ];

  return (
    <div data-testid="contact-detail" className="flex flex-col gap-6">
      <Breadcrumbs items={breadcrumbs} />

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Users className="h-6 w-6 text-primary" />
          <h1 className="text-xl font-bold text-foreground" data-testid="contact-detail-title">
            {contact.title}
          </h1>
        </div>
        <Link
          href={`/crm/contacts/${contactId}/edit`}
          className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm text-foreground hover:bg-accent"
          data-testid="contact-edit-btn"
        >
          <PenLine className="h-4 w-4" />
          Edit
        </Link>
      </div>

      <div className="flex gap-6">
        <div className="min-w-0 flex-1 space-y-6">
          {/* Attributes card */}
          <div className="rounded-lg border border-border p-4" data-testid="contact-attributes">
            <h3 className="mb-3 text-sm font-semibold text-foreground">Contact Details</h3>
            <div className="grid grid-cols-2 gap-3 text-sm">
              {meta.contact_email && <Field label="Email" value={meta.contact_email} />}
              {meta.contact_name && <Field label="Title" value={meta.contact_name} />}
              {meta.assigned_to && <Field label="Owner" value={meta.assigned_to} />}
            </div>
          </div>

          {/* Activity timeline */}
          <div>
            <h2 className="mb-3 text-sm font-semibold text-foreground">
              Activity ({messages.length})
            </h2>
            <MessageTimeline messages={messages} currentUserId={userId ?? undefined} />
          </div>
        </div>

        <aside className="w-72 shrink-0 space-y-4">
          {/* Parent company */}
          {meta.company && (
            <div
              className="rounded-lg border border-border p-4"
              data-testid="contact-company-panel"
            >
              <div className="mb-2 flex items-center gap-2">
                <Building2 className="h-4 w-4 text-muted-foreground" />
                <h3 className="text-sm font-semibold text-foreground">Company</h3>
              </div>
              <Link
                href={`/crm/companies/${meta.customer_org_id ?? ""}`}
                className="text-sm text-primary hover:underline"
                data-testid="contact-company-link"
              >
                {meta.company}
              </Link>
            </div>
          )}

          {/* Opportunities */}
          <div className="rounded-lg border border-border p-4" data-testid="contact-opps-panel">
            <div className="mb-2 flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              <h3 className="text-sm font-semibold text-foreground">
                Opportunities ({opportunities.length})
              </h3>
            </div>
            {opportunities.length === 0 ? (
              <p className="text-xs text-muted-foreground">No opportunities</p>
            ) : (
              <ul className="space-y-1">
                {opportunities.map((opp) => {
                  return (
                    <li key={opp.id} className="flex items-center justify-between">
                      <Link
                        href={`/crm/pipeline/${opp.id}`}
                        className="text-sm text-primary hover:underline"
                        data-testid={`opp-link-${opp.id}`}
                      >
                        {opp.title}
                      </Link>
                      <span className="text-xs text-muted-foreground">
                        {STAGE_LABELS[(opp.stage ?? "new_lead") as PipelineStage] ?? opp.stage}
                      </span>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </aside>
      </div>
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }): React.ReactNode {
  return (
    <div>
      <span className="text-xs text-muted-foreground">{label}</span>
      <p className="font-medium text-foreground">{value}</p>
    </div>
  );
}
