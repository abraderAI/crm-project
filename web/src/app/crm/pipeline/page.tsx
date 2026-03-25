import { redirect } from "next/navigation";

/** Pipeline page redirects to the CRM root which renders the Kanban. */
export default function PipelinePage(): never {
  redirect("/crm");
}
