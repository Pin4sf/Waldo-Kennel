import en from "./en.json";

/** English is the source-of-truth catalog; keys are typed from it. */
export const enMessages = en;

export type MessageKey = keyof typeof enMessages;

type PluralCategory = "zero" | "one" | "two" | "few" | "many" | "other";
export type PluralMessageKey = MessageKey extends infer Key extends string
	? Key extends `${infer Base}_${PluralCategory}`
		? Base
		: never
	: never;

export type MessageCatalog = Record<MessageKey, string>;

export function catalogFor(_locale?: string): Readonly<Record<string, string>> {
	return enMessages;
}
