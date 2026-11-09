"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, type Booking, type DemoReservation } from "@/lib/api";
import { useNow } from "@/lib/use-now";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export default function MyBookingsPage() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [demoBookings, setDemoBookings] = useState<DemoReservation[]>([]);
  const [demoError, setDemoError] = useState("");
  const [loaded, setLoaded] = useState(false);
  const now = useNow();

  useEffect(() => {
    const ids = JSON.parse(localStorage.getItem("cinema_bookings") ?? "[]") as string[];
    const customer = localStorage.getItem("cinema_user_id")?.trim() ?? "";
    const legacyBookings = Promise.allSettled(ids.map((id) => api.getBooking(id))).then((results) =>
      setBookings(results.flatMap((result) => result.status === "fulfilled" ? [result.value] : [])),
    );
    const localBookings = customer
      ? api.listDemoBookings(customer)
          .then(setDemoBookings)
          .catch((error) => setDemoError(String(error)))
      : Promise.resolve();
    Promise.all([legacyBookings, localBookings]).finally(() => setLoaded(true));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold">My bookings</h1>
      {!loaded ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : bookings.length === 0 && demoBookings.length === 0 ? (
        <Card>
          <CardContent className="pt-6 text-center text-sm text-muted-foreground">
            {demoError && <p role="alert" className="mb-2 text-destructive">{demoError}</p>}
            No bookings found for this browser user.{" "}
            <Link href="/" className="underline">Browse movies</Link> to book a seat.
          </CardContent>
        </Card>
      ) : (
        <>
          {demoError && <p role="alert" className="text-sm text-destructive">{demoError}</p>}
          {demoBookings.map((booking) => (
            <Card key={booking.id}>
              <CardContent className="flex flex-col justify-between gap-3 pt-4 sm:flex-row sm:items-center">
                <div className="space-y-1 text-sm">
                  <div className="flex flex-wrap items-center gap-2 font-medium">
                    <span>{booking.movie_title} · Seat {booking.seat}</span>
                    <Badge variant="outline">DEMO</Badge>
                  </div>
                  <div>{booking.cinema} · {booking.screen}</div>
                  <div className="text-xs text-muted-foreground">
                    {new Date(booking.starts_at).toLocaleString("en-IN", {
                      dateStyle: "medium",
                      timeStyle: "short",
                    })}
                  </div>
                  <div className="text-xs text-muted-foreground">{booking.status}</div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  nativeButton={false}
                  render={<Link href={`/demo-booking/${booking.id}`} />}
                >
                  Open demo booking
                </Button>
              </CardContent>
            </Card>
          ))}
          {bookings.map((b) => {
            const expired = b.status === "held" && new Date(b.expires_at).getTime() <= now;
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
                    nativeButton={false}
                    render={<Link href={`/booking/${b.id}`} />}
                  >
                    Open
                  </Button>
                </div>
              </CardContent>
            </Card>
            );
          })}
        </>
      )}
    </div>
  );
}
