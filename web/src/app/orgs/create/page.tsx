import { EntityCreateView } from "@/components/entities/entity-create-view";

/** Page for creating a new organization. */
export default function CreateOrgPage(): React.ReactNode {
  return <EntityCreateView entityKind="org" cancelHref="/" />;
}
