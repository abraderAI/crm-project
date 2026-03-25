import { ContactForm } from "@/components/crm/contact-form";

interface EditContactPageProps {
  params: Promise<{ id: string }>;
}

/** Edit contact page. */
export default async function EditContactPage({ params }: EditContactPageProps): Promise<React.ReactNode> {
  const { id } = await params;
  return (
    <div className="mx-auto max-w-5xl p-6">
      <ContactForm contactId={id} />
    </div>
  );
}
