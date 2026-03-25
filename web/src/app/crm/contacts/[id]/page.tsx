import { ContactDetail } from "@/components/crm/contact-detail";

interface ContactDetailPageProps {
  params: Promise<{ id: string }>;
}

/** Contact detail page under /crm/contacts/[id]. */
export default async function ContactDetailPage({ params }: ContactDetailPageProps): Promise<React.ReactNode> {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-7xl p-6">
      <ContactDetail contactId={id} />
    </div>
  );
}
