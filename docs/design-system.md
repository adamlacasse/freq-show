# FreqShow frontend design system

A short reference for the tokens and component classes in
`apps/frontend/src/styles.css` and `tailwind.config.js`. The goal is
consistency, not novelty: reach for one of these before hand-typing a
card/button/badge/input utility string again.

## Palette (`tailwind.config.js`)

| Token | Hex | Role |
|---|---|---|
| `freq-ink` | `#0f172a` | Page background |
| `freq-midnight` | `#111827` | Header/footer background |
| `freq-teal` | `#2dd4bf` | Primary accent — interactive, primary CTAs |
| `freq-amber` | `#fbbf24` | Secondary accent — ratings, tertiary metadata |
| `freq-rose` | `#fb7185` | **Reserved for destructive/error/negative states only** (sign out, retry-after-error, remove from collection). Not a decorative rotation color — see "Rose collision" below. |
| `freq-cream` | `#f5f1e0` | Body text, used at opacity steps (`/40`–`/90`) for hierarchy instead of separate grays |

## Surfaces (`@layer components` in `styles.css`)

Layered translucency over `freq-ink`, darkest to lightest:

| Class | Value | Use for |
|---|---|---|
| `.surface-sunken` | `bg-white/[0.02]` | Nested rows inside a panel (track list, review card) |
| `.surface-base` | `bg-white/[0.04]` | Default panel/card body, inputs |
| `.surface-raised` | `bg-white/5` | Hero header cards, image placeholders |
| `.surface-hover` | `bg-white/10` | Hover/active state, neutral chips |
| `.surface-strong` | `bg-white/20` | Secondary button fill, strong neutral accent |

Borders follow the same idea: `.border-subtle` (`/5`), `.border-base` (`/10`), `.border-strong` (`/20`).

## Components

- **Cards**: `.card` (rounded-3xl panel, most content sections), `.card-hero`
  (rounded-3xl, top-of-page header), `.card-inset` (rounded-xl, nested row),
  `.card-interactive` (rounded-2xl, clickable tile — search results,
  discography, related artists, collection grid).
- **Buttons**: `.btn-primary` (teal fill, main CTA), `.btn-secondary`
  (neutral fill), `.btn-ghost` (outlined, back buttons / low-emphasis
  actions), `.btn-nav` (header nav links), `.btn-danger-ghost` (retry /
  destructive-adjacent).
- **Badges**: `.badge-teal` / `.badge-rose` / `.badge-amber` / `.badge-neutral`
  — small `rounded-full` tag, `text-xs font-semibold`. Note: the larger
  metadata pills on artist/album headers (`text-sm`, no `.badge` base) are a
  deliberately different, bigger variant — don't force them into `.badge-*`.
- **Chips**: `.chip-muted` — small muted tag (aliases, "also known as",
  secondary-type overrides).
- **Inputs**: `.input-field`.

All of the above are plain Tailwind utility classes underneath (`@apply`),
so you can still layer one-off utilities after the class name to override
sizing (e.g. `class="btn-primary px-6 py-3"` for a larger CTA) — Tailwind's
utilities layer always wins over the components layer, so trailing
utilities safely override the base class's padding/size without touching
its color/behavior.

## The rose collision (fixed 2026-09-27)

Before this pass, `freq-rose` was used two ways: as the error/destructive
color everywhere, *and* as a decorative "2nd color in a 3-color rotation"
for genre tags and metadata pills (country, primary type). That's a
semantic collision — a color that means "something went wrong" was also
just "the pink one." Genre-tag rotation and the country/primaryType
metadata pills were changed to use a neutral (`surface-strong`) in that
slot instead. `freq-rose` is now reserved for destructive/negative
meaning only. The one intentional exception is the "remove from
collection" toggle state on the album page, which is a legitimate
negative/undo action, not decoration.

## What this pass did and didn't touch

Done: named the surface/border scale, added the component classes above,
applied them everywhere the visual output was byte-identical (or a
sub-1%-opacity normalization — a couple of `bg-white/[0.03]` panels were
folded into the `.surface-base` (`0.04`) level, since they were meant to
be the same "default panel" role and had just drifted apart).

Left alone (each is a legitimate one-off, not a bug, but worth a look in
a future "more punch" pass since they don't map cleanly to a single
scale):
- The auth modal's input/submit-button sizing (`rounded-xl`, larger
  padding) doesn't match the page-level `.input-field`/`.btn-primary`
  sizing — modals arguably deserve their own scale, but it's currently
  just accidental drift.
- A few small badges (discography year tag, Spotify/Album link buttons on
  the discover results) use their own padding/opacity rather than
  `.badge-*` — sizes are close but not identical, so they weren't forced.
- The collection grid card (`rounded-3xl` + hover) doesn't match either
  `.card` (no hover) or `.card-interactive` (`rounded-2xl`) — it's a
  legitimate third "hover-highlighted static card" pattern that could
  become its own class if it recurs elsewhere.
