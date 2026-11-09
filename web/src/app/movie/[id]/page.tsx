import { Suspense } from "react";
import { MovieDetail } from "@/components/movie-detail";
import { Skeleton } from "@/components/ui/skeleton";

export default function MoviePage() {
  return (
    <Suspense fallback={<Skeleton className="h-96 w-full" />}>
      <MovieDetail />
    </Suspense>
  );
}
