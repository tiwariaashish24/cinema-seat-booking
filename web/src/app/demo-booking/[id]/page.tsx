import { Suspense } from "react";
import { DemoBookingView } from "@/components/demo-booking-view";
import { Skeleton } from "@/components/ui/skeleton";

export default function DemoBookingPage() {
  return (
    <Suspense fallback={<Skeleton className="mx-auto h-72 max-w-lg" />}>
      <DemoBookingView />
    </Suspense>
  );
}
