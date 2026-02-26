/**
 * OfflineBanner.tsx — Sticky top bar shown when the device is offline.
 * Informs the user that actions are being queued for sync.
 */

import { Wifi, WifiOff } from "lucide-react";
import { useOnlineStatus } from "../hooks/useOnlineStatus";
import { useEffect, useRef } from "react";
import toast from "react-hot-toast";

export default function OfflineBanner() {
  const online = useOnlineStatus();
  const prevOnline = useRef(online);

  // Show a toast notification on status change (not on initial render)
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
      toast.success("Back online — syncing your queued actions…", {
        id: "online-toast",
        duration: 4000,
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
