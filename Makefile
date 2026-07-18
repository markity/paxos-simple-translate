.PHONY: all original translation clean

all:
	latexmk -jobname=paper-trans -xelatex -interaction=nonstopmode -halt-on-error translation-template.tex

original:
	latexmk -pdf -interaction=nonstopmode -halt-on-error paper.tex

translation:
	latexmk -jobname=paper-trans -xelatex -interaction=nonstopmode -halt-on-error translation-template.tex

clean:
	latexmk -C paper.tex
	latexmk -jobname=paper-trans -C translation-template.tex
