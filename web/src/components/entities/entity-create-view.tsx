"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";

import { EntityForm, type EntityFormValues } from "./entity-form";
import type { EntityType } from "./entity-card";
import { createOrg, createSpace, createBoard } from "@/lib/entity-api";

export interface EntityCreateViewProps {
  /** Entity kind (org, space, board). */
  entityKind: EntityType;
  /** Parent org slug (required for space and board creation). */
  orgSlug?: string;
  /** Parent space slug (required for board creation). */
  spaceSlug?: string;
  /** Path to navigate to on cancel. */
  cancelHref: string;
}

/** Dispatch to the correct entity-api create function and return the redirect path. */
async function dispatchCreate(
  entityKind: EntityType,
  token: string,
  values: EntityFormValues,
  orgSlug?: string,
  spaceSlug?: string,
): Promise<string> {
  switch (entityKind) {
    case "org": {
      const org = await createOrg(token, values);
      return `/orgs/${org.slug}`;
    }
    case "space": {
      if (!orgSlug) throw new Error("orgSlug required for space creation");
      const space = await createSpace(token, orgSlug, values);
      return `/orgs/${orgSlug}/spaces/${space.slug}`;
    }
    case "board": {
      if (!orgSlug || !spaceSlug)
        throw new Error("orgSlug and spaceSlug required for board creation");
      const board = await createBoard(token, orgSlug, spaceSlug, values);
      return `/orgs/${orgSlug}/spaces/${spaceSlug}/boards/${board.slug}`;
    }
  }
}

/** Client component wiring EntityForm (create mode) to mutation API with auth + router. */
export function EntityCreateView({
  entityKind,
  orgSlug,
  spaceSlug,
  cancelHref,
}: EntityCreateViewProps): React.ReactNode {
  const router = useRouter();
  const { getToken } = useAuth();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: EntityFormValues): Promise<void> => {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      const redirect = await dispatchCreate(entityKind, token, values, orgSlug, spaceSlug);
      router.push(redirect);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl p-6">
      <EntityForm
        mode="create"
        entityKind={entityKind}
        onSubmit={handleSubmit}
        onCancel={() => router.push(cancelHref)}
        loading={loading}
      />
    </div>
  );
}
