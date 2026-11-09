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

function friendlyNearbyError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  if (/502|503|504|temporarily unavailable|timed out|failed to fetch|lookup request failed/i.test(message)) {
    return "Nearby cinema data is temporarily unavailable because the free map service is busy. Please retry in a moment or search another area.";
  }
  return message;
}

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
  const fromTab = searchParams.get("from") ?? "";
  const detailsOnly = fromTab === "popular" || fromTab === "trending";

  const [location, setLocation] = useState("Delhi NCR");
  const [locationPreset, setLocationPreset] = useState("Delhi NCR");
  const [nearby, setNearby] = useState<NearbyCinemaResult | null>(null);
  const [nearbyLoading, setNearbyLoading] = useState(true);
  const [nearbyError, setNearbyError] = useState("");
  const [showtimes, setShowtimes] = useState<DemoShowtime[] | null>(null);
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
    if (detailsOnly) return;
    let active = true;
    api.demoShowtimes()
      .then((items) => {
        if (active) setShowtimes(items);
      })
      .catch((error) => {
        if (active) setShowtimesError(String(error));
      });
    api.nearbyCinemas("Delhi NCR")
      .then((result) => {
        if (active) setNearby(result);
      })
      .catch((error) => {
        if (active) setNearbyError(friendlyNearbyError(error));
      })
      .finally(() => {
        if (active) setNearbyLoading(false);
      });
    return () => {
      active = false;
    };
  }, [detailsOnly]);

  async function searchNearby(event?: FormEvent) {
    event?.preventDefault();
    setNearbyLoading(true);
    setNearbyError("");
    setNearby(null);
    try {
      setNearby(await api.nearbyCinemas(location.trim()));
    } catch (error) {
      setNearbyError(friendlyNearbyError(error));
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
    setNearby(null);
    navigator.geolocation.getCurrentPosition(
      async ({ coords }) => {
        try {
          setNearby(await api.nearbyCinemasAt(coords.latitude, coords.longitude));
        } catch (error) {
          setNearbyError(friendlyNearbyError(error));
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
  const totalPricePaise = selectedSeat?.price_paise ?? 0;

  return (
    <div className="space-y-8">
      <Card className="overflow-hidden">
        <CardContent className="flex flex-col gap-5 pt-5 sm:flex-row sm:items-center sm:gap-8">
          {poster ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={poster} alt={title} className="h-64 w-44 shrink-0 rounded-lg object-cover shadow-md" />
          ) : (
            <div className="flex h-64 w-44 shrink-0 items-center justify-center rounded-lg bg-muted text-4xl font-bold text-muted-foreground">
              {title.charAt(0)}
            </div>
          )}
          <div className="space-y-3">
            <div className="flex flex-wrap gap-2">
              <Badge variant="secondary">TMDB movie</Badge>
              {releaseDate && <Badge variant="outline">{releaseDate.slice(0, 4)}</Badge>}
              {rating > 0 && <Badge variant="outline">★ {rating.toFixed(1)}</Badge>}
            </div>
            <h1 className="text-3xl font-bold">{title}</h1>
            {overview && <p className="max-w-3xl text-sm leading-6 text-muted-foreground">{overview}</p>}
            <Link href="/" className="inline-block text-sm text-primary underline">Back to movies</Link>
          </div>
        </CardContent>
      </Card>

      {!detailsOnly && (
        <>
          <section className="space-y-4">
            <div>
              <h2 className="text-xl font-semibold">Nearby real cinemas</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Locations are from OpenStreetMap. Real showtimes and ticket availability are not
                provided by this free map data.
              </p>
            </div>
            <form onSubmit={searchNearby} className="grid gap-2 sm:grid-cols-[200px_1fr_auto_auto]">
              <select
                aria-label="Choose a nearby location"
                className="h-9 rounded-lg border border-input bg-background px-3 text-sm"
                value={locationPreset}
                onChange={(event) => {
                  const preset = event.target.value;
                  setLocationPreset(preset);
                  if (preset !== "Other area or PIN code") setLocation(preset);
                  else setLocation("");
                }}
              >
                <option>Delhi NCR</option>
                <option>New Delhi</option>
                <option>Noida</option>
                <option>Gurugram</option>
                <option>Faridabad</option>
                <option>Other area or PIN code</option>
              </select>
              <Input
                aria-label="Area or city in India"
                value={location}
                onChange={(event) => {
                  setLocation(event.target.value);
                  setLocationPreset("Other area or PIN code");
                }}
                placeholder="Search area, city or PIN code"
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
            {nearbyLoading && (
              <div className="grid gap-3 sm:grid-cols-2" aria-label="Loading nearby cinemas">
                {[0, 1, 2, 3].map((item) => (
                  <Card key={item}>
                    <CardContent className="space-y-2 pt-5">
                      <div className="h-4 w-2/3 animate-pulse rounded bg-muted" />
                      <div className="h-3 w-full animate-pulse rounded bg-muted" />
                      <div className="h-3 w-1/3 animate-pulse rounded bg-muted" />
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
            {nearby && !nearbyLoading && (
              nearby.cinemas.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No mapped cinemas were found near {nearby.location}. Try another area.
                </p>
              ) : (
                <>
                  <p className="text-xs text-muted-foreground">
                    Around {nearby.location} · {nearby.source}
                  </p>
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
                  <p className="text-xs text-muted-foreground">{nearby.attribution}</p>
                </>
              )
            )}
          </section>

          {showtimesError && <p role="alert" className="text-sm text-destructive">{showtimesError}</p>}
          {showtimes === null && !showtimesError && (
            <p className="text-sm text-muted-foreground">Loading sample showtimes…</p>
          )}
          {showtimes?.length === 0 && !showtimesError && (
            <p className="text-sm text-muted-foreground">No sample showtimes are available right now.</p>
          )}
          {!!showtimes?.length && (
            <section className="space-y-4">
              <div className="rounded-lg border border-amber-500/40 bg-amber-500/5 p-4">
                <h2 className="font-semibold">Demo booking — local sample inventory</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  The cinema, showtimes, prices, and seats below are sample data stored in your local
                  PostgreSQL database. They are not real listings or availability and do not reserve
                  seats at any of the nearby cinemas.
                </p>
              </div>
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
                        <p className="text-xs text-muted-foreground">Sample location · {showtime.address}</p>
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
          )}
        </>
      )}

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
          {!seatsLoading && seats.length === 0 && (
            <p className="text-sm text-muted-foreground">
              No seat inventory is available for this showtime.
            </p>
          )}
          {!seatsLoading && seats.length > 0 && (
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
                <div className="flex flex-wrap justify-center gap-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1.5"><span className="size-3 rounded border" /> Available</span>
                  <span className="flex items-center gap-1.5"><span className="size-3 rounded bg-primary" /> Selected</span>
                  <span className="flex items-center gap-1.5"><span className="size-3 rounded bg-muted" /> Booked</span>
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
              <div className="flex items-center justify-between rounded-md bg-muted/50 px-3 py-2 text-sm">
                <span>Demo total · {selectedSeat?.label ?? "select a seat"}</span>
                <span className="font-semibold">{PRICE_FORMAT.format(totalPricePaise / 100)}</span>
              </div>
              {bookingError && <p role="alert" className="text-sm text-destructive">{bookingError}</p>}
              <Button
                className="w-full"
                onClick={reserveSeat}
                disabled={!selectedSeat || !customer.trim() || bookingBusy || seatsLoading}
              >
                {bookingBusy ? "Reserving demo seat…" : `Reserve demo seat${selectedSeat ? ` · ${PRICE_FORMAT.format(totalPricePaise / 100)}` : ""}`}
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
