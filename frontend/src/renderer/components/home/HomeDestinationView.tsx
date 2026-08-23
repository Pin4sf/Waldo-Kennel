import type { HomeDestination, HomeFixtureState } from "../../lib/home-fixture";
import { HomeDailyClose } from "./HomeDailyClose";
import { HomeChat } from "./HomeChat";
import { HomeHistory } from "./HomeHistory";
import { HomeMemoryReview } from "./HomeMemoryReview";
import { HomeOpenLoops } from "./HomeOpenLoops";
import { useTranslation } from "react-i18next";

export function HomeDestinationView({
  destination,
  fixture,
  initialRecordId,
  previewEnabled,
}: {
  destination: Exclude<HomeDestination, "today">;
  fixture: HomeFixtureState;
  initialRecordId?: string;
  previewEnabled: boolean;
}) {
  const { t } = useTranslation();
  if (destination === "chat") {
    return (
      <HomeChat
        contextLabel={`${t("home.visual.title")} · ${t("home.visual.navigation.chat")}`}
        previewEnabled={previewEnabled}
      />
    );
  }

  if (destination === "open_loops") {
    return <HomeOpenLoops fixture={fixture} />;
  }

  if (destination === "memory") {
    return <HomeMemoryReview fixture={fixture} />;
  }

  if (destination === "daily_close") {
    return <HomeDailyClose fixture={fixture} />;
  }

  return <HomeHistory fixture={fixture} initialRecordId={initialRecordId} previewEnabled={previewEnabled} />;
}
