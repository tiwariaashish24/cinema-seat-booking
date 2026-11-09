"use client";

import { useSyncExternalStore } from "react";

type Clock = {
  now: number;
  listeners: Set<() => void>;
  interval: ReturnType<typeof setInterval> | null;
};

const clocks = new Map<number, Clock>();

function getClock(stepMs: number): Clock {
  let clock = clocks.get(stepMs);
  if (!clock) {
    clock = { now: Date.now(), listeners: new Set(), interval: null };
    clocks.set(stepMs, clock);
  }
  return clock;
}

function subscribe(stepMs: number, listener: () => void) {
  const clock = getClock(stepMs);
  clock.listeners.add(listener);
  if (clock.interval === null) {
    clock.interval = setInterval(() => {
      clock.now = Date.now();
      clock.listeners.forEach((notify) => notify());
    }, stepMs);
  }

  return () => {
    clock.listeners.delete(listener);
    if (clock.listeners.size === 0 && clock.interval !== null) {
      clearInterval(clock.interval);
      clock.interval = null;
      clocks.delete(stepMs);
    }
  };
}

export function useNow(stepMs = 1000): number {
  return useSyncExternalStore(
    (onStoreChange) => subscribe(stepMs, onStoreChange),
    () => getClock(stepMs).now,
    () => 0,
  );
}
