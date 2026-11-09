"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { api, posterUrl, type TmdbMovie } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";

type Tab = "showing" | "popular" | "trending" | "upcoming";
const PAGE_SIZE = 20;
const MAX_PAGE = 500;

function uniqueMovies(movies: TmdbMovie[]) {
  const seen = new Set<number>();
  return movies.filter((movie) => {
    if (seen.has(movie.id)) return false;
    seen.add(movie.id);
    return true;
  });
}

function dateInIndia(timestamp: number) {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Kolkata",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(timestamp);
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((value) => value.type === type)?.value ?? "";
  return `${part("year")}-${part("month")}-${part("day")}`;
}

const TABS: { key: Tab; label: string }[] = [
  { key: "showing", label: "Now Showing" },
  { key: "popular", label: "Popular" },
  { key: "trending", label: "Trending" },
  { key: "upcoming", label: "Upcoming" },
];

export default function HomePage() {
  const [tab, setTab] = useState<Tab>("showing");
  const [query, setQuery] = useState("");
  const [searching, setSearching] = useState(false);
  const [results, setResults] = useState<TmdbMovie[] | null>(null);
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
  const [newReleases, setNewReleases] = useState<TmdbMovie[]>([]);
  const [newReleasesLoaded, setNewReleasesLoaded] = useState(false);
  const [today, setToday] = useState<string | null>(null);

  useEffect(() => {
    setToday(dateInIndia(Date.now()));
    const loaders: Record<Tab, (page: number) => Promise<TmdbMovie[]>> = {
      showing: api.nowPlaying,
      popular: api.popular,
      trending: api.trending,
      upcoming: api.upcoming,
    };
    (Object.keys(loaders) as Tab[]).forEach((currentTab) => {
      loaders[currentTab](1)
        .then((movies) => {
          setTmdb((previous) => ({ ...previous, [currentTab]: uniqueMovies(movies) }));
          setHasMore((previous) => ({ ...previous, [currentTab]: movies.length === PAGE_SIZE }));
        })
        .catch((error) =>
          setErrors((previous) => ({ ...previous, [currentTab]: String(error) })),
        )
        .finally(() =>
          setLoaded((previous) => ({ ...previous, [currentTab]: true })),
        );
    });
    api.newReleases(1)
      .then((movies) => setNewReleases(uniqueMovies(movies)))
      .catch((error) =>
        setErrors((previous) => ({ ...previous, newReleases: String(error) })),
      )
      .finally(() => setNewReleasesLoaded(true));
  }, []);

  const nowShowingMovies = today
    ? tmdb.showing.filter(
        (movie) => !!movie.release_date && movie.release_date <= today,
      )
    : [];
  const upcomingMovies = today
    ? uniqueMovies([...tmdb.showing, ...tmdb.upcoming]).filter(
        (movie) => !!movie.release_date && movie.release_date > today,
      )
    : [];

  async function loadMoreMovies() {
    if (loadingMore || pages[tab] >= MAX_PAGE) return;

    setLoadingMore(true);
    const nextPage = pages[tab] + 1;
    try {
      const fn = {
        showing: api.nowPlaying,
        popular: api.popular,
        trending: api.trending,
        upcoming: api.upcoming,
      }[tab];
      const movies = await fn(nextPage);
      setTmdb((p) => ({ ...p, [tab]: uniqueMovies([...p[tab], ...movies]) }));
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
      setResults(uniqueMovies(await api.search(query.trim())));
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

          <p className="text-xs text-muted-foreground">
            Movie listings from TMDB. Theatre listings and booking inventory are
            shown on each movie page.
          </p>
          {tab === "showing" ? (
            <div className="space-y-8">
              <section className="space-y-3">
                <h2 className="text-lg font-semibold">Now Playing in Cinemas</h2>
                <MovieGrid
                  movies={nowShowingMovies}
                  from="showing"
                  loading={!loaded.showing || !today}
                  error={errors.showing}
                  hasMore={hasMore.showing}
                  loadingMore={loadingMore}
                  onLoadMore={loadMoreMovies}
                />
              </section>
              <section className="space-y-3">
                <h2 className="text-lg font-semibold">New Releases</h2>
                <MovieGrid
                  movies={newReleases}
                  from="new-releases"
                  loading={!newReleasesLoaded}
                  error={errors.newReleases}
                  hasMore={false}
                  loadingMore={false}
                  onLoadMore={() => {}}
                />
              </section>
            </div>
          ) : (
            <MovieGrid
              movies={tab === "upcoming" ? upcomingMovies : tmdb[tab]}
              from={tab}
              comingSoon={tab === "upcoming"}
              loading={!loaded[tab] || (tab === "upcoming" && !today)}
              error={errors[tab]}
              hasMore={hasMore[tab]}
              loadingMore={loadingMore}
              onLoadMore={loadMoreMovies}
            />
          )}
        </>
      )}
    </div>
  );
}

function MovieGrid({
  movies,
  from,
  comingSoon = false,
  loading,
  error,
  hasMore,
  loadingMore,
  onLoadMore,
}: {
  movies: TmdbMovie[];
  from: string;
  comingSoon?: boolean;
  loading: boolean;
  error?: string;
  hasMore: boolean;
  loadingMore: boolean;
  onLoadMore: () => void;
}) {
  return (
    <div>
      {error && (
        <p className="mb-3 text-sm text-destructive">
          Could not load movies: {error}
        </p>
      )}
      {loading && !error && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {[0, 1, 2, 3, 4, 5, 6, 7].map((i) => (
            <Skeleton key={i} className="h-72 w-full" />
          ))}
        </div>
      )}
      {!loading && movies.length === 0 && !error && (
        <p className="text-sm text-muted-foreground">No movies found.</p>
      )}
      {movies.length > 0 && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {movies.map((movie) => (
            <TmdbCard key={movie.id} movie={movie} from={from} comingSoon={comingSoon} />
          ))}
        </div>
      )}
      {hasMore && (
        <div className="mt-6 flex justify-center">
          <Button variant="outline" onClick={onLoadMore} disabled={loadingMore}>
            {loadingMore ? "Loading…" : "Load more movies"}
          </Button>
        </div>
      )}
    </div>
  );
}

const DETAILS_ONLY_SOURCES = new Set(["popular", "trending"]);

function TmdbCard({
  movie,
  from = "search",
  comingSoon = false,
}: {
  movie: TmdbMovie;
  from?: string;
  comingSoon?: boolean;
}) {
  const img = posterUrl(movie.poster_path);
  const params = new URLSearchParams({
    title: movie.title,
    overview: movie.overview ?? "",
    poster: movie.poster_path ?? "",
    release_date: movie.release_date ?? "",
    rating: String(movie.vote_average ?? 0),
    from,
  });
  const detailsOnly = DETAILS_ONLY_SOURCES.has(from);
  return (
    <Link href={`/movie/tmdb-${movie.id}?${params.toString()}`}>
      <Card className="h-full overflow-hidden transition-shadow hover:shadow-md">
        <div className="relative">
          {img ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={img} alt={movie.title} className="aspect-[2/3] w-full object-cover" />
          ) : (
            <div className="flex aspect-[2/3] items-center justify-center bg-muted text-3xl font-bold text-muted-foreground">
              {movie.title.charAt(0)}
            </div>
          )}
          {comingSoon && (
            <Badge className="absolute left-2 top-2" variant="secondary">
              Coming Soon
            </Badge>
          )}
        </div>
        <CardContent className="space-y-1 pt-3">
          <h3 className="line-clamp-1 text-sm font-semibold">{movie.title}</h3>
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>{movie.release_date?.slice(0, 4) || "—"}</span>
            <span>★ {movie.vote_average?.toFixed(1)}</span>
          </div>
          <p className="pt-1 text-xs font-medium text-primary">
            {detailsOnly ? "View details" : "View details & cinemas"}
          </p>
        </CardContent>
      </Card>
    </Link>
  );
}
