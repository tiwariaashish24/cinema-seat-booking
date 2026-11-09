"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import {
  api,
  posterUrl,
  type DemoSeat,
  type DemoShowtime,
  type NearbyCinemaResult,
} from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const PRICE_FORMAT = new Intl.NumberFormat("en-IN", {
  style: "currency",
  currency: "INR",
  maximumFractionDigits: 0,
});

export function MovieDetail() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const movieId = params.id;
  const title = searchParams.get("title") || "Movie details";
  const overview = searchParams.get("overview") ?? "";
  const poster = posterUrl(searchParams.get("poster"));
  const releaseDate = searchParams.get("release_date") ?? "";
  const rating = Number(searchParams.get("rating") ?? 0);

  const [location, setLocation] = useState("Delhi NCR");
  const [nearby, setNearby] = useState<NearbyCinemaResult | null>(null);
  const [nearbyLoading, setNearbyLoading] = useState(false);
  const [nearbyError, setNearbyError] = useState("");
  const [showtimes, setShowtimes] = useState<DemoShowtime[]>([]);
  const [showtimesLoading, setShowtimesLoading] = useState(true);
  const [showtimesError, setShowtimesError] = useState("");
  const [selectedShowtime, setSelectedShowtime] = useState<DemoShowtime | null>(null);
  const [seats, setSeats] = useState<DemoSeat[]>([]);
  const [seatsLoading, setSeatsLoading] = useState(false);
  const [selectedSeat, setSelectedSeat] = useState<DemoSeat | null>(null);
  const [customer, setCustomer] = useState("");
  const [bookingBusy, setBookingBusy] = useState(false);
  const [bookingError, setBookingError] = useState("");

  useEffect(() => {
    const timer = window.setTimeout(
      () => setCustomer(localStorage.getItem("cinema_user_id") ?? ""),
      0,
    );
    return () => window.clearTimeout(timer);
  }, []);

  useEffect(() => {
    api.demoShowtimes()
      .then(setShowtimes)
      .catch((error) => setShowtimesError(String(error)))
      .finally(() => setShowtimesLoading(false));
    api.nearbyCinemas("Delhi NCR")
      .then(setNearby)
      .catch((error) => setNearbyError(String(error)));
  }, []);

  async function searchNearby(event?: FormEvent) {
    event?.preventDefault();
    setNearbyLoading(true);
    setNearbyError("");
    try {
      setNearby(await api.nearbyCinemas(location.trim()));
    } catch (error) {
      setNearbyError(String(error));
    } finally {
      setNearbyLoading(false);
    }
  }

  function searchMyLocation() {
    if (!navigator.geolocation) {
      setNearbyError("This browser does not support location access. Search by area instead.");
      return;
    }
    setNearbyLoading(true);
    setNearbyError("");
    navigator.geolocation.getCurrentPosition(
      async ({ coords }) => {
        try {
          setNearby(await api.nearbyCinemasAt(coords.latitude, coords.longitude));
        } catch (error) {
          setNearbyError(String(error));
        } finally {
          setNearbyLoading(false);
        }
      },
      (error) => {
        setNearbyError(`Location access failed: ${error.message}. Search by area instead.`);
        setNearbyLoading(false);
      },
      { enableHighAccuracy: false, timeout: 10000, maximumAge: 300000 },
    );
  }

  async function chooseShowtime(showtime: DemoShowtime) {
    setSelectedShowtime(showtime);
    setSelectedSeat(null);
    setSeats([]);
    setSeatsLoading(true);
    setBookingError("");
    try {
      setSeats(await api.demoSeats(showtime.id));
    } catch (error) {
      setBookingError(String(error));
    } finally {
      setSeatsLoading(false);
    }
  }

  async function reserveSeat() {
    if (!selectedShowtime || !selectedSeat || !customer.trim()) return;
    setBookingBusy(true);
    setBookingError("");
    try {
      const reservation = await api.createDemoBooking({
        showtime_id: selectedShowtime.id,
        seat_id: selectedSeat.id,
        movie_id: movieId,
        movie_title: title,
        customer: customer.trim(),
      });
      localStorage.setItem("cinema_user_id", customer.trim());
      router.push(`/demo-booking/${reservation.id}`);
    } catch (error) {
      setBookingError(String(error));
    } finally {
      setBookingBusy(false);
    }
  }

  const availableSeats = useMemo(
    () => seats.filter((seat) => seat.status === "available").length,
    [seats],
  );

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-5 sm:flex-row">
        {poster && (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={poster}
            alt={title}
            className="h-56 w-40 rounded-lg object-cover"
          />
        )}
        <div className="space-y-3">
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">TMDB movie</Badge>
            {releaseDate && <Badge variant="outline">{releaseDate.slice(0, 4)}</Badge>}
            {rating > 0 && <Badge variant="outline">★ {rating.toFixed(1)}</Badge>}
          </div>
          <h1 className="text-3xl font-bold">{title}</h1>
          {overview && <p className="max-w-3xl text-sm text-muted-foreground">{overview}</p>}
          <Link href="/" className="text-sm text-primary underline">Back to movies</Link>
        </div>
      </div>

      <section className="space-y-4">
        <div>
          <h2 className="text-xl font-semibold">Nearby real cinemas</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Locations are from OpenStreetMap. Real showtimes and ticket availability are not
            provided by this free map data.
          </p>
        </div>
        <form onSubmit={searchNearby} className="flex flex-col gap-2 sm:flex-row">
          <Input
            aria-label="Area or city in India"
            value={location}
            onChange={(event) => setLocation(event.target.value)}
            placeholder="Search a Delhi NCR area"
            maxLength={160}
          />
          <Button type="submit" disabled={nearbyLoading || !location.trim()}>
            {nearbyLoading ? "Searching…" : "Search area"}
          </Button>
          <Button type="button" variant="outline" onClick={searchMyLocation} disabled={nearbyLoading}>
            Use my location
          </Button>
        </form>
        {nearbyError && <p role="alert" className="text-sm text-destructive">{nearbyError}</p>}
        {nearbyLoading && <p className="text-sm text-muted-foreground">Finding nearby cinemas…</p>}
        {nearby && !nearbyLoading && (
          <>
            <p className="text-xs text-muted-foreground">
              Around {nearby.location} · {nearby.source}
            </p>
            {nearby.cinemas.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                OpenStreetMap has no cinema listings near this location. Try another area;
                map listings may be incomplete.
              </p>
            ) : (
              <div className="grid gap-3 sm:grid-cols-2">
                {nearby.cinemas.map((cinema) => (
                  <Card key={cinema.id}>
                    <CardContent className="flex items-start justify-between gap-3 pt-5">
                      <div className="space-y-1">
                        <h3 className="font-medium">{cinema.name}</h3>
                        {cinema.address && (
                          <p className="text-sm text-muted-foreground">{cinema.address}</p>
                        )}
                        <p className="text-xs text-muted-foreground">
                          {cinema.distance_km.toFixed(1)} km away · real map listing only
                        </p>
                      </div>
                      <a
                        className="shrink-0 text-sm text-primary underline"
                        href={cinema.map_url}
                        target="_blank"
                        rel="noreferrer"
                      >
                        Map
                      </a>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
            <p className="text-xs text-muted-foreground">{nearby.attribution}</p>
          </>
        )}
      </section>

      <section className="space-y-4">
        <div className="rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
          <h2 className="font-semibold">Demo booking — local sample inventory</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            The cinema, showtimes, prices, and seats below are sample data stored in your local
            PostgreSQL database. They are not real listings or availability and do not reserve
            seats at any of the nearby cinemas.
          </p>
        </div>

        {showtimesError && <p role="alert" className="text-sm text-destructive">{showtimesError}</p>}
        {showtimesLoading && <p className="text-sm text-muted-foreground">Loading sample showtimes…</p>}
        {!showtimesLoading && !showtimesError && showtimes.length === 0 && (
          <p className="text-sm text-muted-foreground">No sample showtimes are available.</p>
        )}
        <div className="grid gap-3 sm:grid-cols-2">
          {showtimes.map((showtime) => (
            <button
              key={showtime.id}
              type="button"
              onClick={() => chooseShowtime(showtime)}
              className={`rounded-lg text-left ${selectedShowtime?.id === showtime.id ? "ring-2 ring-primary" : ""}`}
            >
              <Card className="h-full transition-colors hover:bg-accent/40">
                <CardContent className="space-y-2 pt-5">
                  <div className="flex items-center justify-between gap-2">
                    <h3 className="font-semibold">{showtime.cinema}</h3>
                    <Badge variant="outline">DEMO</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground">{showtime.address}</p>
                  <p className="text-sm">
                    {new Date(showtime.starts_at).toLocaleString("en-IN", {
                      dateStyle: "medium",
                      timeStyle: "short",
                    })}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {showtime.screen} · {showtime.language} · {showtime.format}
                  </p>
                  <p className="text-sm">
                    {showtime.available_seats} sample seats · {PRICE_FORMAT.format(showtime.price_paise / 100)}
                  </p>
                </CardContent>
              </Card>
            </button>
          ))}
        </div>
      </section>

      {selectedShowtime && (
        <section className="space-y-4">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 className="text-xl font-semibold">Choose a sample seat</h2>
              <p className="text-sm text-muted-foreground">
                {selectedShowtime.screen} · {availableSeats} available in the demo database
              </p>
            </div>
            <Badge variant="outline">Demo seat map</Badge>
          </div>
          {seatsLoading && <p className="text-sm text-muted-foreground">Loading sample seats…</p>}
          {!seatsLoading && (
            <Card>
              <CardContent className="space-y-4 pt-6">
                <div className="mx-auto h-2 w-2/3 rounded-full bg-primary/40" />
                <p className="text-center text-xs text-muted-foreground">SCREEN</p>
                <div className="grid grid-cols-5 gap-2 sm:grid-cols-10">
                  {seats.map((seat) => (
                    <button
                      key={seat.id}
                      type="button"
                      disabled={seat.status !== "available"}
                      aria-pressed={selectedSeat?.id === seat.id}
                      onClick={() => setSelectedSeat(seat)}
                      className={`rounded-md border px-2 py-2 text-xs ${
                        seat.status !== "available"
                          ? "cursor-not-allowed bg-muted text-muted-foreground line-through"
                          : selectedSeat?.id === seat.id
                            ? "border-primary bg-primary text-primary-foreground"
                            : "hover:border-primary"
                      }`}
                      title={`${seat.label} · ${seat.status}`}
                    >
                      {seat.label}
                    </button>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
          <Card className="max-w-xl">
            <CardHeader>
              <CardTitle className="text-base">Reserve a demo seat</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="space-y-1">
                <Label htmlFor="demo-customer">Name or user ID</Label>
                <Input
                  id="demo-customer"
                  value={customer}
                  onChange={(event) => setCustomer(event.target.value)}
                  maxLength={100}
                  placeholder="Your name"
                />
              </div>
              {bookingError && <p role="alert" className="text-sm text-destructive">{bookingError}</p>}
              <Button
                className="w-full"
                onClick={reserveSeat}
                disabled={!selectedSeat || !customer.trim() || bookingBusy || seatsLoading}
              >
                {bookingBusy ? "Reserving demo seat…" : "Reserve demo seat"}
              </Button>
              <p className="text-xs text-muted-foreground">
                Demo reservations are saved locally in PostgreSQL. View yours on{" "}
                <Link href="/my-bookings" className="underline">My Bookings</Link>.
              </p>
            </CardContent>
          </Card>
        </section>
      )}
    </div>
  );
}
