"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import {
  PIPELINE_STAGES,
  STAGE_LABELS,
  OPPORTUNITY_TYPES,
  OPPORTUNITY_TYPE_LABELS,
  LEAD_SOURCES,
  LEAD_SOURCE_LABELS,
  calculateWeightedForecast,
  getEffectiveProbability,
  formatCurrency,
  type PipelineStage,
} from "@/lib/crm-types";
import { createOpportunity, updateOpportunity, fetchCompanies, fetchContacts } from "@/lib/crm-api";
import { useTier } from "@/hooks/use-tier";
import { Breadcrumbs, type BreadcrumbItem } from "@/components/layout/breadcrumbs";

export interface OpportunityFormProps {
  opportunity?: Thread;
  opportunityId?: string;
}

/** Opportunity create/edit form. Company is required. */
export function OpportunityForm({
  opportunity,
  opportunityId,
}: OpportunityFormProps): React.ReactNode {
  const isEdit = Boolean(opportunityId && opportunity);
  const router = useRouter();
  const { orgId } = useTier();
  const { getToken } = useAuth();

  const [name, setName] = useState(opportunity?.title ?? "");
  const [companyId, setCompanyId] = useState("");
  const [companySearch, setCompanySearch] = useState("");
  const [companies, setCompanies] = useState<Thread[]>([]);
  const [showCompanyDropdown, setShowCompanyDropdown] = useState(false);
  const [contactIds, setContactIds] = useState<string[]>([]);
  const [primaryContactId, setPrimaryContactId] = useState("");
  const [contactSearch, setContactSearch] = useState("");
  const [contactResults, setContactResults] = useState<Thread[]>([]);
  const [showContactDropdown, setShowContactDropdown] = useState(false);
  const [dealAmount, setDealAmount] = useState("");
  const [expectedCloseDate, setExpectedCloseDate] = useState("");
  const [opportunityType, setOpportunityType] = useState<string>("");
  const [leadSource, setLeadSource] = useState<string>("");
  const [stage, setStage] = useState<PipelineStage>("new_lead");
  const [probabilityOverride, setProbabilityOverride] = useState("");

  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [companyError, setCompanyError] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (opportunity) {
      try {
        const meta = JSON.parse(opportunity.metadata) as Record<string, unknown>;
        if (typeof meta.company_id === "string") setCompanyId(meta.company_id);
        if (typeof meta.company === "string") setCompanySearch(meta.company);
        if (typeof meta.deal_amount === "number") setDealAmount(String(meta.deal_amount));
        if (typeof meta.expected_close_date === "string")
          setExpectedCloseDate(meta.expected_close_date);
        if (typeof meta.opportunity_type === "string") setOpportunityType(meta.opportunity_type);
        if (typeof meta.lead_source === "string") setLeadSource(meta.lead_source);
        if (typeof meta.probability_override === "number")
          setProbabilityOverride(String(meta.probability_override));
        if (Array.isArray(meta.contact_ids)) setContactIds(meta.contact_ids as string[]);
        if (typeof meta.primary_contact_id === "string")
          setPrimaryContactId(meta.primary_contact_id);
      } catch {
        /* ignore */
      }
      if (opportunity.stage) setStage(opportunity.stage as PipelineStage);
    }
  }, [opportunity]);

  const searchCompanies = useCallback(
    async (query: string): Promise<void> => {
      if (!orgId || query.length < 1) {
        setCompanies([]);
        return;
      }
      try {
        const token = await getToken();
        if (!token) return;
        const result = await fetchCompanies(token, orgId, { search: query, limit: "10" });
        setCompanies(result.data);
        setShowCompanyDropdown(true);
      } catch {
        /* ignore */
      }
    },
    [getToken, orgId],
  );

  const searchContacts = useCallback(
    async (query: string): Promise<void> => {
      if (!orgId || query.length < 1) {
        setContactResults([]);
        return;
      }
      try {
        const token = await getToken();
        if (!token) return;
        const result = await fetchContacts(token, orgId, { search: query, limit: "10" });
        setContactResults(result.data);
        setShowContactDropdown(true);
      } catch {
        /* ignore */
      }
    },
    [getToken, orgId],
  );

  const handleCompanySearch = (value: string): void => {
    setCompanySearch(value);
    setCompanyError(false);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void searchCompanies(value);
    }, 300);
  };

  const handleContactSearch = (value: string): void => {
    setContactSearch(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void searchContacts(value);
    }, 300);
  };

  // Weighted forecast preview
  const amount = parseFloat(dealAmount) || 0;
  const probability = getEffectiveProbability(
    stage,
    probabilityOverride ? parseInt(probabilityOverride, 10) : undefined,
  );
  const forecast = calculateWeightedForecast(amount, probability);

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    if (!orgId || !name.trim()) return;
    if (!companyId) {
      setCompanyError(true);
      return;
    }

    setIsSaving(true);
    setError(null);
    try {
      const token = await getToken();
      if (!token) return;

      const data = {
        title: name.trim(),
        metadata: JSON.stringify({
          company_id: companyId,
          company: companySearch,
          deal_amount: amount || undefined,
          expected_close_date: expectedCloseDate || undefined,
          opportunity_type: opportunityType || undefined,
          lead_source: leadSource || undefined,
          probability_override: probabilityOverride ? parseInt(probabilityOverride, 10) : undefined,
          contact_ids: contactIds.length > 0 ? contactIds : undefined,
          primary_contact_id: primaryContactId || undefined,
          crm_type: "opportunity",
        }),
        stage,
        company_id: companyId,
        contact_ids: contactIds,
        primary_contact_id: primaryContactId || undefined,
      };

      if (isEdit && opportunityId) {
        await updateOpportunity(token, orgId, opportunityId, data);
        router.push(`/crm/pipeline/${opportunityId}`);
      } else {
        const created = await createOpportunity(token, orgId, data);
        router.push(`/crm/pipeline/${created.id}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save opportunity");
    } finally {
      setIsSaving(false);
    }
  };

  const breadcrumbs: BreadcrumbItem[] = [
    { label: "CRM", href: "/crm" },
    { label: "Pipeline", href: "/crm/pipeline" },
    { label: isEdit ? "Edit" : "New Opportunity" },
  ];

  return (
    <div data-testid="opportunity-form" className="flex flex-col gap-6">
      <Breadcrumbs items={breadcrumbs} />
      <h1 className="text-xl font-bold text-foreground">
        {isEdit ? "Edit Opportunity" : "New Opportunity"}
      </h1>

      {error && (
        <div
          data-testid="opportunity-form-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      <form onSubmit={(e) => void handleSubmit(e)} className="max-w-lg space-y-4">
        <div>
          <label htmlFor="opp-name" className="mb-1 block text-sm font-medium text-foreground">
            Name *
          </label>
          <input
            id="opp-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            data-testid="opp-name-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Company picker (required) */}
        <div className="relative">
          <label htmlFor="opp-company" className="mb-1 block text-sm font-medium text-foreground">
            Company *
          </label>
          <input
            id="opp-company"
            type="text"
            value={companySearch}
            onChange={(e) => handleCompanySearch(e.target.value)}
            placeholder="Search companies..."
            data-testid="opp-company-input"
            className={`h-9 w-full rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-1 ${companyError ? "border-red-500 focus:border-red-500 focus:ring-red-500" : "border-border focus:border-primary focus:ring-primary"}`}
            onFocus={() => {
              if (companies.length > 0) setShowCompanyDropdown(true);
            }}
            onBlur={() => setTimeout(() => setShowCompanyDropdown(false), 200)}
          />
          {companyError && (
            <p className="mt-1 text-xs text-red-600" data-testid="opp-company-error">
              Company is required
            </p>
          )}
          {showCompanyDropdown && companies.length > 0 && (
            <ul
              data-testid="opp-company-dropdown"
              className="absolute z-10 mt-1 max-h-40 w-full overflow-auto rounded-md border border-border bg-background shadow-md"
            >
              {companies.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onMouseDown={() => {
                      setCompanyId(c.id);
                      setCompanySearch(c.title);
                      setShowCompanyDropdown(false);
                      setCompanyError(false);
                    }}
                    className="w-full px-3 py-2 text-left text-sm hover:bg-accent"
                  >
                    {c.title}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* Contact picker */}
        <div className="relative">
          <label htmlFor="opp-contact" className="mb-1 block text-sm font-medium text-foreground">
            Contacts
          </label>
          <input
            id="opp-contact"
            type="text"
            value={contactSearch}
            onChange={(e) => handleContactSearch(e.target.value)}
            placeholder="Search contacts..."
            data-testid="opp-contact-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            onFocus={() => {
              if (contactResults.length > 0) setShowContactDropdown(true);
            }}
            onBlur={() => setTimeout(() => setShowContactDropdown(false), 200)}
          />
          {showContactDropdown && contactResults.length > 0 && (
            <ul
              data-testid="opp-contact-dropdown"
              className="absolute z-10 mt-1 max-h-40 w-full overflow-auto rounded-md border border-border bg-background shadow-md"
            >
              {contactResults.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onMouseDown={() => {
                      if (!contactIds.includes(c.id)) {
                        setContactIds((prev) => [...prev, c.id]);
                        if (!primaryContactId) setPrimaryContactId(c.id);
                      }
                      setContactSearch("");
                      setShowContactDropdown(false);
                    }}
                    className="w-full px-3 py-2 text-left text-sm hover:bg-accent"
                  >
                    {c.title}
                  </button>
                </li>
              ))}
            </ul>
          )}
          {contactIds.length > 0 && (
            <div className="mt-1 flex flex-wrap gap-1" data-testid="opp-selected-contacts">
              {contactIds.map((id) => (
                <span
                  key={id}
                  className="inline-flex items-center gap-1 rounded bg-muted px-2 py-0.5 text-xs"
                >
                  {id === primaryContactId ? "★ " : ""}
                  {id}
                  <button
                    type="button"
                    onClick={() => {
                      setContactIds((prev) => prev.filter((x) => x !== id));
                      if (primaryContactId === id) setPrimaryContactId("");
                    }}
                    className="ml-1 text-muted-foreground hover:text-foreground"
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="opp-amount" className="mb-1 block text-sm font-medium text-foreground">
              Deal Amount (USD)
            </label>
            <input
              id="opp-amount"
              type="number"
              min={0}
              step="0.01"
              value={dealAmount}
              onChange={(e) => setDealAmount(e.target.value)}
              data-testid="opp-amount-input"
              className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div>
            <label
              htmlFor="opp-close-date"
              className="mb-1 block text-sm font-medium text-foreground"
            >
              Expected Close Date
            </label>
            <input
              id="opp-close-date"
              type="date"
              value={expectedCloseDate}
              onChange={(e) => setExpectedCloseDate(e.target.value)}
              data-testid="opp-close-date-input"
              className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="opp-type" className="mb-1 block text-sm font-medium text-foreground">
              Opportunity Type
            </label>
            <select
              id="opp-type"
              value={opportunityType}
              onChange={(e) => setOpportunityType(e.target.value)}
              data-testid="opp-type-input"
              className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm"
            >
              <option value="">Select type...</option>
              {OPPORTUNITY_TYPES.map((t) => (
                <option key={t} value={t}>
                  {OPPORTUNITY_TYPE_LABELS[t]}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label htmlFor="opp-source" className="mb-1 block text-sm font-medium text-foreground">
              Lead Source
            </label>
            <select
              id="opp-source"
              value={leadSource}
              onChange={(e) => setLeadSource(e.target.value)}
              data-testid="opp-source-input"
              className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm"
            >
              <option value="">Select source...</option>
              {LEAD_SOURCES.map((s) => (
                <option key={s} value={s}>
                  {LEAD_SOURCE_LABELS[s]}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="opp-stage" className="mb-1 block text-sm font-medium text-foreground">
              Stage
            </label>
            <select
              id="opp-stage"
              value={stage}
              onChange={(e) => setStage(e.target.value as PipelineStage)}
              data-testid="opp-stage-input"
              className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm"
            >
              {PIPELINE_STAGES.map((s) => (
                <option key={s} value={s}>
                  {STAGE_LABELS[s]}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label
              htmlFor="opp-probability"
              className="mb-1 block text-sm font-medium text-foreground"
            >
              Close Probability % (0-100)
            </label>
            <input
              id="opp-probability"
              type="number"
              min={0}
              max={100}
              value={probabilityOverride}
              onChange={(e) => setProbabilityOverride(e.target.value)}
              data-testid="opp-probability-input"
              className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              placeholder={`Default: ${getEffectiveProbability(stage)}%`}
            />
          </div>
        </div>

        {/* Weighted forecast display */}
        {amount > 0 && (
          <div data-testid="opp-forecast-preview" className="rounded-md bg-muted p-3 text-sm">
            <span className="text-muted-foreground">Weighted Forecast: </span>
            <span className="font-medium text-foreground">{formatCurrency(forecast)}</span>
            <span className="ml-2 text-xs text-muted-foreground">
              ({probability}% of {formatCurrency(amount)})
            </span>
          </div>
        )}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={isSaving || !name.trim()}
            data-testid="opp-submit-btn"
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {isSaving ? "Saving..." : isEdit ? "Update Opportunity" : "Create Opportunity"}
          </button>
          <button
            type="button"
            onClick={() => router.back()}
            className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
