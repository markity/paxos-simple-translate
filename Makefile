.PHONY: all original translation clean

all:
	latexmk -xelatex -interaction=nonstopmode -halt-on-error paper.tex

original:
	latexmk -jobname=original -pdf -interaction=nonstopmode -halt-on-error original.tex

translation:
	latexmk -xelatex -interaction=nonstopmode -halt-on-error paper.tex

clean:
	latexmk -C paper.tex
	latexmk -jobname=original -C original.tex
