"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Booking } from "@/lib/api";
import { useNow } from "@/lib/use-now";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export default function MyBookingsPage() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [loaded, setLoaded] = useState(false);
  const now = useNow();

  useEffect(() => {
    const ids = JSON.parse(localStorage.getItem("cinema_bookings") ?? "[]") as string[];
    Promise.allSettled(ids.map((id) => api.getBooking(id)))
      .then((rs) =>
        setBookings(
          rs.flatMap((r) => (r.status === "fulfilled" ? [r.value] : [])),
        ),
      )
      .finally(() => setLoaded(true));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">My bookings</h1>
      {!loaded ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : bookings.length === 0 ? (
        <Card>
          <CardContent className="pt-6 text-center text-sm text-muted-foreground">
            No bookings yet.{" "}
            <Link href="/" className="underline">Browse movies</Link> to book a seat.
          </CardContent>
        </Card>
      ) : (
        bookings.map((b) => {
          const expired =
            b.status === "held" && new Date(b.expires_at).getTime() <= now;
          return (
            <Card key={b.id}>
              <CardContent className="flex items-center justify-between pt-4">
                <div className="space-y-1 text-sm">
                  <div className="font-medium">
                    Seat {b.seat_id} · Movie {b.movie_id}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {b.status === "held" && !expired
                      ? `Held until ${new Date(b.expires_at).toLocaleTimeString()}`
                      : b.status}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Badge
                    variant={
                      b.status === "confirmed" ? "default" : expired ? "destructive" : "secondary"
                    }
                  >
                    {expired ? "expired" : b.status}
                  </Badge>
                  <Button
                    size="sm"
                    variant="outline"
                    render={<Link href={`/booking/${b.id}`} />}
                  >
                    Open
                  </Button>
                </div>
              </CardContent>
            </Card>
          );
        })
      )}
    </div>
  );
}
