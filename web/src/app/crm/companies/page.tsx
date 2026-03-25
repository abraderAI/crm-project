import { CompanyList } from "@/components/crm/company-list";

/** Companies list page under /crm/companies. */
export default function CompaniesPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-7xl p-6">
      <CompanyList />
    </div>
  );
}
