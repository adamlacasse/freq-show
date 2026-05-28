# Persona Cleanup — Implementation Plan

> **Goal:** Bring the running FreqShow Angular app into compliance with [`specs/persona-and-voice.md`](../../specs/persona-and-voice.md) by removing all engineer-facing and hiring-manager-facing copy and CTAs from the user-facing surface, reframing schema/infra vocabulary in listener voice, and reducing the home page to a tagline + search entrance. This is a pure copy-and-markup pass; no new data sources, no backend changes, no new components, no new routes, no new dependencies.

This plan is the build-time companion to the persona spec. The spec settles *what is allowed*; this plan settles *which specific strings and blocks in the current codebase to remove or rewrite, and in what order*. If they conflict, the spec wins and this plan gets corrected.

---

## Status

_Last updated 2026-05-28._

| Phase | Status | Notes |
| --- | --- | --- |
| 1. Home page strip | ⬜ Not started | Remove brochure sections + API-doc CTAs from `home.component.html`; remove now-unused arrays from `home.component.ts`. Result: hero (tagline) + search box only. |
| 2. Global chrome reframe | ⬜ Not started | Footer attribution line + cold-start banner in `app.component.html`. |
| 3. Discover copy pass | ⬜ Not started | Strapline and zero-results empty state in `discover.component.html`. |
| 4. Artist detail copy pass | ⬜ Not started | Section header, empty states, attribution micro-credits in `artist-detail.component.html`. |
| 5. Album detail copy pass | ⬜ Not started | Field labels and the "Additional classifications" block in `album-detail.component.html`. |
| 6. Verification | ⬜ Not started | Build the frontend, walk every primary route, run the grep checklist below. |

**To resume work:** read this Status section, jump to the first non-✅ phase below, and proceed. Phases 1–5 are independent and may land in any order; Phase 6 gates declaring this iteration done.

**Open questions / decisions parked:**

- The cold-start banner copy (Phase 2) replaces a real infra symptom (Render free-tier cold start). The replacement copy is a draft; if the team prefers to hide the banner entirely and accept a slow first request silently, that's an acceptable variant. The spec forbids server-voice narration, not the banner itself.
- Album detail's `Type` and `Also classified as` labels (Phase 5) still expose MusicBrainz's release-group taxonomy through the values themselves (`Album`, `EP`, `Single`, `Compilation`, etc.). A future pass may translate these into everyday English. Out of scope here.

---

## What This Plan Is Doing

This is a copy and markup pass — the first iteration in the broader repositioning of FreqShow around the music-nerd persona ([see Roadmap Context below](#roadmap-context)). It exercises:

- A consistent voice rewrite across every user-facing template in `apps/frontend/src/app/`.
- The removal of three home-page sections (the brochure) and two hero CTAs (API-doc links).
- A reduction in `home.component.ts` of two now-unused readonly properties (`highlights`, `sources`) and, if it becomes unused, the `CommonModule` import.
- A re-anchoring of attribution from build-manifest framing to credit-at-the-point-of-use framing, without removing the attribution itself.

It is **not** doing any of the following:

- No new components, services, routes, or data fetches.
- No backend changes. No changes to the OpenAPI contract.
- No changes to component logic, routing, search behavior, or the discovery pipeline mechanics.
- No new tests beyond updating existing `.spec.ts` files where they assert on removed DOM.
- No introduction of the planned future features (front door, performer credits, smart-linking infrastructure, collections, etc.). Those are separate plans; see [Roadmap Context](#roadmap-context).

---

## Scope and Deferrals

### In scope for this iteration

- Removals and copy rewrites in:
  - `apps/frontend/src/app/app.component.html`
  - `apps/frontend/src/app/pages/home/home.component.html`
  - `apps/frontend/src/app/pages/home/home.component.ts`
  - `apps/frontend/src/app/pages/discover/discover.component.html`
  - `apps/frontend/src/app/pages/artist-detail/artist-detail.component.html`
  - `apps/frontend/src/app/pages/album-detail/album-detail.component.html`
- Updates to corresponding `*.spec.ts` files only as needed to keep existing tests passing after DOM removal.

### Deferred (future iterations / separate plans)

- A real listener-first front door for the home page (replacing the now-bare hero + search with featured threads, recently viewed, a "drop me somewhere" entrance). Roadmap item #6.
- Performer credits on albums and tracks (the data unlock that makes the rabbit hole work). Roadmap items #1 and #2.
- The comparative "albums they're NOT on" view. Roadmap item #3.
- A smart-linking infrastructure pass (link components, hover previews, the wander trail). Roadmap item #4.
- Discovery as connective tissue (contextual suggestions on every page; embedding people, not just albums). Roadmap item #5.
- Artist images and album cover art (both detail pages currently render only placeholder SVGs).
- Backend or schema changes of any kind.
- Re-writing `README.md` to drop the engineer-facing narrative. README serves GitHub landing visitors and is not part of the runtime app surface. Repointing the engineer/hiring-manager story to adamlacasse.dev is a separate, parallel workstream owned outside this codebase.

---

## Roadmap Context

This iteration is item #7 of a seven-item slate that re-anchors the product around the music-nerd persona ([spec](../../specs/persona-and-voice.md)) and the **"AllMusic + Wikipedia + AI-powered smart rabbit-holing"** north star. The slate, in dependency order:

1. **People as first-class entities.** Person pages distinct from band pages, showing everything a person played on across bands, sessions, and guest spots. Requires fetching MusicBrainz performance relationships (`release-group-rels`, `recording-rels` with instrument attributes) — not currently fetched. Today's `LookupArtist` uses `inc=tags+artist-rels` only, and discography is browsed via release-groups credited to the artist as primary, so e.g. Chris Squire's page would surface only his solo records, not his bass work on every Yes album.
2. **Performer credits on albums and tracks.** "Who played what" on each release and ideally each track, each name a link. Requires the data from #1 plus enriched track fetches (e.g. recording-level artist credits and `recording-rels`). Today's `getReleaseRecordings` fetches `inc=recordings` only — track number, title, and length, with no performer information.
3. **The comparative / negative view.** Band lineup over time; per-album participation; "plays on X of N albums" with the gaps named. Signature delight feature.
4. **Smart-linking infrastructure.** Link component, hover-preview cards, wander-trail/breadcrumb, routing for new entity types. Built against today's referential links; designed so #2 and #5 plug straight in. *Distinguish* this from semantic ("smart") links, which come from #5.
5. **Discovery as connective tissue.** Extend the existing `/discover` pipeline to power contextual "where to wander next" on entity pages. Likely requires embedding people, not just albums (today's `album_embeddings` table is the only embedded surface).
6. **A rabbit-hole front door + trail.** Replace the now-bare home with a real entrance (featured thread, "drop me somewhere," recently-viewed breadcrumb).
7. **Persona cleanup** *(this plan).* Strip out the engineer / hiring-manager UI; reframe schema-speak and infra-speak.

This iteration is sequenced first because it has no data dependencies and is fully reversible, while every other item assumes a listener-first surface to build onto.

---

## Required Setup

- Node 18+ and the frontend dev environment described in `README.md`. The plan does not require backend, database, or any external API keys to verify the changes themselves; running the backend is only needed if the implementer wants to visually verify against live data.
- No new dependencies. No `package.json` changes.
- Verification (Phase 6) requires only `npm install` and the project's existing `npm run build`, `npm test`, and `npm start` commands in `apps/frontend/`.

---

## Step-by-Step Implementation

Each phase below lists exact files, exact blocks to remove, and before/after strings for rewrites. Implementers should make the smallest possible change to achieve each item and should not add new copy, components, or styles beyond what is specified.

### Phase 1 — Home page strip

Goal: reduce the home page to exactly the hero (eyebrow + headline + description + search box). Remove everything else.

**File: `apps/frontend/src/app/pages/home/home.component.html`**

Remove the following blocks. Block boundaries refer to the file as of 2026-05-28.

1. Remove the inner `<div class="mt-6 flex flex-wrap items-center justify-center gap-4">…</div>` block that sits immediately after `<app-search></app-search>`. This block contains the two `<a>` CTAs linking to `https://musicbrainz.org/doc/MusicBrainz_API` and `https://en.wikipedia.org/api/rest_v1/`. Remove the entire wrapping `<div>` and both child `<a>` tags. Keep `<app-search></app-search>` and its parent `<div class="mt-10">`.

2. Remove the entire `<div class="w-full max-w-md rounded-3xl border border-white/10 bg-white/5 p-6 …">…</div>` block — the "What is live now" panel that follows the hero text within the outer flex container. Its content begins with `<p class="font-semibold uppercase tracking-wide text-freq-cream/70">What is live now</p>` and ends with the closing `</div>` of the panel.

3. After removing the panel, the outer hero `<div class="mx-auto flex max-w-6xl flex-col gap-10 px-6 py-20 lg:flex-row lg:items-end">` no longer needs the row layout (it has a single remaining child). Simplify the class list to: `<div class="mx-auto flex max-w-6xl flex-col gap-10 px-6 py-20">`. Optionally tighten `py-20` if the resulting hero feels too tall; visual judgment, not mandatory.

4. Remove the entire `<section class="mx-auto max-w-6xl px-6 py-16">…</section>` — the "Core experiences" section. Identified by `<h2 …>Core experiences</h2>`.

5. Remove the entire `<section class="mx-auto max-w-6xl px-6 pb-24">…</section>` — the "Current sources" section. Identified by `<h2 …>Current sources</h2>`.

After these removals, `home.component.html` contains exactly one `<section>` (the hero with the search box).

**File: `apps/frontend/src/app/pages/home/home.component.ts`**

- Remove the `readonly highlights = […]` property and the `readonly sources = […]` property in their entirety. The `readonly hero = { … }` property remains.
- After removal, the template uses interpolation only (no `*ngIf`, `*ngFor`, `@for`, or `@if`). `CommonModule` is no longer required. Remove `CommonModule` from both the `imports` array on the `@Component` decorator and the top-of-file `import { CommonModule } from '@angular/common';` line. `SearchComponent` and `Component` imports remain.

**File: `apps/frontend/src/app/pages/home/home.component.spec.ts`**

- If the existing spec asserts on the presence of `What is live now`, `Core experiences`, `Current sources`, or any of the `highlights`/`sources` array text, remove those assertions. Do not add new assertions in this phase. The spec must continue to pass after the DOM reductions.

**Acceptance for Phase 1:**

- The `/` route renders only the hero (eyebrow + headline + description) and the search component.
- No links to API documentation appear anywhere on the home route.
- `home.component.ts` contains no `highlights` or `sources` property and does not import `CommonModule`.
- `npm run build` succeeds without warnings introduced by this change.
- `home.component.spec.ts` passes.

### Phase 2 — Global chrome reframe

Goal: reframe the footer attribution line and the cold-start banner without removing either.

**File: `apps/frontend/src/app/app.component.html`**

- Replace the second `<span>` in the footer:
  - **Before:** `<span>Current sources: MusicBrainz, Wikipedia, and Discogs.</span>`
  - **After:** `<span>Music data via MusicBrainz, Wikipedia, and Discogs.</span>`
- Replace the cold-start banner text inside the `@if ((apiStatus.status$ | async) === 'waking')` block:
  - **Before:** `Service is waking up — first request may be slow.`
  - **After:** `Just a moment — pulling the first record off the shelf.`
- Do not change the banner's structure, accent colors, or the pulsing indicator dot. Do not change the left-hand footer span (`FreqShow - for those who still read liner notes.`).

**Acceptance for Phase 2:**

- The footer reads `FreqShow - for those who still read liner notes.` (unchanged) on the left and `Music data via MusicBrainz, Wikipedia, and Discogs.` on the right.
- The cold-start banner reads `Just a moment — pulling the first record off the shelf.` with the leading dot animation unchanged.

### Phase 3 — Discover copy pass

Goal: rewrite the strapline and the zero-results empty state in listener voice.

**File: `apps/frontend/src/app/pages/discover/discover.component.html`**

- Replace the introductory `<p>` immediately under the `Find the next record.` heading:
  - **Before:** `Describe a listening mood, reference point, or texture. FreqShow will search the hydrated album index and return five ranked picks.`
  - **After:** `Describe what you want to hear — a mood, a reference, a texture. FreqShow will pick five records that fit.`
- Replace the empty-state copy in the `@else` branch of the picks loop:
  - **Before:** `No picks matched after the avoid filters.`
  - **After:** `No records matched what you asked for. Try loosening one of the constraints.`
- Do not change the form labels, the example chips, the submit button text, the `Reading the shelves...` loading copy (it is in-voice), or the interpreted-query chip styling and content.

**Acceptance for Phase 3:**

- The Discover route's strapline reads the After string above.
- Submitting a query that yields zero picks renders the After empty-state string.

### Phase 4 — Artist detail copy pass

Goal: rewrite the empty states, section header, and attribution micro-credits.

**File: `apps/frontend/src/app/pages/artist-detail/artist-detail.component.html`**

1. Section header — change `<h2 class="text-xl font-semibold text-white">Genres & Tags</h2>` to `<h2 class="text-xl font-semibold text-white">Genres</h2>`.

2. Biography empty state — inside `<ng-template #noBiography>`:
   - Keep the `No biography available` headline.
   - **Before subline:** `We couldn't find biographical information for this artist in our sources.`
   - **After subline:** `No biography yet for this artist.`

3. Genres empty state — inside `<ng-template #noGenres>`:
   - Keep the `No genre information available` headline.
   - **Before subline:** `Genre tags haven't been assigned to this artist yet in our database.`
   - **After subline:** `No genres listed for this artist yet.`

4. Biography attribution micro-credit:
   - Anchor text when `artist.biographyUrl` is present — **Before:** `Read the Wikipedia source` → **After:** `Read on Wikipedia`.
   - `#wikipediaLabel` template content — **Before:** `Information sourced from Wikipedia` → **After:** `Bio via Wikipedia`.

5. Genres attribution micro-credit:
   - **Before:** `Genre data sourced from MusicBrainz community tags`
   - **After:** `Genres via MusicBrainz`

6. Albums empty state:
   - **Before:** `Album information is not available for this artist.`
   - **After:** `No albums listed for this artist yet.`

7. Leave the `Sorted by year (newest first)` indicator unchanged — it is a useful affordance, not persona leakage.

**Acceptance for Phase 4:**

- No string on the artist detail page contains the substrings `our sources`, `our database`, or `Genres & Tags`.
- Wikipedia and MusicBrainz attribution remains visible at the point of the contributed content, in the shorter forms specified.

### Phase 5 — Album detail copy pass

Goal: soften the schema-vocabulary labels and the "Release Information" framing.

**File: `apps/frontend/src/app/pages/album-detail/album-detail.component.html`**

1. **Before:** `<div class="text-xs text-freq-cream/50 mb-2">Additional classifications:</div>`
   **After:** `<div class="text-xs text-freq-cream/50 mb-2">Also classified as:</div>`

2. In the bottom "Release Information" section, replace the section header and the field labels (the `<div class="text-xs text-freq-cream/50 mb-1">…</div>` lines):
   - Section header — **Before:** `Release Information` → **After:** `Release details`.
   - **Before:** `First Release Date` → **After:** `First released`.
   - **Before:** `Release Type` → **After:** `Type`.
   - **Before:** `Record Label` → **After:** `Label`.
   - `Genre` label — unchanged.

3. Tracklist empty state:
   - **Before:** `Track listing is not available for this album.`
   - **After:** `No tracks listed for this album yet.`

4. Reviews empty state:
   - **Before:** `Reviews are not available for this album.`
   - **After:** `No review yet for this album.`

5. Do not change the top metadata chips (year, primary type, genre), the secondary-types list rendering, the cover-art placeholder SVG, the review rendering structure, or the `Read full review →` link.

**Acceptance for Phase 5:**

- No field label on the album detail page reads `Release Type`, `First Release Date`, `Record Label`, or `Release Information`.
- The "Additional classifications" label reads `Also classified as:`.

### Phase 6 — Verification

Run from `apps/frontend/`:

1. `npm install` (if not already done).
2. `npm run build` — must succeed.
3. Run the existing spec command (whatever `package.json` defines — typically `npm test`); all specs must pass. The only intended spec change is removing assertions on deleted DOM in Phase 1.
4. `npm start`, then walk these routes in a browser:
   - `/` — verify only the hero and search box render. No `What is live now`, no `Core experiences`, no `Current sources`, no API-doc CTAs.
   - `/discover` — verify the After strapline; submit a deliberately under-specified query and confirm the After empty state.
   - `/artists/5b11f4ce-a62d-471e-81fc-a69a8278c7da` (Nirvana) — verify the Genres header reads `Genres`, attribution reads `Bio via Wikipedia` and `Genres via MusicBrainz`, empty states (if any apply) match Phase 4.
   - `/albums/1b022e01-4da6-387b-8658-8678046e4cef` (Nevermind) — verify the `Release details` section uses the new labels.
   - Trigger the cold-start banner if possible (against a paused backend) — verify the After banner copy.
5. **Grep checklist.** From the repo root, run a grep across `apps/frontend/src/app/` for each string below; every grep must return zero matches.

   - `What is live now`
   - `Core experiences`
   - `Current sources`
   - `hydrated album index`
   - `avoid filters`
   - `in our sources`
   - `in our database`
   - `Genres & Tags`
   - `Additional classifications`
   - `Release Type`
   - `Release Information`
   - `First Release Date`
   - `Record Label`
   - `Information sourced from Wikipedia`
   - `Genre data sourced from MusicBrainz community tags`
   - `musicbrainz.org/doc/MusicBrainz_API`
   - `en.wikipedia.org/api/rest_v1`
   - `Service is waking up`

If all greps return zero and the walkthrough confirms the new strings, Phase 6 is complete.

---

## Acceptance Criteria (Overall)

This iteration is done when:

- All six phases above are complete.
- The grep checklist in Phase 6 returns zero matches.
- A walk-through of every primary route confirms compliance with `specs/persona-and-voice.md`.
- `npm run build` and the existing test command pass.

---

## Out of Scope (explicit)

- Any backend, schema, or API change.
- Any new component, route, service, or dependency.
- Replacement content for the now-bare home page (deferred to roadmap item #6).
- Any change to `README.md`, `agent-context/`, or any other developer-facing documentation in this repo.
- Cover art / artist images.
- Performer credits / smart links / discovery extensions / collections / trails.
