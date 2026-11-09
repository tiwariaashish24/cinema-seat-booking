"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, posterUrl, type SampleMovie, type TmdbMovie } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

type Tab = "showing" | "popular" | "trending" | "upcoming";
const PAGE_SIZE = 20;
const MAX_PAGE = 500;

const TABS: { key: Tab; label: string }[] = [
  { key: "showing", label: "Now Showing" },
  { key: "popular", label: "Popular" },
  { key: "trending", label: "Trending" },
  { key: "upcoming", label: "Upcoming" },
];

function sampleId(m: SampleMovie) {
  return m.ID ?? m.id ?? "";
}
function sampleTitle(m: SampleMovie) {
  return m.Title ?? m.title ?? "Untitled";
}
function sampleDesc(m: SampleMovie) {
  return m.Description ?? m.description ?? "";
}
function sampleSeats(m: SampleMovie) {
  return m.Seats ?? m.seats ?? 0;
}

export default function HomePage() {
  const [tab, setTab] = useState<Tab>("popular");
  const [query, setQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [results, setResults] = useState<TmdbMovie[] | null>(null);
  const [samples, setSamples] = useState<SampleMovie[]>([]);
  const [samplesLoaded, setSamplesLoaded] = useState(false);
  const [tmdb, setTmdb] = useState<Record<Tab, TmdbMovie[]>>({
    showing: [],
    popular: [],
    trending: [],
    upcoming: [],
  });
  const [pages, setPages] = useState<Record<Tab, number>>({
    showing: 1,
    popular: 1,
    trending: 1,
    upcoming: 1,
  });
  const [hasMore, setHasMore] = useState<Record<Tab, boolean>>({
    showing: false,
    popular: false,
    trending: false,
    upcoming: false,
  });
  const [loaded, setLoaded] = useState<Record<Tab, boolean>>({
    showing: false,
    popular: false,
    trending: false,
    upcoming: false,
  });
  const [loadingMore, setLoadingMore] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [released, setReleased] = useState<number | null>(null);

  useEffect(() => {
    api
      .listMovies()
      .then(setSamples)
      .catch((e) => setErrors((p) => ({ ...p, showing: String(e) })))
      .finally(() => setSamplesLoaded(true));
    (Object.keys(tmdb) as Tab[])
      .filter((t) => t !== "showing")
      .forEach((t) => {
        const fn = { popular: api.popular, trending: api.trending, upcoming: api.upcoming }[t];
        fn(1)
          .then((ms) => {
            setTmdb((p) => ({ ...p, [t]: ms }));
            setHasMore((p) => ({ ...p, [t]: ms.length === PAGE_SIZE }));
            setLoaded((p) => ({ ...p, [t]: true }));
          })
          .catch((e) => {
            setErrors((p) => ({ ...p, [t]: String(e) }));
            setLoaded((p) => ({ ...p, [t]: true }));
          });
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function loadMoreMovies() {
    if (tab === "showing" || loadingMore || pages[tab] >= MAX_PAGE) return;

    setLoadingMore(true);
    const nextPage = pages[tab] + 1;
    try {
      const fn = { popular: api.popular, trending: api.trending, upcoming: api.upcoming }[tab];
      const movies = await fn(nextPage);
      setTmdb((p) => ({ ...p, [tab]: [...p[tab], ...movies] }));
      setPages((p) => ({ ...p, [tab]: nextPage }));
      setHasMore((p) => ({ ...p, [tab]: movies.length === PAGE_SIZE && nextPage < MAX_PAGE }));
      setErrors((p) => {
        const next = { ...p };
        delete next[tab];
        return next;
      });
    } catch (e) {
      setErrors((p) => ({ ...p, [tab]: String(e) }));
    } finally {
      setLoadingMore(false);
    }
  }

  async function runSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) {
      setResults(null);
      return;
    }
    setSearching(true);
    try {
      setResults(await api.search(query.trim()));
    } catch (err) {
      setErrors((p) => ({ ...p, search: String(err) }));
    } finally {
      setSearching(false);
    }
  }

  async function releaseExpired() {
    try {
      const r = await api.releaseExpired();
      setReleased(r.released);
    } catch (e) {
      setErrors((p) => ({ ...p, admin: String(e) }));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <form onSubmit={runSearch} className="flex w-full max-w-md gap-2">
          <Input
            placeholder="Search movies…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <Button type="submit" disabled={searching}>
            {searching ? "Searching…" : "Search"}
          </Button>
        </form>
        <Button variant="outline" size="sm" onClick={releaseExpired}>
          Release expired holds{released !== null ? ` (${released})` : ""}
        </Button>
      </div>

      {results ? (
        <section>
          <h2 className="mb-3 text-lg font-semibold">Search results</h2>
          {results.length === 0 ? (
            <p className="text-sm text-muted-foreground">No movies found.</p>
          ) : (
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {results.map((m) => (
                <TmdbCard key={m.id} movie={m} />
              ))}
            </div>
          )}
          <Button variant="ghost" size="sm" className="mt-3" onClick={() => setResults(null)}>
            Clear search
          </Button>
        </section>
      ) : (
        <>
          <div className="flex gap-2">
            {TABS.map((t) => (
              <Button
                key={t.key}
                variant={tab === t.key ? "default" : "outline"}
                size="sm"
                onClick={() => setTab(t.key)}
              >
                {t.label}
              </Button>
            ))}
          </div>

          {errors[tab] && (
            <p className="text-sm text-destructive">
              Could not load {tab} movies: {errors[tab]}
            </p>
          )}

          {tab === "showing" ? (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {!samplesLoaded &&
                [0, 1, 2].map((i) => <Skeleton key={i} className="h-36 w-full" />)}
              {samplesLoaded && samples.length === 0 && !errors.showing && (
                <p className="text-sm text-muted-foreground">No movies are currently showing.</p>
              )}
              {samples.map((m) => (
                <Card key={sampleId(m)} className="overflow-hidden">
                  <div className="bg-primary/10 px-4 pt-4">
                    <div className="flex h-24 items-center justify-center rounded-md bg-gradient-to-br from-primary/30 to-primary/5 text-2xl font-bold text-primary">
                      {sampleTitle(m).charAt(0)}
                    </div>
                  </div>
                  <CardContent className="space-y-2 pt-4">
                    <div className="flex items-center justify-between">
                      <h3 className="font-semibold">{sampleTitle(m)}</h3>
                      <Badge variant="secondary">{sampleSeats(m)} seats</Badge>
                    </div>
                    <p className="line-clamp-2 text-sm text-muted-foreground">
                      {sampleDesc(m)}
                    </p>
                    <Button
                      size="sm"
                      className="w-full"
                      render={<Link href={`/movie/${sampleId(m)}`} />}
                    >
                      Select seats
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <>
              {loaded[tab] && tmdb[tab].length === 0 && !errors[tab] && (
                <p className="text-sm text-muted-foreground">No movies found.</p>
              )}
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
              {!loaded[tab] && !errors[tab] &&
                [0, 1, 2, 3, 4, 5, 6, 7].map((i) => (
                  <Skeleton key={i} className="h-72 w-full" />
                ))}
              {tmdb[tab].map((m) => (
                <TmdbCard key={m.id} movie={m} />
              ))}
              </div>
              {hasMore[tab] && (
                <div className="mt-6 flex justify-center">
                  <Button variant="outline" onClick={loadMoreMovies} disabled={loadingMore}>
                    {loadingMore ? "Loading…" : "Load more movies"}
                  </Button>
                </div>
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}

function TmdbCard({ movie }: { movie: TmdbMovie }) {
  const img = posterUrl(movie.poster_path);
  return (
    <Link href={`/movie/tmdb-${movie.id}?title=${encodeURIComponent(movie.title)}&poster=${encodeURIComponent(movie.poster_path ?? "")}`}>
      <Card className="h-full overflow-hidden transition-shadow hover:shadow-md">
        {img ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={img} alt={movie.title} className="aspect-[2/3] w-full object-cover" />
        ) : (
          <div className="flex aspect-[2/3] items-center justify-center bg-muted text-3xl font-bold text-muted-foreground">
            {movie.title.charAt(0)}
          </div>
        )}
        <CardContent className="space-y-1 pt-3">
          <h3 className="line-clamp-1 text-sm font-semibold">{movie.title}</h3>
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>{movie.release_date?.slice(0, 4) || "—"}</span>
            <span>★ {movie.vote_average?.toFixed(1)}</span>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
