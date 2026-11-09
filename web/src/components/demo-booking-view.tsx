"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { api, type DemoReservation } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function DemoBookingView() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [reservation, setReservation] = useState<DemoReservation | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.getDemoBooking(id)
      .then(setReservation)
      .catch((requestError) => setError(String(requestError)));
  }, [id]);

  async function cancelReservation() {
    setBusy(true);
    setError("");
    try {
      await api.cancelDemoBooking(id);
      router.push("/my-bookings");
    } catch (requestError) {
      setError(String(requestError));
      setBusy(false);
    }
  }

  if (error && !reservation) {
    return (
      <Card className="mx-auto max-w-lg">
        <CardContent className="space-y-4 pt-6 text-center">
          <p role="alert" className="text-destructive">{error}</p>
          <Button variant="outline" nativeButton={false} render={<Link href="/my-bookings" />}>
            My Bookings
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (!reservation) {
    return <Card className="mx-auto max-w-lg"><CardContent className="pt-6">Loading demo booking…</CardContent></Card>;
  }

  return (
    <Card className="mx-auto max-w-lg">
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle>Demo booking details</CardTitle>
        <Badge variant="outline">DEMO ONLY</Badge>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          This reservation is stored in the local demo database. It is not a ticket for a real cinema.
        </p>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
          <dt className="text-muted-foreground">Movie</dt><dd>{reservation.movie_title}</dd>
          <dt className="text-muted-foreground">Cinema</dt><dd>{reservation.cinema}</dd>
          <dt className="text-muted-foreground">Screen</dt><dd>{reservation.screen}</dd>
          <dt className="text-muted-foreground">Showtime</dt>
          <dd>{new Date(reservation.starts_at).toLocaleString("en-IN", {
            dateStyle: "medium",
            timeStyle: "short",
          })}</dd>
          <dt className="text-muted-foreground">Seat</dt><dd>{reservation.seat}</dd>
          <dt className="text-muted-foreground">Customer</dt><dd>{reservation.customer}</dd>
          <dt className="text-muted-foreground">Status</dt><dd>{reservation.status}</dd>
          <dt className="text-muted-foreground">Booking ID</dt><dd className="break-all font-mono text-xs">{reservation.id}</dd>
        </dl>
        {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
        <div className="flex flex-col gap-2 sm:flex-row">
          {reservation.status === "confirmed" && (
            <Button variant="destructive" onClick={cancelReservation} disabled={busy}>
              {busy ? "Cancelling…" : "Cancel demo reservation"}
            </Button>
          )}
          <Button variant="outline" nativeButton={false} render={<Link href="/my-bookings" />}>
            My Bookings
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
