# Embedding Provider Benchmark — Results

_Corpus: 48 albums. Queries: 14. Top-K: 5._

## Per-query rankings

### radiohead-textural

> I love Radiohead's In Rainbows but want something less melancholy, more textural — instrumental preferred. Maybe something with electronic elements but organic-feeling.


**Expected vibe:** Textural, instrumental-leaning art rock or warm electronic. Aphex Twin SAW1 should rank well; Mogwai plausibly. Radiohead itself should NOT appear (the user already named it). Anything vocal-driven or aggressive is wrong.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Radiohead — _In Rainbows_ (2007) · 0.663 | Radiohead — _In Rainbows_ (2007) · 0.609 | Radiohead — _In Rainbows_ (2007) · 0.626 |
| 2 | Aphex Twin — _Selected Ambient Works 85-92_ (1992) · 0.501 | Radiohead — _Kid A_ (2000) · 0.418 | Radiohead — _Kid A_ (2000) · 0.522 |
| 3 | Björk — _Vespertine_ (2001) · 0.492 | Radiohead — _The Bends_ (1995) · 0.412 | Radiohead — _OK Computer_ (1997) · 0.481 |
| 4 | Radiohead — _OK Computer_ (1997) · 0.440 | Radiohead — _OK Computer_ (1997) · 0.400 | Radiohead — _The Bends_ (1995) · 0.469 |
| 5 | Radiohead — _Kid A_ (2000) · 0.437 | Aphex Twin — _Selected Ambient Works 85-92_ (1992) · 0.395 | Aphex Twin — _Selected Ambient Works 85-92_ (1992) · 0.428 |

### saturday-jazz

> Saturday morning coffee, jazzy but modern, nothing harsh. Something I can read the paper to.


**Expected vibe:** Modern jazz or jazz-adjacent records that read as gentle. Kind of Blue is the obvious classic answer; To Pimp a Butterfly has the jazz lineage but it's NOT Saturday-morning music — would be a tell of weak retrieval if it ranks 1.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Miles Davis — _Kind of Blue_ (1959) · 0.346 | Boston — _Boston_ (1976) · 0.395 | Steely Dan — _Aja_ (1977) · 0.339 |
| 2 | Steely Dan — _Pretzel Logic_ (1974) · 0.309 | Deep Purple — _Machine Head_ (1972) · 0.379 | Miles Davis — _Kind of Blue_ (1959) · 0.300 |
| 3 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.287 | Sly & The Family Stone — _Fresh_ (1973) · 0.371 | Steely Dan — _The Royal Scam_ (1976) · 0.283 |
| 4 | Steely Dan — _The Royal Scam_ (1976) · 0.281 | Steely Dan — _Aja_ (1977) · 0.367 | Sly & The Family Stone — _Fresh_ (1973) · 0.281 |
| 5 | Sly & The Family Stone — _Fresh_ (1973) · 0.276 | The Beatles — _Sgt. Pepper's Lonely Hearts Club Band_ (1967) · 0.360 | Steely Dan — _Pretzel Logic_ (1974) · 0.258 |

### late-night-electronic

> Late-night drive music, hypnotic and electronic, no lyrics if possible.


**Expected vibe:** Aphex Twin SAW1 is the bullseye. Daft Punk Discovery is electronic but too poppy/danceable for late-night-drive — should not be top-1. Anything vocal-heavy is wrong.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Björk — _Vespertine_ (2001) · 0.510 | The Beatles — _Revolver_ (1966) · 0.459 | Aphex Twin — _Selected Ambient Works 85-92_ (1992) · 0.444 |
| 2 | Aphex Twin — _Selected Ambient Works 85-92_ (1992) · 0.493 | Deep Purple — _Machine Head_ (1972) · 0.459 | Björk — _Vespertine_ (2001) · 0.370 |
| 3 | Jimi Hendrix — _Are You Experienced_ (1967) · 0.456 | Led Zeppelin — _Houses of the Holy_ (1973) · 0.458 | Jimi Hendrix — _Electric Ladyland_ (1968) · 0.355 |
| 4 | Jimi Hendrix — _Axis: Bold as Love_ (1967) · 0.431 | Daft Punk — _Discovery_ (2001) · 0.444 | Daft Punk — _Discovery_ (2001) · 0.347 |
| 5 | Jimi Hendrix — _Electric Ladyland_ (1968) · 0.430 | Radiohead — _Kid A_ (2000) · 0.443 | Mogwai — _Young Team_ (1997) · 0.345 |

### deep-cut-soul

> I want classic 70s soul but the deeper cuts. I already know Marvin Gaye and Stevie Wonder.


**Expected vibe:** What's Going On is a Marvin Gaye record so it should NOT appear (the user named the artist). The corpus is thin on soul outside Marvin Gaye, so the model may need to reach — interesting test of how it handles a cold-start genre. Reasonable to surface adjacent picks (jazz, funk).


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | The Beatles — _Rubber Soul_ (1965) · 0.469 | Marvin Gaye — _What's Going On_ (1971) · 0.423 | Marvin Gaye — _What's Going On_ (1971) · 0.437 |
| 2 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.447 | Sufjan Stevens — _Carrie & Lowell_ (2015) · 0.376 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.379 |
| 3 | Marvin Gaye — _What's Going On_ (1971) · 0.446 | The Beatles — _Rubber Soul_ (1965) · 0.372 | Sly & The Family Stone — _Fresh_ (1973) · 0.366 |
| 4 | Boston — _Boston_ (1976) · 0.422 | Miles Davis — _Kind of Blue_ (1959) · 0.365 | Sly & The Family Stone — _Stand!_ (1969) · 0.355 |
| 5 | Deep Purple — _Machine Head_ (1972) · 0.419 | Funkadelic — _One Nation Under a Groove_ (1978) · 0.365 | Aerosmith — _Toys in the Attic_ (1975) · 0.346 |

### surprise-me

> Surprise me — something completely outside what I'd usually pick. I usually listen to mainstream rock.


**Expected vibe:** Discovery mode. Less canonical picks should rank higher. Anything from the corpus could be defensible here; the test is whether the model surfaces things genuinely unlike mainstream rock vs. just picking the closest rock-adjacent record.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Radiohead — _In Rainbows_ (2007) · 0.513 | The Rolling Stones — _Exile on Main St._ (1972) · 0.401 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.329 |
| 2 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.476 | Daft Punk — _Discovery_ (2001) · 0.396 | Aerosmith — _Rocks_ (1976) · 0.325 |
| 3 | Radiohead — _The Bends_ (1995) · 0.467 | Aerosmith — _Rocks_ (1976) · 0.390 | Radiohead — _In Rainbows_ (2007) · 0.321 |
| 4 | The Rolling Stones — _Exile on Main St._ (1972) · 0.451 | Deep Purple — _Machine Head_ (1972) · 0.383 | Radiohead — _The Bends_ (1995) · 0.311 |
| 5 | Aerosmith — _Rocks_ (1976) · 0.448 | Boston — _Boston_ (1976) · 0.368 | The Rolling Stones — _Exile on Main St._ (1972) · 0.310 |

### heavy-and-slow

> Heavy and slow, doom-y but not metal exactly.


**Expected vibe:** Black Sabbath Paranoid is metal so it's a borderline match — the 'not metal exactly' qualifier should ideally pull it down a slot. Mogwai's Young Team has the loud-slow thing without being metal. A model that ignores the 'not metal' qualifier and picks Sabbath at #1 has weaker semantic understanding.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Black Sabbath — _Paranoid_ (1970) · 0.414 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.432 | Black Sabbath — _Paranoid_ (1970) · 0.372 |
| 2 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.377 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.419 | Mogwai — _Young Team_ (1997) · 0.356 |
| 3 | Aerosmith — _Rocks_ (1976) · 0.372 | Black Sabbath — _Paranoid_ (1970) · 0.409 | Funkadelic — _Maggot Brain_ (1971) · 0.348 |
| 4 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.367 | Radiohead — _Kid A_ (2000) · 0.407 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.336 |
| 5 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.349 | Aerosmith — _Rocks_ (1976) · 0.399 | Deep Purple — _Machine Head_ (1972) · 0.322 |

### electric-ladyland-maggot-brain

> Stuff that lives between Electric Ladyland and Maggot Brain.


**Expected vibe:** Psychedelic, guitar-forward, expansive jams; heavy but not metal; emphasis on solos, atmosphere, and long-form grooves rather than tight pop song forms. Lean toward psychedelic funk, acid rock, and extended jam pieces rather than concise radio singles.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Funkadelic — _Maggot Brain_ (1971) · 0.408 | Funkadelic — _Maggot Brain_ (1971) · 0.418 | Funkadelic — _Maggot Brain_ (1971) · 0.569 |
| 2 | Jimi Hendrix — _Electric Ladyland_ (1968) · 0.396 | Jimi Hendrix — _Electric Ladyland_ (1968) · 0.372 | Jimi Hendrix — _Electric Ladyland_ (1968) · 0.543 |
| 3 | Radiohead — _In Rainbows_ (2007) · 0.224 | Radiohead — _OK Computer_ (1997) · 0.348 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.368 |
| 4 | Björk — _Vespertine_ (2001) · 0.215 | Deep Purple — _Machine Head_ (1972) · 0.344 | Steely Dan — _Pretzel Logic_ (1974) · 0.358 |
| 5 | Aphex Twin — _Selected Ambient Works 85-92_ (1992) · 0.211 | Radiohead — _Kid A_ (2000) · 0.341 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.354 |

### aja-produced-rock

> Rock records that are as meticulously produced as Aja but still have teeth.


**Expected vibe:** Rock or jazz-rock albums with immaculate production, sophisticated harmony, and groove, but with some grit or edge left—no completely smoothed-over adult-contemporary sheen. Think studio-obsessive bandleaders and session-player precision that still feels like a rock record.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Steely Dan — _Aja_ (1977) · 0.423 | Steely Dan — _Aja_ (1977) · 0.480 | Steely Dan — _Aja_ (1977) · 0.530 |
| 2 | Aerosmith — _Rocks_ (1976) · 0.321 | Aerosmith — _Rocks_ (1976) · 0.403 | Aerosmith — _Rocks_ (1976) · 0.462 |
| 3 | The Rolling Stones — _Sticky Fingers_ (1971) · 0.306 | Radiohead — _Kid A_ (2000) · 0.399 | Aerosmith — _Toys in the Attic_ (1975) · 0.410 |
| 4 | Boston — _Boston_ (1976) · 0.300 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.374 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.395 |
| 5 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.279 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.358 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.390 |

### funk-political-bounce

> 70s funk with the political edge of There's a Riot Goin' On and the bounce of Parliament.


**Expected vibe:** Early-to-mid 70s funk and soul with strong grooves and danceable bounce, but lyrically darker, socially aware, or politically charged. Sonically somewhere between the murky, tape-saturated feel of Riot and the bright party energy of Parliament; absolutely funk, not full-on disco.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.674 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.680 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.647 |
| 2 | Parliament — _Mothership Connection_ (1975) · 0.581 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.612 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.589 |
| 3 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.557 | Parliament — _Mothership Connection_ (1975) · 0.601 | Parliament — _Mothership Connection_ (1975) · 0.582 |
| 4 | Sly & The Family Stone — _Fresh_ (1973) · 0.545 | Funkadelic — _One Nation Under a Groove_ (1978) · 0.534 | Funkadelic — _One Nation Under a Groove_ (1978) · 0.545 |
| 5 | Funkadelic — _One Nation Under a Groove_ (1978) · 0.486 | Sly & The Family Stone — _Fresh_ (1973) · 0.530 | Sly & The Family Stone — _Fresh_ (1973) · 0.534 |

### beatles-fool-guitar-crunch

> Give me some songs that sound like the Beatles 'Fool on the Hill', but with more guitar crunch.


**Expected vibe:** Melodic, somewhat pastoral or lightly psychedelic rock/pop with rich harmony and arrangements like late-60s Beatles, but with more prominent electric guitar tone. Tuneful and song-centric rather than metal or heavy riff rock; think color and chords first, then a bit more bite.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Jimi Hendrix — _Axis: Bold as Love_ (1967) · 0.503 | The Beatles — _Revolver_ (1966) · 0.487 | The Beatles — _Rubber Soul_ (1965) · 0.492 |
| 2 | Radiohead — _The Bends_ (1995) · 0.477 | The Beatles — _Abbey Road_ (1969) · 0.443 | The Beatles — _Revolver_ (1966) · 0.480 |
| 3 | The Beatles — _Rubber Soul_ (1965) · 0.468 | The Beatles — _Rubber Soul_ (1965) · 0.437 | The Beatles — _The Beatles (White Album)_ (1968) · 0.454 |
| 4 | The Beatles — _Abbey Road_ (1969) · 0.458 | The Beatles — _Sgt. Pepper's Lonely Hearts Club Band_ (1967) · 0.436 | The Beatles — _Abbey Road_ (1969) · 0.449 |
| 5 | The Beatles — _The Beatles (White Album)_ (1968) · 0.418 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.432 | The Beatles — _Sgt. Pepper's Lonely Hearts Club Band_ (1967) · 0.432 |

### zeppelin-mature-albums

> While I grew up with Led Zeppelin's first few albums, as I've aged I've grown increasingly fond of Physical Graffiti. What are some other 'mature' albums from veteran rock bands?


**Expected vibe:** Later-career albums by rock bands who already have several records behind them, where they broaden their palette and sound more assured and varied. Sprawling or multi-style records that feel like a culmination rather than a debut burst; still rock, not a soft adult-contemporary pivot.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.722 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.650 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.676 |
| 2 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.515 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.542 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.492 |
| 3 | Led Zeppelin — _Led Zeppelin II_ (1969) · 0.491 | Led Zeppelin — _Led Zeppelin II_ (1969) · 0.527 | Led Zeppelin — _Houses of the Holy_ (1973) · 0.477 |
| 4 | Radiohead — _In Rainbows_ (2007) · 0.444 | Led Zeppelin — _Houses of the Holy_ (1973) · 0.487 | Led Zeppelin — _Led Zeppelin II_ (1969) · 0.466 |
| 5 | The Beatles — _The Beatles (White Album)_ (1968) · 0.434 | Aerosmith — _Rocks_ (1976) · 0.467 | The Rolling Stones — _Let It Bleed_ (1969) · 0.460 |

### funk-not-disco

> There's a thin line between funk and disco. Do not cross that line.


**Expected vibe:** 70s funk with strong groove, syncopation, and bass presence that may brush up against disco-adjacent sounds but never tips into straight four-on-the-floor, glossy strings, or overt disco tropes. Live-band feel prioritized over slick, mirror-ball polish.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.456 | Daft Punk — _Discovery_ (2001) · 0.480 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.414 |
| 2 | Funkadelic — _One Nation Under a Groove_ (1978) · 0.454 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.468 | Funkadelic — _One Nation Under a Groove_ (1978) · 0.402 |
| 3 | Sly & The Family Stone — _Fresh_ (1973) · 0.428 | Sly & The Family Stone — _Stand!_ (1969) · 0.460 | Funkadelic — _Maggot Brain_ (1971) · 0.359 |
| 4 | Parliament — _Mothership Connection_ (1975) · 0.381 | Funkadelic — _Maggot Brain_ (1971) · 0.453 | Sly & The Family Stone — _Fresh_ (1973) · 0.349 |
| 5 | Funkadelic — _Maggot Brain_ (1971) · 0.373 | Parliament — _Funkentelechy vs. the Placebo Syndrome_ (1977) · 0.450 | Sly & The Family Stone — _There's a Riot Goin' On_ (1971) · 0.345 |

### 90s-radiohead-between-genres

> Rock albums that, like mid-90s Radiohead, sit between grunge, britpop, and the more experimental 2000s art-rock thing—'of their time' but clearly pushing beyond it.


**Expected vibe:** Mid-to-late-90s records that share DNA with The Bends and OK Computer: melodic, guitar-driven, harmonically interesting, and texturally curious. Still recognizably alt-rock or britpop-era, but with arrangement or production moves that point forward toward 2000s experimental/art-rock rather than backward to pure grunge or straight britpop homage.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Radiohead — _The Bends_ (1995) · 0.528 | Radiohead — _The Bends_ (1995) · 0.591 | Radiohead — _Kid A_ (2000) · 0.548 |
| 2 | Radiohead — _In Rainbows_ (2007) · 0.503 | Radiohead — _OK Computer_ (1997) · 0.558 | Radiohead — _OK Computer_ (1997) · 0.546 |
| 3 | Radiohead — _Kid A_ (2000) · 0.489 | Radiohead — _Kid A_ (2000) · 0.551 | Radiohead — _The Bends_ (1995) · 0.545 |
| 4 | Led Zeppelin — _Physical Graffiti_ (1975) · 0.483 | Aerosmith — _Rocks_ (1976) · 0.537 | Radiohead — _In Rainbows_ (2007) · 0.511 |
| 5 | Radiohead — _OK Computer_ (1997) · 0.481 | Deep Purple — _Deep Purple in Rock_ (1970) · 0.502 | Mogwai — _Young Team_ (1997) · 0.462 |

### holy-trinity-bandleaders

> If my two holy trinities are Miles/Zappa/Mingus for bandleaders and Hendrix/Coltrane/Marley for transcendently honest soul, who else belongs on my Mount Rushmore?


**Expected vibe:** Artists with strong auteur or bandleader energy and/or spiritually intense, emotionally honest work—people who reshape bands and genres the way Miles, Zappa, Mingus, Hendrix, Coltrane, and Marley did. Not just influential or popular, but visionary figures with distinct compositional voices or deep spiritual/emotional weight in their music.


| Rank | HF all-MiniLM-L6-v2 | Voyage voyage-3-lite | OpenAI text-embedding-3-small |
|---|---|---|---|
| 1 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.291 | Led Zeppelin — _Houses of the Holy_ (1973) · 0.381 | Jimi Hendrix — _Band of Gypsys_ (1970) · 0.358 |
| 2 | Radiohead — _In Rainbows_ (2007) · 0.266 | Deep Purple — _Machine Head_ (1972) · 0.351 | Jimi Hendrix — _Axis: Bold as Love_ (1967) · 0.352 |
| 3 | Jimi Hendrix — _Electric Ladyland_ (1968) · 0.263 | Led Zeppelin — _Led Zeppelin II_ (1969) · 0.340 | Miles Davis — _Kind of Blue_ (1959) · 0.329 |
| 4 | Led Zeppelin — _Led Zeppelin II_ (1969) · 0.259 | Boston — _Boston_ (1976) · 0.320 | Led Zeppelin — _Led Zeppelin IV_ (1971) · 0.323 |
| 5 | Sly & The Family Stone — _Stand!_ (1969) · 0.232 | The Beatles — _Abbey Road_ (1969) · 0.318 | Jimi Hendrix — _Are You Experienced_ (1967) · 0.317 |

## Cost and timing summary

| Provider | Model | Corpus tokens | Query tokens | Corpus time | Query time | Est. cost (this run) |
|---|---|---|---|---|---|---|
| HF all-MiniLM-L6-v2 | `sentence-transformers/all-MiniLM-L6-v2` | 3278 | 364 | 0.52s | 0.11s | free tier |
| Voyage voyage-3-lite | `voyage-3-lite` | 3263 | 327 | 0.43s | 0.47s | $0.000072 |
| OpenAI text-embedding-3-small | `text-embedding-3-small` | 3188 | 336 | 2.21s | 0.76s | $0.000070 |

_Token counts for HF are approximated as `chars / 4`. OpenAI and Voyage report exact usage._

## How to read this

Read each query's table left-to-right at rank 1: do all providers pick something defensible? Then scan to rank 5: which provider's tail still looks coherent? The `expected_vibe` row above each table is your gut-check ground truth.

Cost is essentially noise at this scale. Pick on quality. If two providers tie, pick the cheaper one.
