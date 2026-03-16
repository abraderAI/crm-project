"use client";

import type { ReactNode } from "react";
import { WIDGET_IDS } from "@/lib/default-layouts";
import type { WidgetConfig } from "@/lib/tier-types";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import { DocsHighlightsWidget } from "./widgets/docs-highlights-widget";
import { ForumHighlightsWidget } from "./widgets/forum-highlights-widget";
import { GetStartedWidget } from "./widgets/get-started-widget";

/** Widget registry for Tier 1 home screen. */
const TIER_1_REGISTRY: WidgetRegistry = {
  [WIDGET_IDS.DOCS_HIGHLIGHTS]: {
    title: "Documentation",
    render: () => <DocsHighlightsWidget />,
  },
  [WIDGET_IDS.FORUM_HIGHLIGHTS]: {
    title: "Community Forum",
    render: () => <ForumHighlightsWidget />,
  },
  [WIDGET_IDS.GET_STARTED]: {
    title: "Get Started",
    render: () => <GetStartedWidget />,
  },
};

interface Tier1HomeProps {
  /** Widget layout config (from useHomeLayout or default). */
  layout: WidgetConfig[];
}

/**
 * Tier 1 (Anonymous) home screen.
 * Renders docs highlights, forum highlights, and a get-started CTA.
 */
export function Tier1Home({ layout }: Tier1HomeProps): ReactNode {
  return (
    <div data-testid="tier-1-home">
      <h2 className="mb-4 text-lg font-semibold text-foreground">Welcome to DEFT Evolution</h2>
      <HomeLayout layout={layout} registry={TIER_1_REGISTRY} />
    </div>
  );
}
