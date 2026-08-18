import { formatTimeCompact as formatPortableTimeCompact } from "@pin4sf/kennel-product-ui";
import { appI18n, type MessageKey } from "../i18n";

export function formatTimeCompact(isoDate: string | null | undefined): string {
	return formatPortableTimeCompact(isoDate, {
		translate: (key, values) => appI18n.t(key as MessageKey, values),
	});
}
