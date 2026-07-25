# Paxos Made Simple

LaTeX project for Leslie Lamport's *Paxos Made Simple*.

The unchanged source paper is stored at `source/paxos-simple.pdf`.

## Translation build

```sh
make
```

The default build compiles the Chinese translation from `paper.tex` and
outputs `paper.pdf`.

## Exact original build

```sh
make original
```

`original.tex` embeds every original PDF page without reflowing text and
outputs `original.pdf`, preserving the original content, typography, and
pagination.

## Translation workspace

The editable, Chinese-compatible single-column root document is `paper.tex`.
Its body is split across `sections/*.tex` and follows the original section
structure. Every section declares `paper.tex` as its TeX root so editor
auto-builds triggered on save compile the Chinese translation.

```sh
make translation
```
