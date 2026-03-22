/// <reference types="vitest/config" />
import react from "@vitejs/plugin-react";
import path from "path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    coverage: {
      provider: "v8",
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/**/*.test.{ts,tsx}",
        "src/**/*.d.ts",
        "src/app/layout.tsx",
        "src/app/page.tsx",
        "src/middleware.ts",
        "src/app/**/sign-in/**",
        "src/app/**/sign-up/**",
        "src/app/admin/**",
        "src/app/orgs/**",
        "src/app/crm/**",
        "src/app/reports/**",
        "src/app/notifications/**",
        "src/app/upgrade/**",
        "src/app/search/**",
        "src/app/settings/**",
        "src/components/thread/thread-create-view.tsx",
        "src/components/entities/entity-create-view.tsx",
        "src/components/entities/entity-settings-view.tsx",
        "src/lib/api-types.ts",
        "src/lib/api-types-admin.ts",
        "src/lib/api-types-channel.ts",
        "src/lib/reporting-types.ts",
        "src/lib/tier-types.ts",
        "src/lib/widget-api.ts",
        "src/components/home/widgets/billing-overview-widget.tsx",
        "src/components/home/widgets/conversion-metrics-widget.tsx",
        "src/components/home/widgets/lead-pipeline-widget.tsx",
        "src/components/home/widgets/recent-audit-log-widget.tsx",
        "src/components/home/widgets/recent-leads-widget.tsx",
        "src/components/home/widgets/system-health-widget.tsx",
        "src/components/home/widgets/ticket-queue-widget.tsx",
        "src/components/home/widgets/ticket-stats-widget.tsx",
        "src/app/(public)/**",
        "src/app/forum/**",
        "src/app/docs/**",
        "src/app/support/**",
        "e2e/**",
      ],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85,
      },
    },
  },
});
