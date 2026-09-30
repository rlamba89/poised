"use client";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export type Category = { id: string; name: string; noteOnly: boolean };

let cache: Promise<Category[]> | null = null;

/** Note and code categories, loaded once per page. */
export function useCategories(): Category[] {
  const [categories, setCategories] = useState<Category[]>([]);
  useEffect(() => {
    cache ??= api<Category[]>("/categories");
    cache.then(setCategories).catch(() => {
      cache = null;
    });
  }, []);
  return categories;
}
