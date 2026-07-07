# Paxos Made Simple

LaTeX project for Leslie Lamport's *Paxos Made Simple*.

The unchanged source paper is stored at `source/paxos-simple.pdf`.

## Exact original build

```sh
make
```

The default `paper.tex` embeds every original PDF page without reflowing text,
so `paper.pdf` preserves the original content, typography, and pagination.

## Translation workspace

The editable, Chinese-compatible single-column template is
`translation-template.tex`. Its body is split across `sections/*.tex` and
follows the original section structure.

```sh
make translation
```
