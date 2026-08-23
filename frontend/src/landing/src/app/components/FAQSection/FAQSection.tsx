"use client";

import { useState } from "react";
import { HiPlus } from "react-icons/hi2";
import type { FAQItem } from "./constants";
import { FAQ_ITEMS } from "./constants";

function FAQAccordionItem({
  item,
  isOpen,
  onToggle,
}: {
  item: FAQItem;
  isOpen: boolean;
  onToggle: () => void;
}) {
  return (
    <div>
      <button
        type="button"
        onClick={onToggle}
        className="group flex w-full items-center justify-between py-6 text-left outline-none"
      >
        <span className="text-base sm:text-lg font-medium text-foreground pr-4">
          {item.question}
        </span>
        <HiPlus
          className={`h-5 w-5 shrink-0 text-muted-foreground transition-transform duration-normal ease-out ${
            isOpen ? "rotate-45" : ""
          }`}
        />
      </button>
      {/*
       * A grid whose single row animates between 0fr and 1fr, rather than a
       * JS height:auto animation. The row track is composited by the browser
       * off the main thread, so the answer still opens smoothly while the
       * page is loading images below it, and the panel stays in the DOM so
       * search and in-page find can reach a collapsed answer.
       */}
      <div
        className="grid transition-[grid-template-rows] duration-slow ease-out motion-reduce:transition-none"
        style={{ gridTemplateRows: isOpen ? "1fr" : "0fr" }}
      >
        <div className="overflow-hidden">
          <p
            aria-hidden={!isOpen}
            className={`pb-6 text-base text-muted-foreground leading-relaxed pr-12 transition-opacity duration-normal ease-out motion-reduce:transition-none ${
              isOpen ? "opacity-100" : "opacity-0"
            }`}
          >
            {item.answer}
          </p>
        </div>
      </div>
    </div>
  );
}

export function FAQSection() {
  const [openIndex, setOpenIndex] = useState<number | null>(null);

  const handleToggle = (index: number) => {
    setOpenIndex(openIndex === index ? null : index);
  };

  return (
    <section id="faq" className="relative px-4 py-16 sm:px-8 sm:py-20 lg:px-[30px] lg:py-24">
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_1.5fr] gap-12 xl:gap-20">
          <div className="xl:sticky xl:top-24 xl:self-start">
            <h2 className="text-3xl sm:text-4xl xl:text-5xl font-medium tracking-[-0.5px] text-foreground leading-[1.1]">
              Frequently
              <br />
              asked questions
            </h2>
          </div>

          <div>
            <div className="w-full divide-y divide-border">
              {FAQ_ITEMS.map((item, index) => (
                <FAQAccordionItem
                  key={item.question}
                  item={item}
                  isOpen={openIndex === index}
                  onToggle={() => handleToggle(index)}
                />
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
