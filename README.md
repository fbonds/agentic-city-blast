# agentic-city-blast

A fork of [mrf/agentic-city](https://github.com/mrf/agentic-city) by Mark Ferree.

**This is a thought experiment, not a project.** Nothing in this fork is implemented yet — the code is bit-for-bit identical to upstream. The idea lives in [encoding-redesign.md](encoding-redesign.md); the implementation may or may not follow.

## Why this exists

The upstream project is remarkable. It turns a codebase into a living isometric city — files become buildings, directories become districts, AI coding sessions appear as UFOs flying overhead, and everything updates in real time over WebSockets. It is a genuinely impressive piece of work, generously published under MIT, and I would not have thought to build it.

I came across it through a coworker. While chatting with Claude Code about the premise, one thing kept circling: the dominant visual channel — building height — is driven by file size. File size is the cheapest signal to compute, but it may not be the most useful one for the use case the city is pitched at (orchestrating AI agents). And so: *this is awesome — I wonder what it would be like if the metric were something else though.*

The sketch:

- **Height** would encode **blast radius** — how many files transitively depend on this one.
- **Color** would encode **churn** — how often the file has changed recently.

That's it. The reasoning, the code it would touch, and the open decisions are all in [encoding-redesign.md](encoding-redesign.md).

## Honest caveats

I genuinely do not know if this will work, or if it will be of value to anyone beyond satisfying my own curiosity. The dependency graph this would build on is import-extraction-based, and upstream itself flags it as approximate, so blast-radius numbers would inherit that uncertainty. It is entirely possible the result reads worse than the original.

This is also not a critique of upstream, which I think is excellent. The reason it lives in a fork rather than a PR is that the encoding swap conflicts with upstream's stated thesis ("file sizes determine building heights"), and asking Mark to take that on would not be respectful of his vision. Any dependency-analyzer improvements that fall out of this are vision-neutral and I would be happy to offer them back.

## Running it

Unchanged from upstream. See [DESIGN.md](DESIGN.md) for architecture and keyboard bindings:

```bash
make dev          # frontend dev server (http://localhost:5173)
make run          # Go backend (http://localhost:8080)
```

## License

MIT, retaining Mark Ferree's original copyright. See [LICENSE](LICENSE).
