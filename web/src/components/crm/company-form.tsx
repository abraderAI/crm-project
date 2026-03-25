"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import { COMPANY_STATUSES, COMPANY_STATUS_LABELS } from "@/lib/crm-types";
import { createCompany, updateCompany, checkDuplicateCompany } from "@/lib/crm-api";
import { useTier } from "@/hooks/use-tier";
import { Breadcrumbs, type BreadcrumbItem } from "@/components/layout/breadcrumbs";

export interface CompanyFormProps {
  /** Existing company data for edit mode. */
  company?: Thread;
  companyId?: string;
}

/** Company create/edit form with real-time duplicate name check. */
export function CompanyForm({ company, companyId }: CompanyFormProps): React.ReactNode {
  const isEdit = Boolean(companyId && company);
  const router = useRouter();
  const { orgId } = useTier();
  const { getToken } = useAuth();

  const [name, setName] = useState(company?.title ?? "");
  const [description, setDescription] = useState(company?.body ?? "");
  const [industry, setIndustry] = useState("");
  const [status, setStatus] = useState("prospect");
  const [website, setWebsite] = useState("");
  const [phone, setPhone] = useState("");
  const [address, setAddress] = useState("");

  const [duplicates, setDuplicates] = useState<Thread[]>([]);
  const [showDuplicateWarning, setShowDuplicateWarning] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Parse existing metadata for edit mode
  useEffect(() => {
    if (company) {
      try {
        const meta = JSON.parse(company.metadata) as Record<string, unknown>;
        if (typeof meta.industry === "string") setIndustry(meta.industry);
        if (typeof meta.status === "string") setStatus(meta.status);
        if (typeof meta.website === "string") setWebsite(meta.website);
        if (typeof meta.phone === "string") setPhone(meta.phone);
        if (typeof meta.address === "string") setAddress(meta.address);
      } catch {
        /* ignore */
      }
    }
  }, [company]);

  // Debounced duplicate check on name change
  const checkDuplicate = useCallback(
    async (nameValue: string): Promise<void> => {
      if (!orgId || nameValue.length < 2) {
        setDuplicates([]);
        return;
      }
      try {
        const token = await getToken();
        if (!token) return;
        const results = await checkDuplicateCompany(token, orgId, nameValue);
        // Exclude current company in edit mode
        const filtered = companyId ? results.filter((r) => r.id !== companyId) : results;
        setDuplicates(filtered);
      } catch {
        /* ignore duplicate check errors */
      }
    },
    [getToken, orgId, companyId],
  );

  const handleNameChange = (value: string): void => {
    setName(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void checkDuplicate(value);
    }, 500);
  };

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    if (!orgId || !name.trim()) return;

    // Show duplicate warning if duplicates exist and user hasn't confirmed
    if (duplicates.length > 0 && !showDuplicateWarning) {
      setShowDuplicateWarning(true);
      return;
    }

    setIsSaving(true);
    setError(null);
    try {
      const token = await getToken();
      if (!token) return;

      const data = {
        title: name.trim(),
        body: description.trim(),
        metadata: JSON.stringify({
          industry,
          status,
          website,
          phone,
          address,
          crm_type: "company",
        }),
      };

      if (isEdit && companyId) {
        await updateCompany(token, orgId, companyId, data);
        router.push(`/crm/companies/${companyId}`);
      } else {
        const created = await createCompany(token, orgId, data);
        router.push(`/crm/companies/${created.id}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save company");
    } finally {
      setIsSaving(false);
    }
  };

  const breadcrumbs: BreadcrumbItem[] = [
    { label: "CRM", href: "/crm" },
    { label: "Companies", href: "/crm/companies" },
    { label: isEdit ? "Edit" : "New Company" },
  ];

  return (
    <div data-testid="company-form" className="flex flex-col gap-6">
      <Breadcrumbs items={breadcrumbs} />
      <h1 className="text-xl font-bold text-foreground">
        {isEdit ? "Edit Company" : "New Company"}
      </h1>

      {error && (
        <div
          data-testid="company-form-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      <form onSubmit={(e) => void handleSubmit(e)} className="max-w-lg space-y-4">
        <div>
          <label htmlFor="company-name" className="mb-1 block text-sm font-medium text-foreground">
            Name *
          </label>
          <input
            id="company-name"
            type="text"
            value={name}
            onChange={(e) => handleNameChange(e.target.value)}
            required
            data-testid="company-name-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          {duplicates.length > 0 && (
            <div
              data-testid="company-duplicate-warning"
              className="mt-1 rounded-md bg-amber-50 p-2 text-xs text-amber-700"
            >
              <p className="font-medium">Possible duplicates found:</p>
              <ul className="mt-1 list-disc pl-4">
                {duplicates.map((d) => (
                  <li key={d.id}>{d.title}</li>
                ))}
              </ul>
            </div>
          )}
        </div>

        <div>
          <label
            htmlFor="company-industry"
            className="mb-1 block text-sm font-medium text-foreground"
          >
            Industry
          </label>
          <input
            id="company-industry"
            type="text"
            value={industry}
            onChange={(e) => setIndustry(e.target.value)}
            data-testid="company-industry-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label
            htmlFor="company-status"
            className="mb-1 block text-sm font-medium text-foreground"
          >
            Status
          </label>
          <select
            id="company-status"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            data-testid="company-status-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm"
          >
            {COMPANY_STATUSES.map((s) => (
              <option key={s} value={s}>
                {COMPANY_STATUS_LABELS[s]}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label
            htmlFor="company-website"
            className="mb-1 block text-sm font-medium text-foreground"
          >
            Website
          </label>
          <input
            id="company-website"
            type="url"
            value={website}
            onChange={(e) => setWebsite(e.target.value)}
            data-testid="company-website-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label htmlFor="company-phone" className="mb-1 block text-sm font-medium text-foreground">
            Phone
          </label>
          <input
            id="company-phone"
            type="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            data-testid="company-phone-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label
            htmlFor="company-address"
            className="mb-1 block text-sm font-medium text-foreground"
          >
            Address
          </label>
          <input
            id="company-address"
            type="text"
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            data-testid="company-address-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label
            htmlFor="company-description"
            className="mb-1 block text-sm font-medium text-foreground"
          >
            Description
          </label>
          <textarea
            id="company-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            data-testid="company-description-input"
            rows={3}
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Duplicate confirmation dialog */}
        {showDuplicateWarning && (
          <div
            data-testid="company-duplicate-confirm"
            className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-800"
          >
            <p className="font-medium">Possible duplicate companies exist. Continue anyway?</p>
            <div className="mt-2 flex gap-2">
              <button
                type="submit"
                className="rounded-md bg-amber-600 px-3 py-1 text-sm text-white hover:bg-amber-700"
                data-testid="company-duplicate-confirm-yes"
              >
                Yes, create anyway
              </button>
              <button
                type="button"
                onClick={() => setShowDuplicateWarning(false)}
                className="rounded-md border border-border px-3 py-1 text-sm hover:bg-accent"
                data-testid="company-duplicate-confirm-no"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={isSaving || !name.trim()}
            data-testid="company-submit-btn"
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {isSaving ? "Saving..." : isEdit ? "Update Company" : "Create Company"}
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
