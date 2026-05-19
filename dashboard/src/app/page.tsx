"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getKey } from "@/lib/api";

export default function Home() {
  const router = useRouter();
  useEffect(() => {
    router.replace(getKey() ? "/sessions" : "/login");
  }, [router]);
  return null;
}
