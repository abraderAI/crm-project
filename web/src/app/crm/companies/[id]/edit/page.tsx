import { CompanyForm } from "@/components/crm/company-form";

interface EditCompanyPageProps {
  params: Promise<{ id: string }>;
}

/** Edit company page. */
export default async function EditCompanyPage({ params }: EditCompanyPageProps): Promise<React.ReactNode> {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-5xl p-6">
      <CompanyForm companyId={id} />
    </div>
  );
}
