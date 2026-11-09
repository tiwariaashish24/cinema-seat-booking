"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { api, type Booking } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

declare global {
  interface Window {
    Razorpay?: new (options: Record<string, unknown>) => { open: () => void };
  }
}

function loadRazorpayScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (window.Razorpay) return resolve();
    const s = document.createElement("script");
    s.src = "https://checkout.razorpay.com/v1/checkout.js";
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("Failed to load Razorpay checkout"));
    document.body.appendChild(s);
  });
}

export function BookingView() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;

  const [booking, setBooking] = useState<Booking | null>(null);
  const [error, setError] = useState("");
  const [now, setNow] = useState(Date.now());
  const [payOpen, setPayOpen] = useState(false);
  const [amount, setAmount] = useState("500");
  const [busy, setBusy] = useState(false);
  const [payMsg, setPayMsg] = useState("");
  const [testMode, setTestMode] = useState<boolean | null>(null);
  const razorpayKey = process.env.NEXT_PUBLIC_RAZORPAY_KEY_ID ?? "";

  const refresh = useCallback(() => {
    api
      .getBooking(id)
      .then(setBooking)
      .catch((e) => setError(String(e)));
  }, [id]);

  useEffect(() => {
    refresh();
    const t = setInterval(refresh, 5000);
    const tick = setInterval(() => setNow(Date.now()), 1000);
    api.paymentStatus().then((s) => setTestMode(s.test_mode)).catch(() => setTestMode(null));
    return () => {
      clearInterval(t);
      clearInterval(tick);
    };
  }, [refresh]);

  if (error) {
    return (
      <Card className="mx-auto max-w-lg">
        <CardContent className="pt-6 text-center">
          <p className="text-destructive">{error}</p>
          <Button
            variant="outline"
            className="mt-4"
            render={<Link href="/" />}
          >
            Back to movies
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (!booking) {
    return <Card className="mx-auto max-w-lg"><CardContent className="pt-6">Loading…</CardContent></Card>;
  }

  const expired =
    booking.status === "held" && new Date(booking.expires_at).getTime() <= now;
  const remaining = Math.max(0, Math.floor((new Date(booking.expires_at).getTime() - now) / 1000));
  const mm = String(Math.floor(remaining / 60)).padStart(2, "0");
  const ss = String(remaining % 60).padStart(2, "0");

  async function confirm() {
    setBusy(true);
    setPayMsg("");
    try {
      await api.confirmBooking(id);
      refresh();
    } catch (e) {
      setPayMsg(String(e));
    } finally {
      setBusy(false);
    }
  }

  async function cancel() {
    if (!window.confirm("Cancel this booking? The seat will be released.")) return;
    setBusy(true);
    try {
      await api.cancelBooking(id);
      router.push("/my-bookings");
    } catch (e) {
      setPayMsg(String(e));
      setBusy(false);
    }
  }

  async function startPayment() {
    setBusy(true);
    setPayMsg("");
    try {
      const rupees = parseFloat(amount);
      if (!rupees || rupees <= 0) throw new Error("Enter a valid amount");
      const order = await api.createOrder(id, Math.round(rupees * 100));

      if (razorpayKey) {
        await loadRazorpayScript();
        const rzp = new window.Razorpay!({
          key: razorpayKey,
          amount: order.amount,
          currency: order.currency,
          name: "SeatBook Cinemas",
          description: `Seat ${booking?.seat_id} — Movie ${booking?.movie_id}`,
          order_id: order.id,
          handler: async (resp: { razorpay_order_id: string; razorpay_payment_id: string; razorpay_signature: string }) => {
            try {
              await api.verifyPayment({
                booking_id: id,
                order_id: resp.razorpay_order_id,
                payment_id: resp.razorpay_payment_id,
                signature: resp.razorpay_signature,
              });
              setPayOpen(false);
              refresh();
            } catch (e) {
              setPayMsg(String(e));
            }
          },
        });
        rzp.open();
      } else {
        // Test mode: simulate a captured payment, then verify.
        const payment = await api.mockPay(order.id);
        await api.verifyPayment({
          booking_id: id,
          order_id: order.id,
          payment_id: payment.id,
          signature: "mock",
        });
        setPayOpen(false);
        refresh();
      }
    } catch (e) {
      setPayMsg(String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-lg space-y-4">
      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle className="text-lg">Booking details</CardTitle>
          <Badge
            variant={
              booking.status === "confirmed"
                ? "default"
                : expired
                  ? "destructive"
                  : "secondary"
            }
          >
            {expired ? "expired" : booking.status}
          </Badge>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div className="flex justify-between"><span className="text-muted-foreground">Booking ID</span><span className="font-mono text-xs">{booking.id}</span></div>
          <Separator />
          <div className="flex justify-between"><span className="text-muted-foreground">Movie</span><span>{booking.movie_id}</span></div>
          <div className="flex justify-between"><span className="text-muted-foreground">Seat</span><span className="font-semibold">{booking.seat_id}</span></div>
          <div className="flex justify-between"><span className="text-muted-foreground">User</span><span>{booking.user_id}</span></div>
          {booking.status === "held" && !expired && (
            <>
              <Separator />
              <div className="flex justify-between">
                <span className="text-muted-foreground">Hold expires in</span>
                <span className="font-mono font-semibold">{mm}:{ss}</span>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {payMsg && <p className="text-sm text-destructive">{payMsg}</p>}

      {booking.status === "held" && !expired && (
        <Card>
          <CardContent className="flex flex-col gap-2 pt-6">
            <Button onClick={() => setPayOpen(true)} disabled={busy}>
              Pay &amp; confirm
            </Button>
            <Button variant="outline" onClick={confirm} disabled={busy}>
              Confirm without payment
            </Button>
            <Button variant="ghost" className="text-destructive" onClick={cancel} disabled={busy}>
              Cancel booking
            </Button>
            {testMode !== null && (
              <p className="text-xs text-muted-foreground">
                Payment gateway: {testMode ? "test mode" : "production"}
                {razorpayKey ? "" : " — Razorpay key not set, test payments will be simulated."}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {booking.status === "confirmed" && (
        <Card>
          <CardContent className="pt-6 text-center">
            <p className="font-semibold text-green-600">Booking confirmed — enjoy the show!</p>
            <Button
              variant="outline"
              className="mt-3"
              render={<Link href="/" />}
            >
              Book another seat
            </Button>
          </CardContent>
        </Card>
      )}

      {expired && (
        <Card>
          <CardContent className="pt-6 text-center">
            <p className="text-muted-foreground">This hold has expired. The seat may have been released.</p>
            <Button
              variant="outline"
              className="mt-3"
              render={<Link href={`/movie/${booking.movie_id}`} />}
            >
              Pick another seat
            </Button>
          </CardContent>
        </Card>
      )}

      <Dialog open={payOpen} onOpenChange={setPayOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Pay for booking</DialogTitle>
            <DialogDescription>
              Seat {booking.seat_id} · Movie {booking.movie_id}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5 py-2">
            <Label htmlFor="amount">Amount (₹)</Label>
            <Input
              id="amount"
              type="number"
              min="1"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setPayOpen(false)} disabled={busy}>
              Close
            </Button>
            <Button onClick={startPayment} disabled={busy}>
              {busy ? "Processing…" : razorpayKey ? "Pay with Razorpay" : "Simulate test payment"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
