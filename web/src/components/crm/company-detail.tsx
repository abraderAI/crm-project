"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useAuth } from "@clerk/nextjs";
import { Building2, Users, TrendingUp, AlertTriangle, PenLine } from "lucide-react";

import type { Thread, Message } from "@/lib/api-types";
import { parseLeadData, formatCurrency } from "@/lib/crm-types";
import {
  fetchCompany,
  fetchLinkedContacts,
  fetchLinkedOpportunities,
  fetchEntityMessages,
} from "@/lib/crm-api";
import { useTier } from "@/hooks/use-tier";
import { MessageTimeline } from "@/components/thread/message-timeline";
import { Breadcrumbs, type BreadcrumbItem } from "@/components/layout/breadcrumbs";

export interface CompanyDetailProps {
  companyId: string;
}

/** Company detail view: attributes card, linked contacts, opportunities, activity timeline. */
export function CompanyDetail({ companyId }: CompanyDetailProps): React.ReactNode {
  const { orgId } = useTier();
  const { getToken, userId } = useAuth();

  const [company, setCompany] = useState<Thread | null>(null);
  const [contacts, setContacts] = useState<Thread[]>([]);
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

      const [companyData, contactsData, oppsData, msgsData] = await Promise.all([
        fetchCompany(token, orgId, companyId),
        fetchLinkedContacts(token, orgId, companyId),
        fetchLinkedOpportunities(token, orgId, "companies", companyId),
        fetchEntityMessages(token, orgId, "companies", companyId),
      ]);

      if (!mountedRef.current) return;
      setCompany(companyData);
      setContacts(contactsData.data);
      setOpportunities(oppsData.data);
      setMessages(msgsData.data);
    } catch (err) {
      if (mountedRef.current) {
        setError(err instanceof Error ? err.message : "Failed to load company");
      }
    } finally {
      if (mountedRef.current) setIsLoading(false);
    }
  }, [getToken, orgId, companyId]);

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
        data-testid="company-detail-loading"
        className="py-8 text-center text-sm text-muted-foreground"
      >
        Loading company...
      </div>
    );
  }

  if (error || !company) {
    return (
      <div data-testid="company-detail-error" className="flex flex-col items-center gap-3 py-12">
        <AlertTriangle className="h-8 w-8 text-muted-foreground" />
        <p className="text-sm text-foreground">{error ?? "Company not found"}</p>
        <Link href="/crm/companies" className="text-sm text-primary hover:underline">
          Back to companies
        </Link>
      </div>
    );
  }

  const meta = parseLeadData(company.metadata);
  const breadcrumbs: BreadcrumbItem[] = [
    { label: "CRM", href: "/crm" },
    { label: "Companies", href: "/crm/companies" },
    { label: company.title },
  ];

  const openOpps = opportunities.filter(
    (o) => o.stage !== "closed_won" && o.stage !== "closed_lost",
  );
  const closedOpps = opportunities.filter(
    (o) => o.stage === "closed_won" || o.stage === "closed_lost",
  );

  return (
    <div data-testid="company-detail" className="flex flex-col gap-6">
      <Breadcrumbs items={breadcrumbs} />

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Building2 className="h-6 w-6 text-primary" />
          <h1 className="text-xl font-bold text-foreground" data-testid="company-detail-title">
            {company.title}
          </h1>
        </div>
        <Link
          href={`/crm/companies/${companyId}/edit`}
          className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm text-foreground hover:bg-accent"
          data-testid="company-edit-btn"
        >
          <PenLine className="h-4 w-4" />
          Edit
        </Link>
      </div>

      <div className="flex gap-6">
        {/* Main content */}
        <div className="min-w-0 flex-1 space-y-6">
          {/* Attributes card */}
          <div className="rounded-lg border border-border p-4" data-testid="company-attributes">
            <h3 className="mb-3 text-sm font-semibold text-foreground">Company Details</h3>
            <div className="grid grid-cols-2 gap-3 text-sm">
              {meta.source && <Field label="Industry" value={meta.source} />}
              {meta.assigned_to && <Field label="Owner" value={meta.assigned_to} />}
              {meta.contact_email && <Field label="Website" value={meta.contact_email} />}
              {company.body && (
                <div className="col-span-2">
                  <Field label="Description" value={company.body} />
                </div>
              )}
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

        {/* Sidebar */}
        <aside className="w-72 shrink-0 space-y-4">
          {/* Linked contacts */}
          <div className="rounded-lg border border-border p-4" data-testid="company-contacts-panel">
            <div className="mb-3 flex items-center gap-2">
              <Users className="h-4 w-4 text-muted-foreground" />
              <h3 className="text-sm font-semibold text-foreground">
                Contacts ({contacts.length})
              </h3>
            </div>
            {contacts.length === 0 ? (
              <p className="text-xs text-muted-foreground" data-testid="company-contacts-empty">
                0 visible contacts
              </p>
            ) : (
              <ul className="space-y-1">
                {contacts.map((c) => (
                  <li key={c.id}>
                    <Link
                      href={`/crm/contacts/${c.id}`}
                      className="text-sm text-primary hover:underline"
                      data-testid={`contact-link-${c.id}`}
                    >
                      {c.title}
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Opportunity summary */}
          <div className="rounded-lg border border-border p-4" data-testid="company-opps-panel">
            <div className="mb-3 flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              <h3 className="text-sm font-semibold text-foreground">
                Opportunities ({opportunities.length})
              </h3>
            </div>
            <div className="space-y-1 text-xs text-muted-foreground">
              <p data-testid="company-opps-open">Open: {openOpps.length}</p>
              <p data-testid="company-opps-closed">Closed: {closedOpps.length}</p>
              {opportunities.length > 0 && (
                <p data-testid="company-opps-value">
                  Total value:{" "}
                  {formatCurrency(
                    opportunities.reduce(
                      (sum, o) => sum + (parseLeadData(o.metadata).deal_amount ?? 0),
                      0,
                    ),
                  )}
                </p>
              )}
            </div>
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
