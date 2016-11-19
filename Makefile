all: build

build:
	go build

test: build
	./commit-poster -src ~/dev/betalo/await \
	                -mask mask.png \
	                -out poster.png \
	                -width 5 \
	                -height 7
