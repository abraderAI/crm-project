import { ContactList } from "@/components/crm/contact-list";

/** Contacts list page under /crm/contacts. */
export default function ContactsPage(): React.ReactNode {
  return (
    <div className="mx-auto max-w-7xl p-6">
      <ContactList />
    </div>
  );
}
