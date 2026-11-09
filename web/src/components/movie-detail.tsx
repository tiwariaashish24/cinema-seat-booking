"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { api, posterUrl, type Booking, type SampleMovie } from "@/lib/api";
import { useNow } from "@/lib/use-now";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const COLS = 10;
const DEFAULT_TMDB_SEATS = 100;

function buildSeats(total: number) {
  const rows = Math.ceil(total / COLS);
  const seats: { id: string; row: number; col: number }[] = [];
  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < COLS; c++) {
      const n = r * COLS + c;
      if (n >= total) break;
      seats.push({
        id: `${String.fromCharCode(65 + r)}${c + 1}`,
        row: r,
        col: c,
      });
    }
  }
  return seats;
}

function isSeatTaken(b: Booking, now: number) {
  if (b.status === "confirmed") return true;
  if (b.status === "held") return new Date(b.expires_at).getTime() > now;
  return false;
}

export function MovieDetail() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const id = params.id;
  const isTmdb = id.startsWith("tmdb-");
  const movieId = id;

  const [sample, setSample] = useState<SampleMovie | null>(null);
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [userId, setUserId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const now = useNow();

  useEffect(() => {
    const t = setTimeout(
      () => setUserId(localStorage.getItem("cinema_user_id") ?? ""),
      0,
    );
    return () => clearTimeout(t);
  }, []);

  useEffect(() => {
    if (isTmdb) return;
    api
      .listMovies()
      .then((ms) =>
        setSample(ms.find((x) => (x.ID ?? x.id) === id) ?? null),
      )
      .catch(() => setSample(null));
  }, [id, isTmdb]);

  useEffect(() => {
    api
      .listBookings(movieId)
      .then(setBookings)
      .catch(() => setBookings([]));
  }, [movieId]);

  const title = isTmdb
    ? (searchParams.get("title") ?? "Movie")
    : sample
      ? ((sample.Title ?? sample.title) || "Movie")
      : "Loading…";
  const description = isTmdb
    ? ""
    : ((sample?.Description ?? sample?.description) || "");
  const poster = isTmdb ? posterUrl(searchParams.get("poster")) : null;
  const totalSeats = isTmdb
    ? DEFAULT_TMDB_SEATS
    : (sample?.Seats ?? sample?.seats ?? 100);

  const seats = useMemo(() => buildSeats(totalSeats), [totalSeats]);
  const taken = useMemo(() => {
    const s = new Set<string>();
    bookings.forEach((b) => {
      if (isSeatTaken(b, now)) s.add(b.seat_id);
    });
    return s;
  }, [bookings, now]);

  async function book() {
    if (!selected || !userId.trim()) return;
    setBusy(true);
    setError("");
    try {
      localStorage.setItem("cinema_user_id", userId.trim());
      const b = await api.createBooking(movieId, selected, userId.trim());
      const ids = JSON.parse(localStorage.getItem("cinema_bookings") ?? "[]") as string[];
      if (!ids.includes(b.id)) {
        ids.push(b.id);
        localStorage.setItem("cinema_bookings", JSON.stringify(ids));
      }
      router.push(`/booking/${b.id}`);
    } catch (e) {
      setError(String(e));
      setBusy(false);
    }
  }

  const rows = useMemo(() => {
    const map = new Map<number, typeof seats>();
    seats.forEach((s) => {
      const arr = map.get(s.row) ?? [];
      arr.push(s);
      map.set(s.row, arr);
    });
    return [...map.entries()].sort((a, b) => a[0] - b[0]);
  }, [seats]);

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
      <div className="space-y-4">
        <div className="flex gap-4">
          {poster && (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={poster} alt={title} className="hidden h-40 w-28 rounded-lg object-cover sm:block" />
          )}
          <div>
            <h1 className="text-2xl font-bold">{title}</h1>
            {description && (
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>
            )}
            <div className="mt-2 flex gap-2">
              <Badge variant="secondary">{totalSeats} seats</Badge>
              <Badge variant="outline">{taken.size} booked</Badge>
            </div>
          </div>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Select a seat</CardTitle>
          </CardHeader>
          <CardContent>
            {/* Screen */}
            <div className="mx-auto mb-6 h-3 w-2/3 rounded-full bg-gradient-to-r from-transparent via-primary/60 to-transparent" />
            <div className="space-y-2">
              {rows.map(([row, rowSeats]) => (
                <div key={row} className="flex items-center gap-2">
                  <span className="w-5 text-xs text-muted-foreground">
                    {String.fromCharCode(65 + row)}
                  </span>
                  <div className="flex flex-wrap gap-1.5">
                    {rowSeats.map((s) => {
                      const isTaken = taken.has(s.id);
                      const isSelected = selected === s.id;
                      return (
                        <button
                          key={s.id}
                          type="button"
                          disabled={isTaken}
                          onClick={() => setSelected(isSelected ? null : s.id)}
                          title={s.id}
                          className={`flex h-8 w-8 items-center justify-center rounded-md text-xs font-medium transition-colors ${
                            isTaken
                              ? "cursor-not-allowed bg-muted text-muted-foreground/40 line-through"
                              : isSelected
                                ? "bg-primary text-primary-foreground"
                                : "bg-secondary text-secondary-foreground hover:bg-accent"
                          }`}
                        >
                          {s.id.slice(1)}
                        </button>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
            <div className="mt-4 flex gap-4 text-xs text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-3 w-3 rounded bg-secondary" /> Available
              </span>
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-3 w-3 rounded bg-primary" /> Selected
              </span>
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-3 w-3 rounded bg-muted" /> Booked
              </span>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="h-fit">
        <CardHeader>
          <CardTitle className="text-base">Your booking</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="seat">Seat</Label>
            <Input id="seat" value={selected ?? ""} readOnly placeholder="Pick a seat" />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="uid">Name / User ID</Label>
            <Input
              id="uid"
              placeholder="e.g. ajay"
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
            />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <Button className="w-full" disabled={!selected || !userId.trim() || busy} onClick={book}>
            {busy ? "Holding seat…" : "Hold seat (2 min)"}
          </Button>
          <p className="text-xs text-muted-foreground">
            Seats are held for 2 minutes. You can find held bookings under{" "}
            <Link href="/my-bookings" className="underline">My Bookings</Link> if you
            enter the same user ID.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
