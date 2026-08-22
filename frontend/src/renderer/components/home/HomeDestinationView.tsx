import type { HomeDestination, HomeFixtureState } from "../../lib/home-fixture";
import { HomeDailyClose } from "./HomeDailyClose";
import { HomeHistory } from "./HomeHistory";
import { HomeMemoryReview } from "./HomeMemoryReview";
import { HomeOpenLoops } from "./HomeOpenLoops";

export function HomeDestinationView({
  destination,
  fixture,
}: {
  destination: Exclude<HomeDestination, "today">;
  fixture: HomeFixtureState;
}) {
  if (destination === "open_loops") {
    return <HomeOpenLoops fixture={fixture} />;
  }

  if (destination === "memory") {
    return <HomeMemoryReview fixture={fixture} />;
  }

  if (destination === "daily_close") {
    return <HomeDailyClose fixture={fixture} />;
  }

  return <HomeHistory fixture={fixture} />;
}
