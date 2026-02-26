/**
 * OfflineBanner.tsx — Sticky top bar shown when the device is offline.
 * Informs the user that actions are being queued for sync.
 * When coming back online, subscribes to sync events and shows a result toast.
 */

import { WifiOff } from "lucide-react";
import { useOnlineStatus } from "../hooks/useOnlineStatus";
import { useEffect, useRef } from "react";
import toast from "react-hot-toast";
import { onSync } from "../lib/sync";

export default function OfflineBanner() {
  const online = useOnlineStatus();
  const prevOnline = useRef(online);
  // Track verified/failed counts across the sync run
  const verified = useRef(0);
  const failed = useRef(0);

  useEffect(() => {
    if (prevOnline.current === online) return;
    prevOnline.current = online;

    if (!online) {
      toast(
        "You're offline. Actions will be queued and synced when you reconnect.",
        {
          id: "offline-toast",
          icon: "📶",
          duration: Infinity,
          style: { maxWidth: 380 },
        }
      );
    } else {
      toast.dismiss("offline-toast");

      // Reset counters and subscribe to the upcoming sync run
      verified.current = 0;
      failed.current = 0;

      const unsubOk   = onSync("sync:item:ok",   () => { verified.current += 1; });
      const unsubFail = onSync("sync:item:fail",  () => { failed.current  += 1; });
      const unsubDone = onSync("sync:done", () => {
        unsubOk();
        unsubFail();
        unsubDone();

        if (verified.current === 0 && failed.current === 0) {
          toast.success("Back online", { id: "online-toast", duration: 3000 });
        } else if (failed.current === 0) {
          toast.success(
            `✓ Synced ${verified.current} item${verified.current !== 1 ? "s" : ""}`,
            { id: "online-toast", duration: 4000 }
          );
        } else {
          toast(
            `Synced ${verified.current} ✓  ·  ${failed.current} failed — check the Events page`,
            { id: "online-toast", icon: "⚠️", duration: 6000 }
          );
        }
      });
    }
  }, [online]);

  if (online) return null;

  return (
    <div className="sticky top-16 z-40 flex items-center gap-2 bg-amber-500 px-4 py-2 text-sm font-medium text-white">
      <WifiOff className="h-4 w-4 shrink-0" />
      <span>Offline — actions queued for sync</span>
    </div>
  );
}
