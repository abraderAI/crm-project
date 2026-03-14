"use client";

import { EntityList, type EntityListItem, type EntityListProps } from "./entity-list";

export interface EntityListLinkedProps extends Omit<EntityListProps, "getHref"> {
  /** Base path prefix — hrefs are built as `${hrefPrefix}/${item.slug}`. */
  hrefPrefix: string;
}

/** EntityList wrapper that builds hrefs from a prefix, safe for server→client boundary. */
export function EntityListLinked({ hrefPrefix, ...props }: EntityListLinkedProps): React.ReactNode {
  return <EntityList {...props} getHref={(item: EntityListItem) => `${hrefPrefix}/${item.slug}`} />;
}
