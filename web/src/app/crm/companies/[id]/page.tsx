import { CompanyDetail } from "@/components/crm/company-detail";

interface CompanyDetailPageProps {
  params: Promise<{ id: string }>;
}

/** Company detail page under /crm/companies/[id]. */
export default async function CompanyDetailPage({ params }: CompanyDetailPageProps): Promise<React.ReactNode> {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-7xl p-6">
      <CompanyDetail companyId={id} />
    </div>
  );
}
