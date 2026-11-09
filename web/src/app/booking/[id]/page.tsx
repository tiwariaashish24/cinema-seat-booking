import { Suspense } from "react";
import { BookingView } from "@/components/booking-view";
import { Skeleton } from "@/components/ui/skeleton";

export default function BookingPage() {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <BookingView />
    </Suspense>
  );
}
