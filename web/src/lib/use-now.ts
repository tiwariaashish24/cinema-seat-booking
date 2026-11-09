"use client";

import { useSyncExternalStore } from "react";

export function useNow(stepMs = 1000): number {
  return useSyncExternalStore(
    (onStoreChange) => {
      const id = setInterval(onStoreChange, stepMs);
      return () => clearInterval(id);
    },
    () => Date.now(),
    () => 0,
  );
}
