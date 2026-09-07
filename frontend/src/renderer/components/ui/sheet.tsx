"use client";

import * as React from "react";
import { XIcon } from "lucide-react";
import { Dialog as SheetPrimitive } from "radix-ui";
import { useTranslation } from "react-i18next";

import { cn } from "@/lib/utils";

function Sheet({ ...props }: React.ComponentProps<typeof SheetPrimitive.Root>) {
	return <SheetPrimitive.Root data-slot="sheet" {...props} />;
}

function SheetTrigger({ ...props }: React.ComponentProps<typeof SheetPrimitive.Trigger>) {
	return <SheetPrimitive.Trigger data-slot="sheet-trigger" {...props} />;
}

function SheetClose({ ...props }: React.ComponentProps<typeof SheetPrimitive.Close>) {
	return <SheetPrimitive.Close data-slot="sheet-close" {...props} />;
}

function SheetPortal({ ...props }: React.ComponentProps<typeof SheetPrimitive.Portal>) {
	return <SheetPrimitive.Portal data-slot="sheet-portal" {...props} />;
}

function SheetOverlay({ className, ...props }: React.ComponentProps<typeof SheetPrimitive.Overlay>) {
	return (
		<SheetPrimitive.Overlay
			data-slot="sheet-overlay"
			className={cn(
				"dialog-overlay data-[state=open]:animate-overlay-in data-[state=closed]:animate-overlay-out motion-reduce:animate-none",
				className,
			)}
			{...props}
		/>
	);
}

function SheetContent({
	className,
	children,
	side = "right",
	showCloseButton = true,
	...props
}: React.ComponentProps<typeof SheetPrimitive.Content> & {
	side?: "top" | "right" | "bottom" | "left";
	showCloseButton?: boolean;
}) {
	const { t } = useTranslation();
	return (
		<SheetPortal>
			<SheetOverlay />
			<SheetPrimitive.Content
				data-slot="sheet-content"
				className={cn(
					"fixed z-overlay flex flex-col gap-4 bg-background shadow-lg motion-reduce:animate-none",
					side === "right" &&
						"inset-y-0 right-0 h-full w-3/4 border-l data-[state=open]:animate-sheet-in-right data-[state=closed]:animate-sheet-out-right sm:max-w-sm",
					side === "left" &&
						"inset-y-0 left-0 h-full w-3/4 border-r data-[state=open]:animate-sheet-in-left data-[state=closed]:animate-sheet-out-left sm:max-w-sm",
					side === "top" &&
						"inset-x-0 top-0 h-auto border-b data-[state=open]:animate-sheet-in-top data-[state=closed]:animate-sheet-out-top",
					side === "bottom" &&
						"inset-x-0 bottom-0 h-auto border-t data-[state=open]:animate-sheet-in-bottom data-[state=closed]:animate-sheet-out-bottom",
					className,
				)}
				{...props}
			>
				{children}
				{showCloseButton && (
					<SheetPrimitive.Close className="absolute top-4 right-4 rounded-xs opacity-70 ring-offset-background transition-[opacity,transform] duration-fast ease-out hover:opacity-100 active:scale-press focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:outline-hidden disabled:pointer-events-none data-[state=open]:bg-secondary">
						<XIcon className="size-icon-base" />
						<span className="sr-only">{t("common.close")}</span>
					</SheetPrimitive.Close>
				)}
			</SheetPrimitive.Content>
		</SheetPortal>
	);
}

function SheetHeader({ className, ...props }: React.ComponentProps<"div">) {
	return <div data-slot="sheet-header" className={cn("flex flex-col gap-1.5 p-4", className)} {...props} />;
}

function SheetFooter({ className, ...props }: React.ComponentProps<"div">) {
	return <div data-slot="sheet-footer" className={cn("mt-auto flex flex-col gap-2 p-4", className)} {...props} />;
}

function SheetTitle({ className, ...props }: React.ComponentProps<typeof SheetPrimitive.Title>) {
	return (
		<SheetPrimitive.Title
			data-slot="sheet-title"
			className={cn("font-semibold text-foreground", className)}
			{...props}
		/>
	);
}

function SheetDescription({ className, ...props }: React.ComponentProps<typeof SheetPrimitive.Description>) {
	return (
		<SheetPrimitive.Description
			data-slot="sheet-description"
			className={cn("text-sm text-muted-foreground", className)}
			{...props}
		/>
	);
}

export { Sheet, SheetTrigger, SheetClose, SheetContent, SheetHeader, SheetFooter, SheetTitle, SheetDescription };
