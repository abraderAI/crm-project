"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@clerk/nextjs";
import { TierHomeScreen } from "@/components/home/tier-home-screen";

export default function Home(): React.ReactNode {
  const { getToken } = useAuth();
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    getToken().then((t) => {
      if (active) setToken(t);
    });
    return () => {
      active = false;
    };
  }, [getToken]);

  return <TierHomeScreen token={token} />;
}
