import { CompanyForm } from "@/components/crm/company-form";

/** Create new company page. */
export default function NewCompanyPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-5xl p-6">
      <CompanyForm />
    </div>
  );
}
