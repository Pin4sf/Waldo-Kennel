import type { RefObject } from "react";
import type { HomeFixtureState } from "../../lib/home-fixture";
import { HomeBeforeNext } from "./HomeBeforeNext";
import { HomeCatchUp } from "./HomeCatchUp";
import { HomeEveningReview } from "./HomeEveningReview";
import { HomePlansChanged } from "./HomePlansChanged";
import { HomeQuietFocus } from "./HomeQuietFocus";

export type HomeContextPanelProps = {
  fixture: HomeFixtureState;
  headingRef: RefObject<HTMLHeadingElement | null>;
  scrollContainerRef: RefObject<HTMLElement | null>;
};

export function HomeContextPanel(props: HomeContextPanelProps) {
  switch (props.fixture.contextFlow) {
    case "catch_up":
      return <HomeCatchUp {...props} />;
    case "before_next":
      return <HomeBeforeNext fixture={props.fixture} headingRef={props.headingRef} />;
    case "plans_changed":
      return <HomePlansChanged fixture={props.fixture} headingRef={props.headingRef} />;
    case "evening_review":
      return <HomeEveningReview fixture={props.fixture} headingRef={props.headingRef} />;
    case "quiet_focus":
      return <HomeQuietFocus fixture={props.fixture} headingRef={props.headingRef} />;
  }
}
