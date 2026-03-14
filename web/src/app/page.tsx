import Link from "next/link";
import { BarChart3, Bell, Search } from "lucide-react";

import { fetchOrgs } from "@/lib/user-api";
import { EntityListLinked } from "@/components/entities/entity-list-linked";

const QUICK_NAV = [
  { href: "/crm", label: "CRM Pipeline", icon: BarChart3, description: "Manage leads and deals" },
  { href: "/notifications", label: "Notifications", icon: Bell, description: "Recent activity" },
  { href: "/search", label: "Search", icon: Search, description: "Find anything" },
] as const;

export default async function Home(): Promise<React.ReactNode> {
  const { data: orgs } = await fetchOrgs();

  return (
    <div className="mx-auto max-w-5xl space-y-8 p-6">
      {/* Quick-nav cards */}
      <section>
        <h2 className="mb-4 text-lg font-semibold text-foreground">Quick Access</h2>
        <div className="grid gap-3 sm:grid-cols-3">
          {QUICK_NAV.map(({ href, label, icon: Icon, description }) => (
            <Link
              key={href}
              href={href}
              className="flex items-start gap-3 rounded-lg border border-border bg-background p-4 transition-colors hover:bg-accent/50"
            >
              <Icon className="mt-0.5 h-5 w-5 text-primary" />
              <div>
                <span className="text-sm font-medium text-foreground">{label}</span>
                <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Org list */}
      <section>
        <EntityListLinked
          entityType="org"
          title="Organizations"
          items={orgs.map((o) => ({
            id: o.id,
            name: o.name,
            slug: o.slug,
            description: o.description,
          }))}
          hrefPrefix="/orgs"
        />
      </section>
    </div>
  );
}
