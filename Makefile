.PHONY: all translation clean

all:
	latexmk -pdf -interaction=nonstopmode -halt-on-error paper.tex

translation:
	latexmk -xelatex -interaction=nonstopmode -halt-on-error translation-template.tex

clean:
	latexmk -C paper.tex
	latexmk -C translation-template.tex
