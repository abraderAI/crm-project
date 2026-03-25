"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";
import { AlertTriangle } from "lucide-react";

import type { Thread } from "@/lib/api-types";
import { createContact, updateContact, checkDuplicateContact, fetchCompanies } from "@/lib/crm-api";
import { useTier } from "@/hooks/use-tier";
import { Breadcrumbs, type BreadcrumbItem } from "@/components/layout/breadcrumbs";

export interface ContactFormProps {
  contact?: Thread;
  contactId?: string;
}

/** Contact create/edit form with company picker and email duplicate check. */
export function ContactForm({ contact, contactId }: ContactFormProps): React.ReactNode {
  const isEdit = Boolean(contactId && contact);
  const router = useRouter();
  const { orgId } = useTier();
  const { getToken } = useAuth();

  const [name, setName] = useState(contact?.title ?? "");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [title, setTitle] = useState("");
  const [companyId, setCompanyId] = useState("");
  const [companySearch, setCompanySearch] = useState("");
  const [companies, setCompanies] = useState<Thread[]>([]);
  const [showDropdown, setShowDropdown] = useState(false);

  const [duplicates, setDuplicates] = useState<Thread[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (contact) {
      try {
        const meta = JSON.parse(contact.metadata) as Record<string, unknown>;
        if (typeof meta.email === "string") setEmail(meta.email);
        if (typeof meta.phone === "string") setPhone(meta.phone);
        if (typeof meta.title === "string") setTitle(meta.title);
        if (typeof meta.company_id === "string") setCompanyId(meta.company_id);
        if (typeof meta.company === "string") setCompanySearch(meta.company);
      } catch {
        /* ignore */
      }
    }
  }, [contact]);

  // Search companies for picker
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
        setShowDropdown(true);
      } catch {
        /* ignore */
      }
    },
    [getToken, orgId],
  );

  const handleCompanySearch = (value: string): void => {
    setCompanySearch(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void searchCompanies(value);
    }, 300);
  };

  const selectCompany = (c: Thread): void => {
    setCompanyId(c.id);
    setCompanySearch(c.title);
    setShowDropdown(false);
  };

  // Email duplicate check
  const checkEmailDuplicate = useCallback(
    async (emailValue: string): Promise<void> => {
      if (!orgId || emailValue.length < 3) {
        setDuplicates([]);
        return;
      }
      try {
        const token = await getToken();
        if (!token) return;
        const results = await checkDuplicateContact(token, orgId, emailValue);
        const filtered = contactId ? results.filter((r) => r.id !== contactId) : results;
        setDuplicates(filtered);
      } catch {
        /* ignore */
      }
    },
    [getToken, orgId, contactId],
  );

  const handleEmailChange = (value: string): void => {
    setEmail(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      void checkEmailDuplicate(value);
    }, 500);
  };

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();
    if (!orgId || !name.trim()) return;

    setIsSaving(true);
    setError(null);
    try {
      const token = await getToken();
      if (!token) return;

      const data = {
        title: name.trim(),
        metadata: JSON.stringify({
          email,
          phone,
          title: title,
          company_id: companyId || undefined,
          company: companySearch || undefined,
          crm_type: "contact",
        }),
        ...(companyId ? { company_id: companyId } : {}),
      };

      if (isEdit && contactId) {
        await updateContact(token, orgId, contactId, data);
        router.push(`/crm/contacts/${contactId}`);
      } else {
        const created = await createContact(token, orgId, data);
        router.push(`/crm/contacts/${created.id}`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save contact");
    } finally {
      setIsSaving(false);
    }
  };

  const breadcrumbs: BreadcrumbItem[] = [
    { label: "CRM", href: "/crm" },
    { label: "Contacts", href: "/crm/contacts" },
    { label: isEdit ? "Edit" : "New Contact" },
  ];

  return (
    <div data-testid="contact-form" className="flex flex-col gap-6">
      <Breadcrumbs items={breadcrumbs} />
      <h1 className="text-xl font-bold text-foreground">
        {isEdit ? "Edit Contact" : "New Contact"}
      </h1>

      {error && (
        <div
          data-testid="contact-form-error"
          className="flex items-center gap-2 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          <AlertTriangle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      <form onSubmit={(e) => void handleSubmit(e)} className="max-w-lg space-y-4">
        <div>
          <label htmlFor="contact-name" className="mb-1 block text-sm font-medium text-foreground">
            Name *
          </label>
          <input
            id="contact-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            data-testid="contact-name-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label htmlFor="contact-email" className="mb-1 block text-sm font-medium text-foreground">
            Email
          </label>
          <input
            id="contact-email"
            type="email"
            value={email}
            onChange={(e) => handleEmailChange(e.target.value)}
            data-testid="contact-email-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          {duplicates.length > 0 && (
            <div
              data-testid="contact-duplicate-warning"
              className="mt-1 rounded-md bg-amber-50 p-2 text-xs text-amber-700"
            >
              <p className="font-medium">Duplicate email found:</p>
              <ul className="mt-1 list-disc pl-4">
                {duplicates.map((d) => (
                  <li key={d.id}>{d.title}</li>
                ))}
              </ul>
            </div>
          )}
        </div>

        <div>
          <label htmlFor="contact-phone" className="mb-1 block text-sm font-medium text-foreground">
            Phone
          </label>
          <input
            id="contact-phone"
            type="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            data-testid="contact-phone-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        <div>
          <label htmlFor="contact-title" className="mb-1 block text-sm font-medium text-foreground">
            Job Title
          </label>
          <input
            id="contact-title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            data-testid="contact-title-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {/* Company picker */}
        <div className="relative">
          <label
            htmlFor="contact-company"
            className="mb-1 block text-sm font-medium text-foreground"
          >
            Company
          </label>
          <input
            id="contact-company"
            type="text"
            value={companySearch}
            onChange={(e) => handleCompanySearch(e.target.value)}
            placeholder="Search companies..."
            data-testid="contact-company-input"
            className="h-9 w-full rounded-md border border-border bg-background px-3 text-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            onFocus={() => {
              if (companies.length > 0) setShowDropdown(true);
            }}
            onBlur={() => setTimeout(() => setShowDropdown(false), 200)}
          />
          {showDropdown && companies.length > 0 && (
            <ul
              data-testid="contact-company-dropdown"
              className="absolute z-10 mt-1 max-h-40 w-full overflow-auto rounded-md border border-border bg-background shadow-md"
            >
              {companies.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onMouseDown={() => selectCompany(c)}
                    className="w-full px-3 py-2 text-left text-sm hover:bg-accent"
                    data-testid={`company-option-${c.id}`}
                  >
                    {c.title}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={isSaving || !name.trim()}
            data-testid="contact-submit-btn"
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {isSaving ? "Saving..." : isEdit ? "Update Contact" : "Create Contact"}
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
