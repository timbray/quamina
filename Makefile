# The only purpose of this makefile is to run code_gen/code_gen, which will rebuild the case_folding.go file if
# it is more than three months out of date
casefold:
	@ cd code_gen && go build && cd ..
	@ code_gen/code_gen
