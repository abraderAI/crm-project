"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@clerk/nextjs";

import type { EntityFormValues } from "./entity-form";
import type { EntityType } from "./entity-card";
import { EntitySettings } from "./entity-settings";
import {
  updateOrg,
  deleteOrg,
  updateSpace,
  deleteSpace,
  updateBoard,
  deleteBoard,
} from "@/lib/entity-api";

export interface EntitySettingsViewProps {
  /** Entity type (org, space, board). */
  entityType: EntityType;
  /** Current entity slug. */
  entitySlug: string;
  /** Parent org slug (required for space and board). */
  orgSlug?: string;
  /** Parent space slug (required for board). */
  spaceSlug?: string;
  /** Current entity values. */
  currentValues: EntityFormValues & { id: string; slug: string };
  /** Path to navigate to after deletion. */
  deleteRedirect: string;
  /** Path to navigate back to on cancel. */
  cancelHref: string;
}

/** Dispatch save to the correct entity-api update function. */
async function dispatchSave(
  entityType: EntityType,
  token: string,
  entitySlug: string,
  values: EntityFormValues,
  orgSlug?: string,
  spaceSlug?: string,
): Promise<void> {
  switch (entityType) {
    case "org":
      await updateOrg(token, entitySlug, values);
      return;
    case "space":
      if (!orgSlug) throw new Error("orgSlug required for space update");
      await updateSpace(token, orgSlug, entitySlug, values);
      return;
    case "board":
      if (!orgSlug || !spaceSlug)
        throw new Error("orgSlug and spaceSlug required for board update");
      await updateBoard(token, orgSlug, spaceSlug, entitySlug, values);
      return;
  }
}

/** Dispatch delete to the correct entity-api delete function. */
async function dispatchDelete(
  entityType: EntityType,
  token: string,
  entitySlug: string,
  orgSlug?: string,
  spaceSlug?: string,
): Promise<void> {
  switch (entityType) {
    case "org":
      await deleteOrg(token, entitySlug);
      return;
    case "space":
      if (!orgSlug) throw new Error("orgSlug required for space delete");
      await deleteSpace(token, orgSlug, entitySlug);
      return;
    case "board":
      if (!orgSlug || !spaceSlug)
        throw new Error("orgSlug and spaceSlug required for board delete");
      await deleteBoard(token, orgSlug, spaceSlug, entitySlug);
      return;
  }
}

/** Client component wiring EntitySettings to mutation functions with auth + router. */
export function EntitySettingsView({
  entityType,
  entitySlug,
  orgSlug,
  spaceSlug,
  currentValues,
  deleteRedirect,
  cancelHref,
}: EntitySettingsViewProps): React.ReactNode {
  const router = useRouter();
  const { getToken } = useAuth();
  const [loading, setLoading] = useState(false);

  const handleSave = async (values: EntityFormValues): Promise<void> => {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      await dispatchSave(entityType, token, entitySlug, values, orgSlug, spaceSlug);
      router.refresh();
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (): Promise<void> => {
    if (!window.confirm(`Are you sure you want to delete this ${entityType}?`)) return;
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) throw new Error("Unauthenticated");
      await dispatchDelete(entityType, token, entitySlug, orgSlug, spaceSlug);
      router.push(deleteRedirect);
    } finally {
      setLoading(false);
    }
  };

  return (
    <EntitySettings
      entityType={entityType}
      currentValues={currentValues}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={() => router.push(cancelHref)}
      loading={loading}
    />
  );
}
