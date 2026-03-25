import { ContactForm } from "@/components/crm/contact-form";

/** Create new contact page. */
export default function NewContactPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-5xl p-6">
      <ContactForm />
    </div>
  );
}
